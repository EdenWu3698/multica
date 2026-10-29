package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/testutil"
)

// Use the original candidate builder until WUSH-59's wire supplement is
// approved. Failures before publication do not prove the later assertions ran.
func TestW54Contract_TC02_BlankAndNestedRequirements(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"goal_whitespace", func(s map[string]any) { s["goal"] = " \t\n" }},
		{"goal_placeholder", func(s map[string]any) { s["goal"] = "同上" }},
		{"prohibition_blank", func(s map[string]any) { s["prohibitions"].(map[string]any)["text"] = " " }},
		{"change_reason_blank", func(s map[string]any) { s["change"].(map[string]any)["reason"] = " " }},
		{"change_sources_empty", func(s map[string]any) { s["change"].(map[string]any)["source_refs"] = []any{} }},
		{"acceptance_assertion_blank", func(s map[string]any) { s["acceptance"].([]any)[0].(map[string]any)["assertion"] = " " }},
		{"acceptance_evidence_empty", func(s map[string]any) { s["acceptance"].([]any)[0].(map[string]any)["evidence_required"] = []any{} }},
		{"action_resources_empty", func(s map[string]any) { s["allowed_actions"].([]any)[0].(map[string]any)["resources"] = []any{} }},
		{"action_acceptance_unknown", func(s map[string]any) {
			s["allowed_actions"].([]any)[0].(map[string]any)["acceptance_ids"] = []any{"missing"}
		}},
		{"action_operation_unknown", func(s map[string]any) { s["allowed_actions"].([]any)[0].(map[string]any)["operation"] = "deploy" }},
		{"action_id_duplicate", func(s map[string]any) { a := s["allowed_actions"].([]any); s["allowed_actions"] = append(a, a[0]) }},
		{"deliverable_acceptance_unknown", func(s map[string]any) {
			s["deliverables"].([]any)[0].(map[string]any)["acceptance_ids"] = []any{"missing"}
		}},
		{"document_basis_empty", func(s map[string]any) { s["versions"].(map[string]any)["document_basis"] = []any{} }},
		{"no_code_reason_blank", func(s map[string]any) { s["versions"].(map[string]any)["code_not_applicable_reason"] = " " }},
		{"next_owner_unresolved", func(s map[string]any) { s["next_owner"] = "team" }},
		{"return_to_invalid_id", func(s map[string]any) { s["return_to"] = map[string]any{"type": "member", "id": "not-a-uuid"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := w54New(t)
			f.noNewRuns(t)
			tc.mutate(f.snapshot)
			body := f.publish(t, 0, 400, 422)
			if body["code"] != "invalid_request" && body["code"] != "contract_incomplete" {
				t.Errorf("missing stable validation code: %v", body)
			}
			errors, ok := body["field_errors"].([]any)
			if !ok || len(errors) == 0 {
				t.Errorf("missing field diagnostics: %v", body)
			}
			issue := w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)
			if issue["contract_required"] == true {
				t.Error("rejected first publication adopted the issue")
			}
		})
	}
}

func TestW54Contract_TC02_PublishRequiresBothConcurrencyPreconditions(t *testing.T) {
	for _, field := range []string{"expected_version", "expected_issue_revision"} {
		t.Run(field, func(t *testing.T) {
			f := w54New(t)
			f.noNewRuns(t)
			issue := w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)
			f.snapshot["expected_version"], f.snapshot["expected_issue_revision"] = 0, issue["revision"]
			delete(f.snapshot, field)
			body := w54Body(t, authRequest(t, "PUT", f.path(), f.snapshot), 400)
			if body["code"] != "invalid_request" {
				t.Fatalf("missing concurrency guard: %v", body)
			}
		})
	}
}

func TestW54Contract_TC04_BlockingAndNonblockingAssumptions(t *testing.T) {
	for _, blocking := range []bool{true, false} {
		name := "nonblocking"
		if blocking {
			name = "blocking"
		}
		t.Run(name, func(t *testing.T) {
			f := w54New(t)
			f.noNewRuns(t)
			f.snapshot["assumptions"] = []any{map[string]any{"assertion": "Synthetic input needs confirmation", "validation": "Ask the designated owner", "owner": f.snapshot["return_to"], "blocks_start": blocking}}
			f.publish(t, 0, 201)
			if !blocking {
				f.handoff(t, 1, name, 200, 201)
				return
			}
			body := f.handoff(t, 1, name, 422)
			if body["handoff_state"] != "needs_clarification" || !reflect.DeepEqual(body["return_to"], f.snapshot["return_to"]) {
				t.Errorf("blocking assumption lost its return path: %v", body)
			}
		})
	}
}

func TestW54Contract_TC09_DeclaredResourceBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, locator, match string
		denied               bool
		want                 int
	}{
		{"outside_directory", "private/secret.txt", "exact", false, 409},
		{"parent_escape", "../private/secret.txt", "exact", false, 400},
		{"normalized_escape", "src/../../private/secret.txt", "exact", false, 400},
		{"absolute_path", "/private/secret.txt", "exact", false, 400},
		{"glob_not_supported", "src/**", "glob", false, 400},
		{"deny_overrides_allow", "src/synthetic.txt", "exact", true, 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := w54New(t)
			f.noNewRuns(t)
			const repository = "https://example.invalid/synthetic.git"
			resource := map[string]any{"kind": "repository", "locator": map[string]any{"repository": repository, "path": "src/"}, "match": "directory"}
			f.snapshot["allowed_actions"].([]any)[0].(map[string]any)["resources"] = []any{resource}
			if tc.denied {
				f.snapshot["prohibitions"].(map[string]any)["denied_actions"] = []any{map[string]any{"id": "denied-read", "operation": "read", "resources": []any{resource}, "acceptance_ids": []any{"a1"}}}
			}
			f.publish(t, 0, 201)
			body := w54Body(t, authRequest(t, "POST", f.path()+"/handoffs", map[string]any{
				"current_version": 1, "target": f.snapshot["next_owner"], "planned_action_ids": []string{"read-input"}, "idempotency_key": tc.name,
				"resources": []any{map[string]any{"kind": "repository", "locator": map[string]any{"repository": repository, "path": tc.locator}, "match": tc.match}},
			}), tc.want)
			wantCode := "invalid_request"
			if tc.want == 409 {
				wantCode = "contract_out_of_scope"
			}
			if body["code"] != wantCode {
				t.Errorf("boundary result=%v want=%s", body, wantCode)
			}
		})
	}
}

func w54As(t *testing.T, token, workspace, method, path string, body any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, testServer.URL+path, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Workspace-ID", workspace)
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func TestW54Contract_TC10_CrossWorkspaceReadBaseline(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	outsider := f.fx.User(t, "W54 outsider", "w54-outsider@example.invalid")
	otherWS := f.fx.Workspace(t, "W54 other workspace", "w54-other-workspace")
	f.fx.Member(t, otherWS, outsider, "owner")
	token, err := generateTestJWT(outsider, "w54-outsider@example.invalid", "W54 outsider")
	if err != nil {
		t.Fatal(err)
	}
	other := testutil.New(testPool, otherWS, outsider)
	ownIssue := other.Issue(t, "W54 readable control")
	w54Body(t, w54As(t, token, otherWS, "GET", "/api/issues/"+ownIssue, nil), 200)
	for _, workspace := range []string{otherWS, testWorkspaceID} {
		response := w54As(t, token, workspace, "GET", "/api/issues/"+f.issueID, nil)
		raw, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != 403 && response.StatusCode != 404 {
			t.Errorf("cross-workspace read status=%d", response.StatusCode)
		}
		for _, secret := range []string{"Synthetic approved requirement", "W54 synthetic task", testUserID} {
			if strings.Contains(string(raw), secret) {
				t.Errorf("private value leaked: %q", secret)
			}
		}
	}
}

func TestW54Contract_TC10_RevokedMemberCannotReadContract(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	member := f.fx.User(t, "W54 receiver", "w54-receiver@example.invalid")
	membership := f.fx.Member(t, testWorkspaceID, member, "member")
	token, err := generateTestJWT(member, "w54-receiver@example.invalid", "W54 receiver")
	if err != nil {
		t.Fatal(err)
	}
	f.publish(t, 0, 201)
	w54Body(t, w54As(t, token, testWorkspaceID, "GET", f.path(), nil), 200)
	f.fx.Exec(t, "DELETE FROM member WHERE id = $1", membership)
	body := w54Body(t, w54As(t, token, testWorkspaceID, "GET", f.path(), nil), 403, 404)
	for _, key := range []string{"goal", "current_version", "return_to", "effective_inputs"} {
		if _, found := body[key]; found {
			t.Errorf("revoked member received private %s", key)
		}
	}
}

func TestW54Contract_TC11_SameKeyDifferentPayload(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	second := map[string]any{"id": "read-again", "operation": "read", "resources": f.snapshot["allowed_actions"].([]any)[0].(map[string]any)["resources"], "acceptance_ids": []any{"a1"}}
	f.snapshot["allowed_actions"] = append(f.snapshot["allowed_actions"].([]any), second)
	f.publish(t, 0, 201)
	first := f.handoff(t, 1, "same-key", 200, 201)
	conflict := w54Body(t, authRequest(t, "POST", f.path()+"/handoffs", map[string]any{
		"current_version": 1, "target": f.snapshot["next_owner"], "planned_action_ids": []string{"read-again"}, "resources": second["resources"], "idempotency_key": "same-key",
	}), 409)
	if conflict["code"] != "idempotency_conflict" {
		t.Fatalf("different payload reused key: %v", conflict)
	}
	replay := f.handoff(t, 1, "same-key", 200, 201)
	if first["handoff_id"] == nil || replay["handoff_id"] != first["handoff_id"] {
		t.Error("conflict changed original idempotency result")
	}
}

func TestW54Contract_TC10_RequestBodyCannotImpersonateReceiver(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	receiver := f.fx.User(t, "W54 actual receiver", "w54-actual-receiver@example.invalid")
	f.fx.Member(t, testWorkspaceID, receiver, "member")
	f.snapshot["next_owner"] = map[string]any{"type": "member", "id": receiver}
	f.publish(t, 0, 201)
	handoff := f.handoff(t, 1, "receiver-only", 200, 201)
	id, ok := handoff["handoff_id"].(string)
	if !ok || id == "" {
		t.Fatalf("handoff ID missing: %v", handoff)
	}
	w54Body(t, authRequest(t, "POST", f.path()+"/handoffs/"+id+"/accept", map[string]any{
		"current_version": 1, "target": f.snapshot["next_owner"], "actor": f.snapshot["next_owner"], "accepted": true,
	}), 403)
	token, err := generateTestJWT(receiver, "w54-actual-receiver@example.invalid", "W54 actual receiver")
	if err != nil {
		t.Fatal(err)
	}
	body := w54Body(t, w54As(t, token, testWorkspaceID, "POST", f.path()+"/handoffs/"+id+"/accept", map[string]any{"current_version": 1}), 200)
	if !reflect.DeepEqual(body["target"], f.snapshot["next_owner"]) {
		t.Errorf("accepted actor not bound to authenticated receiver: %v", body)
	}
}

func TestW54Contract_TC10_RevocationPrecedesIdempotencyLookup(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	caller := f.fx.User(t, "W54 handoff caller", "w54-caller@example.invalid")
	membership := f.fx.Member(t, testWorkspaceID, caller, "member")
	token, err := generateTestJWT(caller, "w54-caller@example.invalid", "W54 handoff caller")
	if err != nil {
		t.Fatal(err)
	}
	f.publish(t, 0, 201)
	payload := map[string]any{"current_version": 1, "target": f.snapshot["next_owner"], "planned_action_ids": []string{"read-input"}, "resources": f.snapshot["allowed_actions"].([]any)[0].(map[string]any)["resources"], "idempotency_key": "revoke-before-replay"}
	w54Body(t, w54As(t, token, testWorkspaceID, "POST", f.path()+"/handoffs", payload), 200, 201)
	f.fx.Exec(t, "DELETE FROM member WHERE id = $1", membership)
	body := w54Body(t, w54As(t, token, testWorkspaceID, "POST", f.path()+"/handoffs", payload), 403, 404)
	for _, key := range []string{"handoff_id", "task_id", "target", "return_to", "current_version"} {
		if _, found := body[key]; found {
			t.Errorf("dedup replay leaked %s after revocation", key)
		}
	}
}

func TestW54Contract_TC13_LegacyStartCannotBypassAdoption(t *testing.T) {
	for _, state := range []string{"dispatched", "waiting_local_directory"} {
		t.Run(state, func(t *testing.T) {
			f := w54New(t)
			f.publish(t, 0, 201)
			runtime := f.fx.Runtime(t, "W54 old runtime")
			agent := f.fx.Agent(t, "W54 fake agent", runtime)
			task := f.fx.Task(t, agent, testutil.Cols{"issue_id": f.issueID, "runtime_id": runtime, "status": state, "dispatched_at": testutil.Raw("now()")})
			f.noNewRuns(t)
			body := w54Body(t, authRequest(t, "POST", "/api/daemon/tasks/"+task+"/start", map[string]any{}), 409)
			if body["code"] != "contract_protocol_unsupported" {
				t.Errorf("legacy start not rejected: %v", body)
			}
			var status string
			var started bool
			f.fx.QueryRow(t, "SELECT status, started_at IS NOT NULL FROM agent_task_queue WHERE id = $1", task).Scan(&status, &started)
			if status != "cancelled" || started {
				t.Errorf("refused run state=%s started=%t", status, started)
			}
		})
	}
}

func TestW54Contract_TC14_TerminalRunAllowsExplicitNewVersion(t *testing.T) {
	for _, status := range []string{"completed", "failed", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			f := w54New(t)
			f.publish(t, 0, 201)
			runtime := f.fx.Runtime(t, "W54 fake runtime")
			agent := f.fx.Agent(t, "W54 fake agent", runtime)
			f.fx.Task(t, agent, testutil.Cols{"issue_id": f.issueID, "runtime_id": runtime, "status": status})
			f.noNewRuns(t)
			f.snapshot["goal"] = "Explicit new goal after termination"
			f.publish(t, 1, 200)
			current := w54Body(t, authRequest(t, "GET", f.path(), nil), 200)
			if current["version"] != float64(2) {
				t.Errorf("terminal run blocks publication: %v", current)
			}
		})
	}
}

func TestW54Contract_TC13_ClaimEntrypointsRequireCurrentAdmission(t *testing.T) {
	for _, entry := range []string{"single", "batch", "legacy_batch"} {
		for _, adopted := range []bool{false, true} {
			mode := "ordinary_control"
			if adopted {
				mode = "adopted_missing_binding"
			}
			t.Run(entry+"/"+mode, func(t *testing.T) {
				f := w54New(t)
				if adopted {
					f.publish(t, 0, 201)
				}
				const daemon = "w54-synthetic-daemon"
				runtime := f.fx.Runtime(t, "W54 old protocol runtime", testutil.Cols{"daemon_id": daemon})
				agent := f.fx.Agent(t, "W54 fake claim agent", runtime)
				task := f.fx.Task(t, agent, testutil.Cols{"runtime_id": runtime, "issue_id": f.issueID})
				f.noNewRuns(t)
				path := "/api/daemon/runtimes/" + runtime + "/tasks/claim"
				payload := map[string]any{}
				if entry != "single" {
					path = "/api/daemon/tasks/claim"
					if entry == "legacy_batch" {
						path = "/api/daemon/claim"
					}
					payload = map[string]any{"daemon_id": daemon, "runtime_ids": []string{runtime}, "max_tasks": 1}
				}
				body := w54Body(t, authRequest(t, "POST", path, payload), 200, 409)
				var status string
				var started bool
				f.fx.QueryRow(t, "SELECT status, started_at IS NOT NULL FROM agent_task_queue WHERE id = $1", task).Scan(&status, &started)
				if adopted {
					if status != "cancelled" || started {
						t.Errorf("unbound protected task escaped claim gate: status=%s started=%t", status, started)
					}
					if body["task"] != nil {
						t.Error("protected task delivered to legacy runtime")
					}
					if tasks, ok := body["tasks"].([]any); ok && len(tasks) != 0 {
						t.Error("protected task delivered in batch")
					}
				} else {
					if status != "dispatched" || started {
						t.Errorf("ordinary fixture not claimed: status=%s started=%t body=%v", status, started, body)
					}
					if entry == "single" {
						got, ok := body["task"].(map[string]any)
						if !ok || got["id"] != task {
							t.Errorf("ordinary claim payload missing: %v", body)
						}
					} else {
						got, ok := body["tasks"].([]any)
						if !ok || len(got) != 1 {
							t.Errorf("ordinary batch payload missing: %v", body)
						}
					}
				}
			})
		}
	}
}
