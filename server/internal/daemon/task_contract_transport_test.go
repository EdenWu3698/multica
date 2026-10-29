package daemon

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func w54HasCapability(header string) bool {
	for _, item := range strings.Split(header, ",") {
		if strings.TrimSpace(item) == "task-contract-v1" {
			return true
		}
	}
	return false
}

func TestW54Runtime_WS03_TC13_HTTPClaimCapability(t *testing.T) {
	for _, mode := range []string{"single", "batch", "legacy"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !w54HasCapability(r.Header.Get("X-Client-Capabilities")) {
					t.Error("actual HTTP claim omitted task-contract-v1 capability")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if _, present := body["contract_admission"]; present {
					t.Error("claim must not provide a client-controlled admission")
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"task":null,"tasks":[]}`)
			}))
			defer srv.Close()
			client := NewClient(srv.URL)
			defer client.CloseIdleConnections()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var err error
			switch mode {
			case "single":
				_, err = client.ClaimTask(ctx, uuid.NewString())
			case "batch":
				_, err = client.ClaimTasks(ctx, uuid.NewString(), []string{uuid.NewString()}, 1)
			case "legacy":
				_, err = client.claimTasksLegacy(ctx, []string{uuid.NewString()}, 1)
			}
			if err != nil || calls.Load() != 1 {
				t.Fatalf("claim transport err=%v calls=%d", err, calls.Load())
			}
		})
	}
}

func TestW54Runtime_WS03_TC13_RegistrationCapability(t *testing.T) {
	runtimeID, workspaceID := uuid.NewString(), uuid.NewString()
	var registered atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/daemon/register" {
			http.NotFound(w, r)
			return
		}
		registered.Add(1)
		if !w54HasCapability(r.Header.Get("X-Client-Capabilities")) {
			t.Error("actual runtime registration omitted task-contract-v1 capability")
		}
		json.NewEncoder(w).Encode(map[string]any{"runtimes": []any{map[string]any{"id": runtimeID, "provider": "claude"}}})
	}))
	defer srv.Close()
	d := New(Config{ServerBaseURL: srv.URL, DaemonID: uuid.NewString(), WorkspacesRoot: t.TempDir()}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer d.client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := d.registerBuiltinRuntimesForWorkspaceLocked(ctx, workspaceID, []map[string]string{{"name": "Synthetic", "type": "claude", "version": "1", "status": "online"}})
	if err != nil || registered.Load() != 1 {
		t.Fatalf("register err=%v calls=%d", err, registered.Load())
	}
}

func TestW54Runtime_WS03_TC13_WSHandshakeCapability(t *testing.T) {
	upgrader := websocket.Upgrader{}
	var connected atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !w54HasCapability(r.Header.Get("X-Client-Capabilities")) {
			t.Error("actual WS handshake omitted task-contract-v1 capability")
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		connected.Add(1)
		conn.Close()
	}))
	defer srv.Close()
	d := New(Config{ServerBaseURL: srv.URL, HeartbeatInterval: time.Hour, WorkspacesRoot: t.TempDir()}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer d.client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// The peer closes immediately after the actual upgrade. This call joins its
	// writer/heartbeat/read workers before returning; no reconnect loop is run.
	d.runTaskWakeupConnection(ctx, []string{uuid.NewString()}, make(chan taskWakeup, 2), make(chan struct{}))
	if connected.Load() != 1 {
		t.Fatalf("WS upgrade count=%d", connected.Load())
	}
}

func TestW54Runtime_WS03_TC13_DispatchGenerationOnce(t *testing.T) {
	for _, mode := range []string{"same_batch", "same_generation_replay", "new_generation_control"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, "executions")
			fake := filepath.Join(root, "fake-agent")
			writeTestExecutable(t, fake, []byte(`#!/bin/sh
printf x >> "$W54_PROCESS_MARKER"
IFS= read -r _
printf '%s\n' '{"type":"system","session_id":"synthetic-session"}'
printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"session_id":"synthetic-session","result":"synthetic evidence"}'
`))
			if err := os.Chmod(fake, 0700); err != nil {
				t.Fatal(err)
			}
			task := Task{ID: uuid.NewString(), AgentID: uuid.NewString(), RuntimeID: uuid.NewString(), IssueID: uuid.NewString(), WorkspaceID: uuid.NewString(), AuthToken: "mat_w54_synthetic_test_only", StartClaimSupported: true, DispatchedAt: "2026-10-29T00:00:02.123456Z"}
			task.Agent = &AgentData{ID: task.AgentID, Name: "Synthetic", CustomEnv: map[string]string{"W54_PROCESS_MARKER": marker}}
			_, admission, receipt := w54ProtectedTask(t, task)
			raw, err := json.Marshal(task)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]any
			if err = json.Unmarshal(raw, &wire); err != nil {
				t.Fatal(err)
			}
			wire["contract_admission"], wire["contract_required"] = admission, true
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			pollCtx, stopPolling := context.WithCancel(ctx)
			defer stopPolling()
			var claims, starts, completed, failures atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/api/daemon/tasks/claim":
					n := claims.Add(1)
					if n == 1 && mode == "same_batch" {
						json.NewEncoder(w).Encode(map[string]any{"tasks": []any{wire, wire}})
					} else if n == 1 || (n == 2 && mode != "same_batch") {
						if n == 2 && mode == "new_generation_control" {
							wire["dispatched_at"] = "2026-10-29T00:00:04.123456Z"
							admission["claim_generation"] = wire["dispatched_at"]
						}
						json.NewEncoder(w).Encode(map[string]any{"tasks": []any{wire}})
					} else {
						io.WriteString(w, `{"tasks":[]}`)
						stopPolling()
					}
				case strings.HasSuffix(r.URL.Path, "/start"):
					starts.Add(1)
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					boundReceipt := make(map[string]any, len(receipt))
					for k, v := range receipt {
						boundReceipt[k] = v
					}
					boundReceipt["claim_generation"] = body["dispatched_at"]
					json.NewEncoder(w).Encode(map[string]any{"contract_admission": map[string]any{"status": "accepted", "receipt": boundReceipt}})
				case strings.HasSuffix(r.URL.Path, "/complete"):
					completed.Add(1)
					io.WriteString(w, `{}`)
				case strings.HasSuffix(r.URL.Path, "/fail"):
					failures.Add(1)
					io.WriteString(w, `{}`)
				default:
					io.WriteString(w, `{}`)
				}
			}))
			defer srv.Close()
			slots := 1
			if mode == "same_batch" {
				slots = 2
			}
			d := New(Config{ServerBaseURL: srv.URL, DaemonID: uuid.NewString(), WorkspacesRoot: filepath.Join(root, "workspaces"), MaxConcurrentTasks: slots, PollInterval: time.Hour, AgentTimeout: 5 * time.Second, Agents: map[string]AgentEntry{"claude": {Path: fake}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			defer d.client.CloseIdleConnections()
			d.runtimeIndex[task.RuntimeID] = Runtime{ID: task.RuntimeID, Provider: "claude"}
			d.workspaces[task.WorkspaceID] = &workspaceState{workspaceID: task.WorkspaceID, runtimeIDs: []string{task.RuntimeID}}
			d.cancelPollInterval = time.Hour
			d.runner = taskRunnerFunc(d.runTask)
			// Use real in-process execenv preparation. New's production helper
			// command would invoke this test executable as if it were the CLI.
			d.executionEnvironmentCommand = nil
			var workers sync.WaitGroup
			d.runBatchPoller(pollCtx, ctx, newTaskSlotSemaphore(slots), make(chan struct{}, 1), &workers)
			workers.Wait()
			t.Logf("W54_DISPATCH_COUNTERS claims=%d starts=%d completed=%d failures=%d", claims.Load(), starts.Load(), completed.Load(), failures.Load())
			if ctx.Err() != nil {
				t.Fatalf("scheduler did not finish: %v", ctx.Err())
			}
			data, readErr := os.ReadFile(marker)
			want := 1
			if mode == "new_generation_control" {
				want = 2
			}
			if readErr != nil || len(data) != want {
				t.Errorf("actual scheduler process executions=%d want=%d err=%v", len(data), want, readErr)
			}
			if completed.Load() != int32(want) || failures.Load() != 0 {
				t.Errorf("terminal effects complete=%d fail=%d want=%d/0", completed.Load(), failures.Load(), want)
			}
			t.Logf("W54_DISPATCH mode=%s claims=%d starts=%d executions=%d completed=%d failures=%d", mode, claims.Load(), starts.Load(), len(data), completed.Load(), failures.Load())
		})
	}
}
