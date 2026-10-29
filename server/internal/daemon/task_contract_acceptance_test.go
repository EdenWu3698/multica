package daemon

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Exercise the real runTask path; only the remote server and agent executable
// are doubles. A successful-start control proves the process/file probes work.
func TestW54Runtime_TC15_RefusalHasNoLocalExecution(t *testing.T) {
	for _, tc := range []struct {
		code   string
		status int
	}{
		{"contract_stale", 409}, {"source_changed", 409}, {"input_version_mismatch", 409},
		{"contract_out_of_scope", 409}, {"contract_protocol_unsupported", 409},
		{"blocking_question", 422}, {"permission_denied", 403}, {"validation_unavailable", 503},
		{"positive_probe_control", 200},
	} {
		t.Run(tc.code, func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, "process-started")
			source := filepath.Join(root, "synthetic-source.txt")
			original := []byte("synthetic source before admission\n")
			if err := os.WriteFile(source, original, 0600); err != nil {
				t.Fatal(err)
			}
			fake := filepath.Join(root, "fake-agent")
			writeTestExecutable(t, fake, []byte(`#!/bin/sh
printf started > "$W54_PROCESS_MARKER"
printf changed > "$W54_SOURCE"
IFS= read -r _
printf '%s\n' '{"type":"system","session_id":"synthetic-session"}'
printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"session_id":"synthetic-session","result":"synthetic evidence"}'
`))
			if err := os.Chmod(fake, 0700); err != nil {
				t.Fatal(err)
			}
			var starts, postStartRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/start") {
					starts.Add(1)
					w.WriteHeader(tc.status)
					fmt.Fprintf(w, `{"error":"synthetic admission result","code":%q}`, tc.code)
					return
				}
				if starts.Load() > 0 {
					postStartRequests.Add(1)
				}
				io.WriteString(w, `{}`)
			}))
			t.Cleanup(server.Close)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			d := &Daemon{client: NewClient(server.URL), logger: logger,
				workspaces: make(map[string]*workspaceState), activeEnvRoots: make(map[string]int),
				runtimeIndex: map[string]Runtime{"rt-w54": {ID: "rt-w54", Provider: "claude"}},
				cfg: Config{WorkspacesRoot: filepath.Join(root, "workspaces"), AgentTimeout: 5 * time.Second,
					ServerBaseURL: server.URL, Agents: map[string]AgentEntry{"claude": {Path: fake}}},
			}
			task := Task{ID: "task-w54", WorkspaceID: "workspace-w54", RuntimeID: "rt-w54", IssueID: "issue-w54", AgentID: "agent-w54",
				AuthToken: "mat_w54_synthetic_test_only",
				Agent:     &AgentData{ID: "agent-w54", Name: "Synthetic agent", CustomEnv: map[string]string{"W54_PROCESS_MARKER": marker, "W54_SOURCE": source}},
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := d.runTask(ctx, task, "claude", 0, logger)
			if starts.Load() != 1 {
				t.Fatalf("start requests=%d, want 1 (no legacy replay)", starts.Load())
			}
			_, markerErr := os.Stat(marker)
			after, readErr := os.ReadFile(source)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if tc.status == 200 {
				if err != nil || markerErr != nil || string(after) != "changed" {
					t.Fatalf("probe control failed: run=%v marker=%v source=%q", err, markerErr, after)
				}
				return
			}
			if err == nil {
				t.Error("refused start returned success")
			}
			if !os.IsNotExist(markerErr) {
				t.Errorf("N: business process was started: %v", markerErr)
			}
			if sha256.Sum256(after) != sha256.Sum256(original) {
				t.Error("N: source changed before admission")
			}
			if postStartRequests.Load() != 0 {
				t.Errorf("N: %d requests after refused start", postStartRequests.Load())
			}
		})
	}
}
