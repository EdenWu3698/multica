package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/testutil"
)

func w54BridgeBinary(t *testing.T) string {
	t.Helper()
	compiler := os.Getenv("W54_GO")
	if compiler == "" {
		var err error
		compiler, err = exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
	}
	binary := filepath.Join(t.TempDir(), "w54-daemon.test")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, compiler, "test", "-c", "./internal/daemon", "-o", binary)
	command.Dir = filepath.Join("..", "..")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bridge build failed: %v %s", err, output)
	}
	return binary
}

func w54RunRealDaemon(t *testing.T, binary string, f w54RuntimeFixture, task map[string]any) map[string]any {
	t.Helper()
	root := t.TempDir()
	source, marker, executable := filepath.Join(root, "source.txt"), filepath.Join(root, "executions"), filepath.Join(root, "fake-agent")
	if err := os.WriteFile(source, []byte("synthetic source before admission\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte(`#!/bin/sh
printf x >> "$W54_PROCESS_MARKER"
printf changed > "$W54_SOURCE"
IFS= read -r _
printf '%s\n' '{"type":"system","session_id":"synthetic-session"}'
printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"session_id":"synthetic-session","result":"synthetic evidence"}'
`), 0700); err != nil {
		t.Fatal(err)
	}
	input, output := filepath.Join(root, "input.json"), filepath.Join(root, "output.json")
	payload := map[string]any{"URL": testServer.URL, "Token": f.token, "Root": root, "Executable": executable, "Source": source, "Marker": marker, "Output": output, "Task": task}
	if err := os.WriteFile(input, w54JSON(t, payload), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "-test.run=^$")
	command.Env = append(os.Environ(), "W54_DAEMON_BRIDGE_INPUT="+input)
	if log, err := command.CombinedOutput(); err != nil {
		t.Fatalf("real daemon bridge failed: %v %s", err, log)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	t.Logf("W54_N_BRIDGE child_pid=%d executions=%v before=%v after=%v rejected=%t", command.ProcessState.Pid(), result["executions"], result["before_sha256"], result["after_sha256"], result["run_error"] != "")
	return result
}

func TestW54Contract_WS04_TC15_AppAPIToActualProcess(t *testing.T) {
	binary := w54BridgeBinary(t)
	for _, mode := range []string{"ordinary_control", "protected_control", "source_changed_after_claim", "artifact_deleted_after_claim"} {
		t.Run(mode, func(t *testing.T) {
			spy := w54WithStorage(t)
			f := w54Runtime(t, true)
			source, key := w54Artifact(t, f.w54Fixture, spy, []byte("fixed synthetic evidence\r\n"), testWorkspaceID)
			f.snapshot["effective_inputs"] = append(f.snapshot["effective_inputs"].([]any), map[string]any{"id": "fixed-artifact", "source": source, "purpose": "Approved evidence", "required": true})
			var id string
			if mode == "ordinary_control" {
				id = f.fx.Task(t, f.agent, testutil.Cols{"runtime_id": f.runtime, "issue_id": f.issueID})
			} else {
				_, handoff := f.queued(t)
				id = w54ID(t, handoff, "task_id")
			}
			f.noNewRuns(t)
			f.unchangedIssue(t)
			task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
			readsBefore := spy.count(key)
			if mode == "source_changed_after_claim" {
				if _, err := spy.Upload(context.Background(), key, []byte("changed evidence\n"), "text/plain", "synthetic.txt"); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "artifact_deleted_after_claim" {
				spy.Delete(context.Background(), key)
				reader, err := spy.Storage.GetReader(context.Background(), key)
				if reader != nil {
					reader.Close()
				}
				if err == nil {
					t.Fatal("fixture artifact still readable after deletion")
				}
			}
			result := w54RunRealDaemon(t, binary, f, task)
			if strings.HasSuffix(mode, "control") {
				if result["run_error"] != "" || result["executions"] != float64(1) || result["before_sha256"] == result["after_sha256"] {
					t.Errorf("positive bridge did not execute exactly once: %v", result)
				}
				f.taskState(t, id, "running", true)
			} else {
				if result["run_error"] == "" || result["executions"] != float64(0) || result["before_sha256"] != result["after_sha256"] {
					t.Errorf("N: real API refusal failed local execution fence: %v", result)
				}
				f.taskState(t, id, "cancelled", false)
				if spy.count(key) <= readsBefore {
					t.Error("actual start did not re-read the fixed resource")
				}
				facts := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/admissions?task_id="+id, nil), 200))
				if len(facts) != 1 || facts[0].(map[string]any)["outcome"] != "rejected" {
					t.Errorf("N: durable rejection missing/duplicated: %v", facts)
				}
			}
			t.Logf("W54_N_RESOURCE mode=%s start_resource_reads=%d", mode, spy.count(key)-readsBefore)
		})
	}
}

func TestW54Contract_WS03_TC17_DefaultClaimOmitsHistory(t *testing.T) {
	for _, mode := range []string{"ordinary_control", "protected_fresh", "protected_resumed"} {
		t.Run(mode, func(t *testing.T) {
			f := w54Runtime(t, true)
			history := "UNAPPROVED_HISTORY_" + f.issueID + strings.Repeat(" synthetic old decision", 2048)
			for i := 0; i < 12; i++ {
				f.fx.Comment(t, f.issueID, history)
			}
			prior := testutil.Cols{"runtime_id": f.runtime, "issue_id": f.issueID, "status": "completed", "result": string(w54JSON(t, map[string]any{"output": history})), "completed_at": time.Now().Add(-time.Minute)}
			if mode != "protected_fresh" {
				prior["session_id"], prior["work_dir"] = "synthetic-prior-session", "/synthetic/w54/prior"
			}
			f.fx.Task(t, f.agent, prior)
			var current map[string]any
			if mode == "ordinary_control" {
				f.fx.Task(t, f.agent, testutil.Cols{"runtime_id": f.runtime, "issue_id": f.issueID})
			} else {
				f.publish(t, 0, 201)
				f.snapshot["goal"] = "Current approved replacement goal"
				current = f.publish(t, 1, 200)
				f.handoff(t, 2, "context-v2", 201)
			}
			f.noNewRuns(t)
			task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
			if strings.Contains(string(w54JSON(t, task)), "UNAPPROVED_HISTORY_") {
				t.Error("default claim injected full unapproved history/log")
			}
			if mode != "ordinary_control" {
				admission, _ := task["contract_admission"].(map[string]any)
				if !reflect.DeepEqual(admission["contract"], current) {
					t.Error("default claim omitted/replaced full current v2 contract")
				}
			}
		})
	}
}
