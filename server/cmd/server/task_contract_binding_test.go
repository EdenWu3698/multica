package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/auth"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/internal/service"
	"github.com/multica-ai/multica/server/internal/testutil"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type w54RuntimeFixture struct {
	w54Fixture
	runtime, agent, daemon, token string
}

func w54Runtime(t *testing.T, capable bool) w54RuntimeFixture {
	t.Helper()
	f := w54New(t)
	d := "w54-wire-" + f.issueID
	metadata := `{"capabilities":["task-contract-v1"]}`
	if !capable {
		metadata = `{"capabilities":["unknown-contract-protocol"]}`
	}
	r := f.fx.Runtime(t, "Synthetic authenticated runtime", testutil.Cols{"daemon_id": d, "metadata": metadata})
	a := f.fx.Agent(t, "Synthetic contract receiver", r)
	token, err := auth.GenerateDaemonToken()
	if err != nil {
		t.Fatal(err)
	}
	f.fx.Insert(t, "daemon_token", testutil.Cols{"token_hash": auth.HashToken(token), "workspace_id": testWorkspaceID, "daemon_id": d, "owner_id": testUserID, "expires_at": time.Now().Add(time.Hour)})
	f.snapshot["next_owner"] = map[string]any{"type": "agent", "id": a}
	return w54RuntimeFixture{f, r, a, d, token}
}

func (f w54RuntimeFixture) request(t *testing.T, path string, body any, capabilities string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("POST", testServer.URL+path, bytes.NewReader(w54JSON(t, body)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Capabilities", capabilities)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func (f w54RuntimeFixture) claim(t *testing.T, entry, capability string) map[string]any {
	t.Helper()
	path := "/api/daemon/runtimes/" + f.runtime + "/tasks/claim"
	body := map[string]any{}
	if entry != "single" {
		path = "/api/daemon/tasks/claim"
		if entry == "legacy_batch" {
			path = "/api/daemon/claim"
		}
		body = map[string]any{"daemon_id": f.daemon, "runtime_ids": []string{f.runtime}, "max_tasks": 1}
	}
	return w54Body(t, f.request(t, path, body, capability), 200, 409)
}

func w54Claimed(t *testing.T, body map[string]any, entry string) map[string]any {
	t.Helper()
	if entry == "single" {
		if task, ok := body["task"].(map[string]any); ok {
			return task
		}
	} else {
		if tasks, ok := body["tasks"].([]any); ok && len(tasks) == 1 {
			return tasks[0].(map[string]any)
		}
	}
	t.Fatalf("expected exactly one real claimed task: %v", body)
	return nil
}

func w54StartBody(t *testing.T, task map[string]any) map[string]any {
	t.Helper()
	a, ok := task["contract_admission"].(map[string]any)
	if !ok {
		t.Fatalf("claim missing contract_admission: %v", task)
	}
	return map[string]any{"runtime_id": task["runtime_id"], "dispatched_at": task["dispatched_at"], "capabilities": []any{"task-contract-v1"},
		"contract_admission": map[string]any{"handoff_id": a["handoff_id"], "version": a["version"], "input_digest": a["input_digest"], "planned_actions": a["planned_actions"], "code_observations": []any{}}}
}

func (f w54RuntimeFixture) queued(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	snapshot := f.publish(t, 0, 201)
	h := f.handoff(t, 1, "runtime-first", 201)
	if h["state"] != "pending" || h["receipt"] != nil || h["task_id"] == nil {
		t.Fatalf("queued is not accepted: %v", h)
	}
	return snapshot, h
}

func (f w54RuntimeFixture) taskState(t *testing.T, id, status string, started bool) {
	t.Helper()
	var got string
	var hasStart bool
	f.fx.QueryRow(t, "SELECT status,started_at IS NOT NULL FROM agent_task_queue WHERE id=$1", id).Scan(&got, &hasStart)
	if got != status || hasStart != started {
		t.Errorf("task=%s state=%s started=%t want=%s/%t", id, got, hasStart, status, started)
	}
}

func TestW54Contract_WS03_TC17_AuthenticatedDaemonBaseline(t *testing.T) {
	f := w54Runtime(t, true)
	id := f.fx.Task(t, f.agent, testutil.Cols{"runtime_id": f.runtime, "issue_id": f.issueID})
	f.noNewRuns(t)
	task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
	if task["id"] != id {
		t.Fatal("ordinary control claimed a different task")
	}
	f.taskState(t, id, "dispatched", false)
	w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", map[string]any{"runtime_id": f.runtime, "dispatched_at": task["dispatched_at"], "capabilities": []string{"task-contract-v1"}}, "task-contract-v1"), 200)
	f.taskState(t, id, "running", true)
}

func TestW54Contract_WS03_TC01_ClaimStartAndReceipt(t *testing.T) {
	for _, entry := range []string{"single", "batch", "legacy_batch"} {
		t.Run(entry, func(t *testing.T) {
			f := w54Runtime(t, true)
			snapshot, h := f.queued(t)
			id := w54ID(t, h, "task_id")
			f.noNewRuns(t)
			task := w54Claimed(t, f.claim(t, entry, "task-contract-v1"), entry)
			a, ok := task["contract_admission"].(map[string]any)
			if !ok {
				t.Fatalf("missing protected claim: %v", task)
			}
			if a["status"] != "pending" || a["receipt"] != nil || a["task_id"] != id || a["runtime_id"] != f.runtime || a["claim_generation"] != task["dispatched_at"] || a["input_digest"] != w54InputDigest(t, snapshot) {
				t.Fatalf("claim binding mismatch: %v", a)
			}
			if !reflect.DeepEqual(a["contract"], snapshot) || !reflect.DeepEqual(a["planned_actions"], h["planned_actions"]) || !reflect.DeepEqual(a["code_basis"], snapshot["versions"].(map[string]any)["code_basis"]) {
				t.Error("claim lost complete current contract/plan")
			}
			for _, field := range []string{"history", "versions_history", "logs"} {
				if _, ok := a[field]; ok {
					t.Errorf("default claim includes %s", field)
				}
			}
			f.taskState(t, id, "dispatched", false)
			start := w54StartBody(t, task)
			result := w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", start, "task-contract-v1"), 200)
			accepted := result["contract_admission"].(map[string]any)
			receipt := accepted["receipt"].(map[string]any)
			if accepted["status"] != "accepted" || receipt["task_id"] != id || receipt["runtime_id"] != f.runtime || receipt["claim_generation"] != task["dispatched_at"] || receipt["input_digest"] != a["input_digest"] || !reflect.DeepEqual(receipt["accepted_by"], f.snapshot["next_owner"]) {
				t.Errorf("start receipt mismatch: %v", accepted)
			}
			f.taskState(t, id, "running", true)
			replay := w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", start, "task-contract-v1"), 200)
			if !reflect.DeepEqual(replay["contract_admission"], accepted) {
				t.Error("start replay minted another receipt")
			}
			rows := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/admissions?task_id="+id, nil), 200))
			if len(rows) != 1 {
				t.Fatalf("start admission duplicated: %v", rows)
			}
			ad := rows[0].(map[string]any)
			if ad["operation"] != "start" || ad["outcome"] != "accepted" || ad["runtime_id"] != f.runtime || !reflect.DeepEqual(ad["code_observations"], []any{}) {
				t.Errorf("wrong durable runtime observation: %v", ad)
			}
		})
	}
}

func TestW54Contract_WS03_TC13_CapabilitiesMustBePersistentAndCurrent(t *testing.T) {
	for _, entry := range []string{"single", "batch", "legacy_batch"} {
		for _, mode := range []string{"persistent_missing", "connection_missing", "connection_unknown"} {
			t.Run(entry+"/"+mode, func(t *testing.T) {
				f := w54Runtime(t, mode != "persistent_missing")
				_, h := f.queued(t)
				id := w54ID(t, h, "task_id")
				f.noNewRuns(t)
				capability := "task-contract-v1"
				if mode == "connection_missing" {
					capability = ""
				}
				if mode == "connection_unknown" {
					capability = "task-contract-v9"
				}
				body := f.claim(t, entry, capability)
				if body["task"] != nil {
					t.Error("protected task delivered without negotiated capability")
				}
				if tasks, ok := body["tasks"].([]any); ok && len(tasks) > 0 {
					t.Error("batch delivered protected task")
				}
				f.taskState(t, id, "cancelled", false)
				rows := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/admissions?task_id="+id, nil), 200))
				if len(rows) != 1 || rows[0].(map[string]any)["code"] != "contract_protocol_unsupported" {
					t.Errorf("claim rejection lost: %v", rows)
				}
			})
		}
	}
}

func TestW54Contract_WS03_TC08_StartBindingMutations(t *testing.T) {
	for _, tc := range []struct {
		name, code, path string
		status           int
		mutate           func(map[string]any)
	}{
		{"version", "contract_stale", "/contract_admission/version", 409, func(b map[string]any) { b["contract_admission"].(map[string]any)["version"] = 2 }},
		{"handoff", "contract_stale", "/contract_admission/handoff_id", 409, func(b map[string]any) {
			b["contract_admission"].(map[string]any)["handoff_id"] = "00000000-0000-4000-8000-000000000099"
		}},
		{"digest", "input_version_mismatch", "/contract_admission/input_digest", 409, func(b map[string]any) {
			b["contract_admission"].(map[string]any)["input_digest"] = w54Digest("wrong actual input")
		}},
		{"plan", "contract_out_of_scope", "/contract_admission/planned_actions", 409, func(b map[string]any) { b["contract_admission"].(map[string]any)["planned_actions"] = []any{} }},
		{"missing_observations", "invalid_request", "/contract_admission/code_observations", 400, func(b map[string]any) { delete(b["contract_admission"].(map[string]any), "code_observations") }},
		{"null_observations", "invalid_request", "/contract_admission/code_observations", 400, func(b map[string]any) { b["contract_admission"].(map[string]any)["code_observations"] = nil }},
		{"forged_task", "invalid_request", "/contract_admission/task_id", 400, func(b map[string]any) {
			b["contract_admission"].(map[string]any)["task_id"] = "00000000-0000-4000-8000-000000000099"
		}},
		{"forged_generation", "invalid_request", "/contract_admission/claim_generation", 400, func(b map[string]any) {
			b["contract_admission"].(map[string]any)["claim_generation"] = b["dispatched_at"]
		}},
		{"missing_start_capability", "contract_protocol_unsupported", "/capabilities", 409, func(b map[string]any) { b["capabilities"] = []any{} }},
		{"unknown_start_capability", "contract_protocol_unsupported", "/capabilities", 409, func(b map[string]any) { b["capabilities"] = []any{"task-contract-v9"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := w54Runtime(t, true)
			_, h := f.queued(t)
			id := w54ID(t, h, "task_id")
			f.noNewRuns(t)
			task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
			b := w54StartBody(t, task)
			tc.mutate(b)
			result := w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", b, "task-contract-v1"), tc.status)
			w54Field(t, result, tc.code, tc.path, "")
			if tc.status == 409 {
				f.taskState(t, id, "cancelled", false)
				w54ID(t, result, "admission_id")
			} else {
				f.taskState(t, id, "dispatched", false)
			}
		})
	}
}

func TestW54Contract_WS03_TC10_MemberCannotSubmitRuntimeObservation(t *testing.T) {
	f := w54Runtime(t, true)
	_, h := f.queued(t)
	id := w54ID(t, h, "task_id")
	f.noNewRuns(t)
	task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
	for _, kind := range []string{"member", "task_token", "other_runtime"} {
		t.Run(kind, func(t *testing.T) {
			caller := f
			caller.token = testToken
			if kind == "task_token" {
				token, err := auth.GenerateAgentTaskToken()
				if err != nil {
					t.Fatal(err)
				}
				caller.token = token
				f.fx.Insert(t, "task_token", testutil.Cols{"token_hash": auth.HashToken(token), "task_id": id, "agent_id": f.agent, "workspace_id": testWorkspaceID, "user_id": testUserID, "expires_at": time.Now().Add(time.Hour)})
			}
			if kind == "other_runtime" {
				other := w54Runtime(t, true)
				caller.token = other.token
			}
			body := w54Body(t, caller.request(t, "/api/daemon/tasks/"+id+"/start", w54StartBody(t, task), "task-contract-v1"), 403, 404)
			for _, key := range []string{"contract_admission", "return_to", "input_digest", "current_version", "admission_id"} {
				if _, ok := body[key]; ok {
					t.Errorf("unauthorized identity leaked %s", key)
				}
			}
			f.taskState(t, id, "dispatched", false)
		})
	}
	refused := w54Body(t, authRequest(t, "POST", f.path()+"/handoffs/"+w54ID(t, h, "handoff_id")+"/accept", map[string]any{"current_version": 1}), 403)
	if refused["code"] != "agent_accept_via_runtime_only" {
		t.Errorf("Web signed for agent: %v", refused)
	}
}

func TestW54Contract_WS03_TC12_OldGenerationCannotCancelReclaimedRun(t *testing.T) {
	f := w54Runtime(t, true)
	_, h := f.queued(t)
	id := w54ID(t, h, "task_id")
	f.noNewRuns(t)
	old := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
	// Seed the existing requeue transition, then obtain G2 from the real claim path.
	f.fx.Exec(t, "UPDATE agent_task_queue SET status='queued', dispatched_at=NULL WHERE id=$1", id)
	current := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
	if current["dispatched_at"] == old["dispatched_at"] {
		t.Fatal("fixture failed to create a new generation")
	}
	w54Field(t, w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", w54StartBody(t, old), "task-contract-v1"), 409), "contract_stale", "/dispatched_at", "")
	f.taskState(t, id, "dispatched", false)
	good := w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", w54StartBody(t, current), "task-contract-v1"), 200)
	w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", w54StartBody(t, old), "task-contract-v1"), 409)
	f.taskState(t, id, "running", true)
	handoff := w54Body(t, authRequest(t, "GET", f.path()+"/handoffs/"+w54ID(t, h, "handoff_id"), nil), 200)
	if !reflect.DeepEqual(handoff["receipt"], good["contract_admission"].(map[string]any)["receipt"]) {
		t.Error("old generation rewrote G2 receipt")
	}
	t.Logf("W54_BINDING task=%s runtime=%s G1=%s G2=%s", id, f.runtime, old["dispatched_at"], current["dispatched_at"])
}

func TestW54Contract_WS03_TC13_RecoveryCannotCopyReceipt(t *testing.T) {
	for _, entry := range []string{"issue_rerun", "task_rerun", "resume_claim", "automatic_retry", "wakeup"} {
		t.Run(entry, func(t *testing.T) {
			f := w54Runtime(t, true)
			_, h := f.queued(t)
			id := w54ID(t, h, "task_id")
			task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
			w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", w54StartBody(t, task), "task-contract-v1"), 200)
			f.fx.Exec(t, "UPDATE agent_task_queue SET status='completed', completed_at=now(), session_id='synthetic-old-session' WHERE id=$1", id)
			w54Body(t, authRequest(t, "PUT", "/api/issues/"+f.issueID, map[string]any{"description": "Requirement changed before recovery"}), 200)
			if entry == "automatic_retry" {
				f.fx.Exec(t, "UPDATE agent_task_queue SET status='failed', failure_reason='provider_network', attempt=1, max_attempts=3 WHERE id=$1", id)
				parsed, err := util.ParseUUID(id)
				if err != nil {
					t.Fatal(err)
				}
				queries := db.New(testPool)
				parent, err := queries.GetAgentTask(context.Background(), parsed)
				if err != nil {
					t.Fatal(err)
				}
				svc := service.NewTaskService(queries, testPool, realtime.NewHub(), events.New())
				if _, err = svc.MaybeRetryFailedTask(context.Background(), parent); err != nil {
					t.Fatalf("real retry entry: %v", err)
				}
			} else if entry == "wakeup" {
				f.fx.Cleanup(t, "DELETE FROM issue_wakeup WHERE issue_id=$1", f.issueID)
				f.fx.Cleanup(t, "DELETE FROM issue_wakeup_receipt WHERE wakeup_id IN (SELECT id FROM issue_wakeup WHERE issue_id=$1)", f.issueID)
				wake := w54Body(t, authRequest(t, "POST", "/api/issues/"+f.issueID+"/wakeups", map[string]any{"agent_id": f.agent, "kind": "at", "after_seconds": 600, "instruction": "Recheck approved synthetic input"}), 201)
				response := authRequest(t, "POST", "/api/issues/"+f.issueID+"/wakeups/"+w54ID(t, wake, "id")+"/trigger", map[string]any{})
				response.Body.Close()
				if response.StatusCode != 204 {
					t.Fatalf("wakeup trigger=%d want=204", response.StatusCode)
				}
			} else if entry == "resume_claim" {
				f.fx.Exec(t, "UPDATE agent_task_queue SET status='queued', started_at=NULL, dispatched_at=NULL WHERE id=$1", id)
			} else {
				path := "/api/issues/" + f.issueID + "/rerun"
				if entry == "task_rerun" {
					path = "/api/tasks/" + id + "/rerun"
				}
				response := authRequest(t, "POST", path, map[string]any{"task_id": id})
				// Existing dispatch may reject synchronously or leave a queued task;
				// neither may make the changed input executable.
				w54Body(t, response, 200, 201, 202, 409, 422)
			}
			f.claim(t, "single", "task-contract-v1")
			var active int
			f.fx.QueryRow(t, "SELECT count(*) FROM agent_task_queue WHERE issue_id=$1 AND status IN ('dispatched','running','waiting_local_directory')", f.issueID).Scan(&active)
			if active != 0 {
				t.Errorf("%s reused prior receipt for changed input", entry)
			}
			rows := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/admissions", nil), 200))
			found := false
			for _, v := range rows {
				a := v.(map[string]any)
				found = found || a["outcome"] == "rejected"
			}
			if !found {
				t.Errorf("%s lost recovery rejection: %v", entry, rows)
			}
			t.Log(fmt.Sprintf("W54_RECOVERY entry=%s original_task=%s", entry, id))
		})
	}
}
