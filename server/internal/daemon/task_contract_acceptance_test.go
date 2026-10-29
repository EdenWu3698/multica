package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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
		code         string
		status       int
		resume       bool
		claimed      bool
		dropResponse bool
	}{
		{code: "contract_stale", status: 409}, {code: "source_changed", status: 409}, {code: "input_version_mismatch", status: 409},
		{code: "contract_out_of_scope", status: 409}, {code: "contract_protocol_unsupported", status: 409},
		{code: "blocking_question", status: 422}, {code: "permission_denied", status: 403}, {code: "validation_unavailable", status: 503},
		{code: "positive_probe_control", status: 200},
		{code: "resumed_contract_stale", status: 409, resume: true},
		{code: "resumed_validation_unavailable", status: 503, resume: true},
		{code: "claimed_contract_stale", status: 409, claimed: true},
		{code: "resumed_claimed_contract_stale", status: 409, resume: true, claimed: true},
		{code: "authentication_revoked", status: 401},
		{code: "contract_missing", status: 404},
		{code: "contract_incomplete", status: 422},
		{code: "unknown_rejection", status: 599},
		{code: "response_lost_result_unknown", dropResponse: true},
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
			var starts, preStartRequests, postStartRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/start") {
					starts.Add(1)
					if tc.claimed {
						var input map[string]any
						if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
							t.Errorf("decode actual start request: %v", err)
						} else if input["runtime_id"] != "rt-w54" || input["dispatched_at"] != "2026-10-29T00:00:00.123456Z" {
							t.Errorf("claim binding missing from actual request: %v", input)
						}
					}
					if tc.dropResponse {
						conn, _, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Errorf("fault injection failed: %v", err)
							return
						}
						conn.Close()
						return
					}
					w.WriteHeader(tc.status)
					code := strings.TrimPrefix(strings.TrimPrefix(tc.code, "resumed_"), "claimed_")
					fmt.Fprintf(w, `{"error":"synthetic admission result","code":%q}`, code)
					return
				}
				if starts.Load() > 0 {
					postStartRequests.Add(1)
				} else {
					preStartRequests.Add(1)
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
			if tc.resume {
				task.PriorSessionID = "synthetic-old-session"
				task.PriorWorkDir = root
			}
			if tc.claimed {
				task.DispatchedAt = "2026-10-29T00:00:00.123456Z"
				task.StartClaimSupported = true
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
			if tc.dropResponse {
				t.Log("Remote acceptance is UNKNOWN; only local execution and request counts are asserted. Same-key reconciliation requires the approved wire supplement.")
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
			if preStartRequests.Load() != 0 {
				t.Errorf("N: %d resource requests before admission", preStartRequests.Load())
			}
		})
	}
}
