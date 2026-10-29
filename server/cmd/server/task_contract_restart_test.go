package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/multica-ai/multica/server/internal/analytics"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/realtime"
)

type w54RestartInput struct {
	Token, Workspace, Output string
	Paths                    []string
}

// The child only reconstructs the real router and reads existing synthetic
// records. It must run before TestMain's normal destructive fixture cleanup.
func w54RestartReaderProcess() (int, bool) {
	inputPath := os.Getenv("W54_RESTART_READER_INPUT")
	if inputPath == "" {
		return 0, false
	}
	err := func() error {
		u, err := url.Parse(os.Getenv("DATABASE_URL"))
		if err != nil {
			return err
		}
		if (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") || !strings.HasPrefix(u.Path, "/multica_wush54_") {
			return fmt.Errorf("restart reader requires isolated W54 database")
		}
		raw, err := os.ReadFile(inputPath)
		if err != nil {
			return err
		}
		var in w54RestartInput
		if err = json.Unmarshal(raw, &in); err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
		if err != nil {
			return err
		}
		defer pool.Close()
		if err = pool.Ping(ctx); err != nil {
			return err
		}
		server := httptest.NewServer(NewRouter(pool, realtime.NewHub(), events.New(), analytics.NoopClient{}, nil))
		defer server.Close()
		out := map[string]json.RawMessage{}
		for _, path := range in.Paths {
			req, err := http.NewRequestWithContext(ctx, "GET", server.URL+path, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+in.Token)
			req.Header.Set("X-Workspace-ID", in.Workspace)
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			body, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				return err
			}
			if response.StatusCode != 200 {
				return fmt.Errorf("restart read %s status=%d", path, response.StatusCode)
			}
			if !json.Valid(body) {
				return fmt.Errorf("restart read returned non-JSON")
			}
			out[path] = body
		}
		raw, err = json.Marshal(out)
		if err != nil {
			return err
		}
		return os.WriteFile(in.Output, raw, 0600)
	}()
	if err != nil {
		fmt.Fprintln(os.Stderr, "W54 restart reader:", err)
		return 1, true
	}
	return 0, true
}

func w54RestartRead(t *testing.T, paths ...string) map[string]any {
	t.Helper()
	root := t.TempDir()
	input, output := filepath.Join(root, "input.json"), filepath.Join(root, "output.json")
	in := w54RestartInput{Token: testToken, Workspace: testWorkspaceID, Output: output, Paths: paths}
	if err := os.WriteFile(input, w54JSON(t, in), 0600); err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "-test.run=^$")
	cmd.Env = append(os.Environ(), "W54_RESTART_READER_INPUT="+input)
	log, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("restart reader process failed: %v %s", err, log)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var results map[string]any
	if err = json.Unmarshal(raw, &results); err != nil {
		t.Fatal(err)
	}
	t.Logf("W54_RESTART child_pid=%d exit=0 paths=%d new_pool=true", cmd.ProcessState.Pid(), len(paths))
	return results
}

func TestW54Contract_WS02_TC18_RestartReaderBaseline(t *testing.T) {
	f := w54New(t)
	f.unchangedIssue(t)
	path := "/api/issues/" + f.issueID
	before := w54Body(t, authRequest(t, "GET", path, nil), 200)
	if got := w54RestartRead(t, path); !reflect.DeepEqual(got[path], before) {
		t.Error("new OS process could not read original synthetic issue")
	}
}

func TestW54Contract_WS02_TC18_RestartPreservesAllContractFacts(t *testing.T) {
	f := w54New(t)
	f.unchangedIssue(t)
	f.publish(t, 0, 201)
	h := f.handoff(t, 1, "restart-receipt", 201)
	w54Body(t, authRequest(t, "POST", f.path()+"/handoffs/"+w54ID(t, h, "handoff_id")+"/accept", map[string]any{"current_version": 1}), 200)
	f.snapshot["goal"] = "Approved v2 after first receipt"
	f.publish(t, 1, 200)
	refusal := f.handoff(t, 1, "restart-stale", 409)
	paths := []string{f.path(), f.path() + "/versions", f.path() + "/versions/1", f.path() + "/handoffs", f.path() + "/handoffs/" + w54ID(t, h, "handoff_id"), f.path() + "/admissions", f.path() + "/admissions/" + w54ID(t, refusal, "admission_id")}
	before := map[string]any{}
	for _, path := range paths {
		before[path] = w54Body(t, authRequest(t, "GET", path, nil), 200)
	}
	if after := w54RestartRead(t, paths...); !reflect.DeepEqual(before, after) {
		t.Error("restart lost current/history/handoff/receipt/rejection facts")
	}
}
