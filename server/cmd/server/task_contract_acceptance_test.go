package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/testutil"
)

// W54-SPEC-0.1 + approved wire supplement fixtures; no production validator.
type w54Fixture struct {
	fx       *testutil.Fixture
	issueID  string
	snapshot map[string]any
}

func w54Digest(s string) string {
	d := sha256.Sum256([]byte(s))
	return hex.EncodeToString(d[:])
}

func w54New(t *testing.T) w54Fixture {
	t.Helper()
	fx := testutil.New(testPool, testWorkspaceID, testUserID)
	id := fx.Issue(t, "W54 synthetic task", testutil.Cols{"description": "Synthetic approved requirement", "status": "backlog"})
	issue := w54Body(t, authRequest(t, "GET", "/api/issues/"+id, nil), 200)
	actor := map[string]any{"type": "member", "id": testUserID}
	source := map[string]any{"kind": "issue", "locator": id, "version": issue["revision"],
		"digest": w54JSONDigest(t, map[string]any{"title": issue["title"], "description": issue["description"]}), "supplied_by": actor}
	resource := map[string]any{"kind": "issue", "locator": id, "match": "exact"}
	refs := []any{source}
	stops := []any{}
	for _, code := range []string{"input_unavailable", "input_version_mismatch", "contract_out_of_scope", "permission_denied", "blocking_question"} {
		stops = append(stops, map[string]any{"code": code, "condition": code, "return_to": actor})
	}
	return w54Fixture{fx: fx, issueID: id, snapshot: map[string]any{
		"schema_version": 1, "goal": "Read the synthetic requirement and deliver test evidence", "goal_sources": refs,
		"allowed_actions":  []any{map[string]any{"id": "read-input", "operation": "read", "resources": []any{resource}, "acceptance_ids": []any{"a1"}}},
		"prohibitions":     map[string]any{"text": "Do not edit product code or dispatch downstream runs", "denied_actions": []any{}},
		"acceptance":       []any{map[string]any{"id": "a1", "assertion": "The evidence identifies the input version", "evidence_required": []any{"test-log"}, "source_refs": refs}},
		"effective_inputs": []any{map[string]any{"id": "input1", "source": source, "purpose": "Approved requirement", "required": true}},
		"confirmed_facts":  []any{}, "assumptions": []any{},
		"versions":       map[string]any{"code_basis": []any{}, "code_not_applicable_reason": "Member reads a synthetic issue only", "document_basis": []any{map[string]any{"source": source, "applicability": "Main requirement"}}},
		"deliverables":   []any{map[string]any{"id": "d1", "description": "Versioned test evidence", "acceptance_ids": []any{"a1"}, "evidence_refs": []any{}}},
		"open_questions": []any{}, "stop_conditions": stops, "next_owner": actor, "return_to": actor,
		"decision_sources": []any{map[string]any{"source": source, "actor": actor, "decided_at": "2026-10-29T00:00:00Z"}},
		"change":           map[string]any{"reason": "Establish synthetic test contract", "source_refs": refs},
	}}
}

func (f w54Fixture) path() string { return "/api/issues/" + f.issueID + "/task-contract" }

func w54Body(t *testing.T, response *http.Response, want ...int) map[string]any {
	t.Helper()
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("transport result unknown: %v; confirm with the same key before asserting zero", err)
	}
	ok := false
	for _, status := range want {
		ok = ok || response.StatusCode == status
	}
	if !ok {
		t.Fatalf("behavior: status=%d want=%v body=%s", response.StatusCode, want, raw)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("behavior: App API must return JSON, got %q: %v", raw, err)
	}
	return body
}

func (f w54Fixture) publish(t *testing.T, version int, want ...int) map[string]any {
	t.Helper()
	issue := w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)
	input := map[string]any{}
	for key, value := range f.snapshot {
		input[key] = value
	}
	input["expected_version"] = version
	input["expected_issue_revision"] = issue["revision"]
	return w54Body(t, authRequest(t, "PUT", f.path(), input), want...)
}

func (f w54Fixture) handoff(t *testing.T, version int, key string, want ...int) map[string]any {
	t.Helper()
	return w54Body(t, authRequest(t, "POST", f.path()+"/handoffs", map[string]any{
		"current_version": version, "planned_action_ids": []string{"read-input"},
		"resources": []any{map[string]any{"kind": "issue", "locator": f.issueID, "match": "exact"}},
		"target":    f.snapshot["next_owner"], "idempotency_key": key,
	}), want...)
}

func (f w54Fixture) noNewRuns(t *testing.T) {
	t.Helper()
	// Observe the whole isolated workspace: a downstream run can target a
	// different issue (or no issue) and must not escape the negative assertion.
	const query = `SELECT count(*) FROM agent_task_queue q JOIN agent a ON a.id = q.agent_id WHERE a.workspace_id = $1`
	before := f.fx.Count(t, query, testWorkspaceID)
	t.Cleanup(func() {
		after := f.fx.Count(t, query, testWorkspaceID)
		if after != before {
			t.Errorf("N: run count changed from %d to %d", before, after)
		}
	})
}

func TestW54Contract_TC01_MissingIsMachineReadable(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	body := w54Body(t, authRequest(t, "GET", f.path(), nil), 404)
	if body["code"] != "contract_missing" {
		t.Fatalf("code=%v, want contract_missing", body["code"])
	}
}

func TestW54Contract_TC01_MemberPublishReadAccept(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	f.publish(t, 0, 201)
	current := w54Body(t, authRequest(t, "GET", f.path(), nil), 200)
	if current["version"] != float64(1) || current["goal"] != f.snapshot["goal"] {
		t.Fatalf("current snapshot lost version/goal: %v", current)
	}
	handoff := f.handoff(t, 1, "member-first", 201, 200)
	id, ok := handoff["handoff_id"].(string)
	if !ok || id == "" {
		t.Fatalf("handoff has no durable ID: %v", handoff)
	}
	receipt := w54Body(t, authRequest(t, "POST", f.path()+"/handoffs/"+id+"/accept", map[string]any{"current_version": 1}), 200)
	for _, field := range []string{"handoff_id", "version", "target", "input_digest", "planned_actions", "accepted_at"} {
		if receipt[field] == nil {
			t.Errorf("member receipt missing %s", field)
		}
	}
	for _, field := range []string{"task_id", "claim_generation"} {
		value, present := receipt[field]
		if !present || value != nil {
			t.Errorf("member receipt %s must be explicitly null: %v", field, receipt)
		}
	}
}

func TestW54Contract_TC02_RequiredFields(t *testing.T) {
	fields := []string{"schema_version", "goal", "goal_sources", "allowed_actions", "prohibitions", "acceptance", "effective_inputs", "confirmed_facts", "assumptions", "versions", "deliverables", "open_questions", "stop_conditions", "next_owner", "return_to", "decision_sources", "change"}
	for _, field := range fields {
		for _, variant := range []string{"missing", "null"} {
			t.Run(field+"/"+variant, func(t *testing.T) {
				f := w54New(t)
				f.noNewRuns(t)
				if variant == "missing" {
					delete(f.snapshot, field)
				} else {
					f.snapshot[field] = nil
				}
				body := f.publish(t, 0, 400, 422)
				if body["code"] != "invalid_request" && body["code"] != "contract_incomplete" {
					t.Errorf("wrong rejection code: %v", body)
				}
				encoded, _ := json.Marshal(body["field_errors"])
				var errors []map[string]any
				if json.Unmarshal(encoded, &errors) != nil || len(errors) == 0 {
					t.Errorf("missing field diagnostic for %s: %v", field, body)
				}
				w54Body(t, authRequest(t, "GET", f.path(), nil), 404)
			})
		}
	}
}

func TestW54Contract_TC03_ExplicitEmptyOptionalArrays(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	f.publish(t, 0, 201)
}

func TestW54Contract_TC03_RequiredCollectionsCannotBeEmpty(t *testing.T) {
	for _, field := range []string{"goal_sources", "allowed_actions", "acceptance", "effective_inputs", "deliverables", "stop_conditions", "decision_sources"} {
		t.Run(field, func(t *testing.T) {
			f := w54New(t)
			f.noNewRuns(t)
			f.snapshot[field] = []any{}
			f.publish(t, 0, 400, 422)
		})
	}
}

func TestW54Contract_TC04_BlockingQuestion(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	f.snapshot["open_questions"] = []any{map[string]any{"id": "q1", "question": "Which input is approved?", "owner": f.snapshot["return_to"], "blocks_start": true}}
	f.publish(t, 0, 201)
	body := f.handoff(t, 1, "blocked-question", 422)
	if body["code"] != "blocking_question" || body["handoff_state"] != "needs_clarification" || !reflect.DeepEqual(body["return_to"], f.snapshot["return_to"]) {
		t.Fatalf("wrong return-to result: %v", body)
	}
}

func TestW54Contract_TC05_OldVersionAndImmutableHistory(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	f.publish(t, 0, 201)
	before := w54Body(t, authRequest(t, "GET", f.path()+"/versions/1", nil), 200)
	f.snapshot["acceptance"].([]any)[0].(map[string]any)["assertion"] = "Evidence includes a source digest"
	f.snapshot["change"].(map[string]any)["reason"] = "Tighten evidence clarity"
	f.publish(t, 1, 200)
	body := f.handoff(t, 1, "old-page", 409)
	if body["code"] != "contract_stale" || body["current_version"] != float64(2) {
		t.Fatalf("old page was not rejected as stale: %v", body)
	}
	after := w54Body(t, authRequest(t, "GET", f.path()+"/versions/1", nil), 200)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("historical snapshot was rewritten")
	}
	f.handoff(t, 2, "current-page", 201, 200)
}

func TestW54Contract_TC06_ConcurrentPublish(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	f.publish(t, 0, 201)
	issue := w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)
	gate := make(chan struct{})
	type outcome struct {
		response *http.Response
		err      error
	}
	responses := make(chan outcome, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for i := 0; i < 2; i++ {
		body := map[string]any{}
		for key, value := range f.snapshot {
			body[key] = value
		}
		body["goal"] = fmt.Sprintf("Synthetic concurrent goal %d", i)
		body["expected_version"] = 1
		body["expected_issue_revision"] = issue["revision"]
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			ready.Done()
			<-gate
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, "PUT", testServer.URL+f.path(), bytes.NewReader(raw))
			if err != nil {
				responses <- outcome{err: err}
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+testToken)
			req.Header.Set("X-Workspace-ID", testWorkspaceID)
			response, err := http.DefaultClient.Do(req)
			responses <- outcome{response: response, err: err}
		}()
	}
	ready.Wait()
	close(gate)
	counts := map[int]int{}
	for i := 0; i < 2; i++ {
		result := <-responses
		if result.err != nil {
			t.Errorf("publish result unknown: %v", result.err)
			continue
		}
		counts[result.response.StatusCode]++
		w54Body(t, result.response, 200, 409)
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatalf("two editors: got %v, want one 200 and one 409", counts)
	}
	current := w54Body(t, authRequest(t, "GET", f.path(), nil), 200)
	if current["version"] != float64(2) {
		t.Fatalf("version incremented more than once: %v", current)
	}
}

func TestW54Contract_TC07_CommentRevisionVersusContent(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	f.publish(t, 0, 201)
	before := w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)
	w54Body(t, authRequest(t, "POST", "/api/issues/"+f.issueID+"/comments", map[string]any{"content": "Synthetic ordinary comment"}), 201)
	after := w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)
	if after["revision"].(float64) <= before["revision"].(float64) {
		t.Fatal("fixture did not advance aggregate revision")
	}
	f.handoff(t, 1, "comment-does-not-stale", 201, 200)
	w54Body(t, authRequest(t, "PUT", "/api/issues/"+f.issueID, map[string]any{"description": "Changed requirement"}), 200)
	body := f.handoff(t, 1, "description-does-stale", 409)
	if body["code"] != "source_changed" {
		t.Fatalf("changed issue content accepted: %v", body)
	}
}

func TestW54Contract_TC08_FloatingCodeBaseline(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	versions := f.snapshot["versions"].(map[string]any)
	versions["code_basis"] = []any{map[string]any{"repository": "https://example.invalid/synthetic.git", "commit_sha": "main", "branch": "main", "dirty_snapshot": nil}}
	delete(versions, "code_not_applicable_reason")
	f.publish(t, 0, 400, 422)
}

func TestW54Contract_TC09_UnknownActionAndReceiver(t *testing.T) {
	for _, mode := range []string{"action", "receiver"} {
		t.Run(mode, func(t *testing.T) {
			f := w54New(t)
			f.noNewRuns(t)
			f.publish(t, 0, 201)
			body := map[string]any{"current_version": 1, "planned_action_ids": []string{"read-input"}, "resources": f.snapshot["allowed_actions"].([]any)[0].(map[string]any)["resources"], "target": f.snapshot["next_owner"], "idempotency_key": "invalid-" + mode}
			wantCode := "contract_out_of_scope"
			if mode == "action" {
				body["planned_action_ids"] = []string{"unapproved-edit"}
			} else {
				other := f.fx.User(t, "Synthetic other member", "w54-other@example.invalid")
				f.fx.Member(t, testWorkspaceID, other, "member")
				body["target"] = map[string]any{"type": "member", "id": other}
				wantCode = "receiver_mismatch"
			}
			result := w54Body(t, authRequest(t, "POST", f.path()+"/handoffs", body), 409)
			if result["code"] != wantCode {
				t.Fatalf("code=%v want=%s", result["code"], wantCode)
			}
		})
	}
}

func TestW54Contract_TC10_AnonymousCannotRead(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	response, err := http.Get(testServer.URL + f.path())
	if err != nil {
		t.Fatal(err)
	}
	w54Body(t, response, 401)
}

func TestW54Contract_TC11_IdempotentMemberHandoff(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	f.publish(t, 0, 201)
	a := f.handoff(t, 1, "same-key", 201, 200)
	b := f.handoff(t, 1, "same-key", 200, 201)
	if a["handoff_id"] == nil || a["handoff_id"] != b["handoff_id"] {
		t.Fatalf("duplicate handoff: %v / %v", a, b)
	}
	f.snapshot["goal"] = "New approved goal"
	f.publish(t, 1, 200)
	stale := f.handoff(t, 1, "same-key", 409)
	if stale["code"] != "contract_stale" || stale["handoff_id"] != a["handoff_id"] {
		t.Fatalf("old idempotency result treated as current: %v", stale)
	}
}

func TestW54Contract_TC13_AdoptionCannotBeUnsetByOrdinaryUpdate(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	f.publish(t, 0, 201)
	response := authRequest(t, "PUT", "/api/issues/"+f.issueID, map[string]any{"contract_required": false})
	w54Body(t, response, 200, 400, 403)
	issue := w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)
	if issue["contract_required"] != true {
		t.Fatalf("adoption cleared: %v", issue["contract_required"])
	}
}

func TestW54Contract_TC14_ActiveRunBlocksPublish(t *testing.T) {
	for _, status := range []string{"dispatched", "running", "waiting_local_directory"} {
		t.Run(status, func(t *testing.T) {
			f := w54New(t)
			f.publish(t, 0, 201)
			runtime := f.fx.Runtime(t, "W54 fake runtime")
			agent := f.fx.Agent(t, "W54 fake agent", runtime)
			f.fx.Task(t, agent, testutil.Cols{"issue_id": f.issueID, "status": status})
			f.noNewRuns(t)
			f.snapshot["goal"] = "New goal while active"
			body := f.publish(t, 1, 409)
			if body["code"] != "contract_run_active" {
				t.Fatalf("active run not protected: %v", body)
			}
			current := w54Body(t, authRequest(t, "GET", f.path(), nil), 200)
			if current["version"] != float64(1) {
				t.Fatal("active contract changed")
			}
		})
	}
}

func TestW54Contract_TC17_OrdinaryIssueBaseline(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	body := w54Body(t, authRequest(t, "PUT", "/api/issues/"+f.issueID, map[string]any{"title": "Ordinary task remains editable"}), 200)
	if body["title"] != "Ordinary task remains editable" || body["contract_required"] == true {
		t.Fatalf("ordinary task changed behavior: %v", body)
	}
}

func TestW54Contract_TC18_SeparateRequestsPreserveSnapshot(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	f.publish(t, 0, 201)
	a := w54Body(t, authRequest(t, "GET", f.path(), nil), 200)
	b := w54Body(t, authRequest(t, "GET", f.path(), nil), 200)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("persisted snapshot changed between independent requests")
	}
	var adopted bool
	if err := testPool.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM issue WHERE id = $1)", f.issueID).Scan(&adopted); err != nil || !adopted {
		t.Fatalf("fixture persistence: %v", err)
	}
}

func TestW54Contract_TC17_ReadLatencyBaseline(t *testing.T) {
	f := w54New(t)
	f.noNewRuns(t)
	for _, route := range []string{"/api/issues/" + f.issueID, "/api/issues?workspace_id=" + testWorkspaceID} {
		times := make([]float64, 0, 30)
		for i := 0; i < 31; i++ {
			start := time.Now()
			w54Body(t, authRequest(t, "GET", route, nil), 200)
			if i > 0 {
				times = append(times, float64(time.Since(start).Microseconds())/1000)
			}
		}
		sort.Float64s(times)
		t.Logf("W54_PERF route=%s fixture_issues=1 samples=30 warmup=1 p50_ms=%.3f p95_ms=%.3f max_ms=%.3f", route, times[14], times[28], times[29])
	}
}
