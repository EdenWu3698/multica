package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"
)

// This adapter is deliberately unbound at the product base. JARVIS supplies the
// actual internal hook mapping; MOSS must not implement a transaction/service
// to make these tests run. It may attach only to the real App API test router.
// A nil mapping is LOCAL_PENDING, never a passing fault-injection test.
var w54AttachProductSeams func(*testing.T, func(context.Context, w54SeamEvent) (string, error)) (detach func(), err error)

type w54SeamEvent struct {
	Point, Operation, RequestID, IssueID, HandoffID, TaskID, Generation string
	ResourceLocator                                                     string
	Version                                                             int
	Committed                                                           bool
}

func TestW54Contract_WS05_ActualHookMapping(t *testing.T) {
	f := w54New(t)
	if w54AttachProductSeams == nil {
		t.Fatalf("LOCAL_PENDING: WS-05 product hook mapping is absent; issue=%s; hits=0; transactions/resource-read spy NOT VERIFIED", f.issueID)
	}
}

func TestW54Contract_WS05_UnknownPointFailsBeforeBusiness(t *testing.T) {
	if _, err := w54NewBarrier("publish.after_partial_commit", "unused", "publish", "pause"); err == nil {
		t.Fatal("unknown harness point was silently ignored")
	}
}

var w54KnownPoints = map[string]bool{
	"publish.before_arbitration": true, "claim.before_arbitration": true, "start.before_arbitration": true,
	"publish.before_commit": true, "claim.before_commit": true, "start.before_commit": true,
	"source_validation.read": true, "publish.persist_snapshot": true, "publish.persist_change": true,
	"publish.persist_pointer_adoption": true, "publish.invalidate_queued": true, "publish.commit": true,
	"handoff.persist_binding": true, "handoff.persist_dedup_dispatch": true, "handoff.commit": true,
	"admission.persist_rejection": true, "admission.cancel_queued": true, "admission.commit": true,
	"accept.persist_receipt": true, "accept.make_executable": true, "accept.commit": true,
	"operation.after_commit_before_response": true, "commit.outcome_unknown": true,
}

type w54Barrier struct {
	point, issue, operation, action string
	ready                           chan w54SeamEvent
	release                         chan struct{}
	once                            sync.Once
	mu                              sync.Mutex
	hits                            int
}

func w54NewBarrier(point, issue, operation, action string) (*w54Barrier, error) {
	if !w54KnownPoints[point] {
		return nil, fmt.Errorf("unknown semantic point %q", point)
	}
	return &w54Barrier{point: point, issue: issue, operation: operation, action: action, ready: make(chan w54SeamEvent, 4), release: make(chan struct{})}, nil
}

func (b *w54Barrier) unblock() { b.once.Do(func() { close(b.release) }) }
func (b *w54Barrier) observe(ctx context.Context, e w54SeamEvent) (string, error) {
	if e.Point != b.point || e.IssueID != b.issue || (b.operation != "" && e.Operation != b.operation) {
		return "", nil
	}
	b.mu.Lock()
	b.hits++
	hit := b.hits
	b.mu.Unlock()
	if b.action != "pause" && hit > 1 {
		return "", nil
	}
	select {
	case b.ready <- e:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	switch b.action {
	case "pause":
		select {
		case <-b.release:
			return "", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	case "fail_once":
		return "", errors.New("W54 injected deterministic storage failure")
	case "drop_response", "outcome_unknown":
		return b.action, nil
	default:
		return "", fmt.Errorf("unknown action %q", b.action)
	}
}

func (b *w54Barrier) wait(t *testing.T) w54SeamEvent {
	t.Helper()
	select {
	case e := <-b.ready:
		if e.RequestID == "" || e.Operation == "" {
			t.Errorf("seam lacks request/operation evidence: %+v", e)
		}
		t.Logf("W54_SEAM_HIT %+v", e)
		return e
	case <-time.After(5 * time.Second):
		t.Fatal("LOCAL_PENDING: product semantic point not reached; no ordering/fault claim is valid")
		return w54SeamEvent{}
	}
}

func w54Arm(t *testing.T, point, issue, operation, action string) *w54Barrier {
	t.Helper()
	b, err := w54NewBarrier(point, issue, operation, action)
	if err != nil {
		t.Fatal(err)
	}
	if w54AttachProductSeams == nil {
		t.Fatalf("LOCAL_PENDING: missing actual symbol mapping for %s; hits=0", point)
	}
	detach, err := w54AttachProductSeams(t, b.observe)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		b.unblock()
		detach()
		b.mu.Lock()
		defer b.mu.Unlock()
		t.Logf("W54_SEAM point=%s hits=%d", point, b.hits)
		if b.hits == 0 {
			t.Error("unhit seam cannot verify product behavior")
		}
	})
	return b
}

type w54HTTPResult struct {
	response *http.Response
	err      error
}

func w54Async(t *testing.T, method, path string, body any, token, capabilities string) <-chan w54HTTPResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, method, testServer.URL+path, bytes.NewReader(w54JSON(t, body)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Capabilities", capabilities)
	done := make(chan w54HTTPResult, 1)
	finished := make(chan struct{})
	go func() { defer close(finished); r, e := http.DefaultClient.Do(req); done <- w54HTTPResult{r, e} }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Error("HTTP worker did not stop")
		}
		select {
		case leftover := <-done:
			if leftover.response != nil {
				leftover.response.Body.Close()
			}
		default:
		}
	})
	// Cleanup drains the exact request after releasing its barrier/cancelling its
	// context. No goroutine is handed off to a later turn.
	return done
}
func w54Await(t *testing.T, ch <-chan w54HTTPResult, want ...int) map[string]any {
	t.Helper()
	select {
	case result := <-ch:
		if result.err != nil {
			t.Fatalf("transport outcome unknown: %v", result.err)
		}
		return w54Body(t, result.response, want...)
	case <-time.After(10 * time.Second):
		t.Fatal("request failed to complete after release")
		return nil
	}
}

func (f w54Fixture) facts(t *testing.T) map[string]any {
	t.Helper()
	out := map[string]any{}
	for _, route := range []string{"", "/versions", "/handoffs", "/admissions"} {
		out[route] = w54Body(t, authRequest(t, "GET", f.path()+route, nil), 200)
	}
	var rows string
	f.fx.QueryRow(t, `SELECT COALESCE(jsonb_agg(to_jsonb(q) ORDER BY q.id),'[]'::jsonb)::text FROM agent_task_queue q JOIN agent a ON a.id=q.agent_id WHERE a.workspace_id=$1`, testWorkspaceID).Scan(&rows)
	out["queue_rows"] = rows
	issue := w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)
	out["adopted"] = issue["contract_required"]
	return out
}

func TestW54Contract_WS05_TC06_TwoWritersAtProductArbitration(t *testing.T) {
	f := w54New(t)
	f.unchangedIssue(t)
	f.publish(t, 0, 201)
	b := w54Arm(t, "publish.before_arbitration", f.issueID, "publish", "pause")
	defer b.unblock()
	first := f.putBody(t, 1)
	second := w54Copy(t, first)
	first["goal"] = "Writer A"
	second["goal"] = "Writer B"
	a := w54Async(t, "PUT", f.path(), first, testToken, "")
	c := w54Async(t, "PUT", f.path(), second, testToken, "")
	b.wait(t)
	b.wait(t)
	b.unblock()
	counts := map[int]int{}
	for _, ch := range []<-chan w54HTTPResult{a, c} {
		result := <-ch
		if result.err != nil {
			t.Fatal(result.err)
		}
		counts[result.response.StatusCode]++
		body := w54Body(t, result.response, 200, 409)
		if result.response.StatusCode == 409 && body["code"] != "contract_version_conflict" && body["code"] != "revision_conflict" {
			t.Errorf("unstable conflict: %v", body)
		}
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Errorf("two writers: %v", counts)
	}
	current := w54Body(t, authRequest(t, "GET", f.path(), nil), 200)
	if current["version"] != float64(2) {
		t.Error("more than one version committed")
	}
	versions := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/versions", nil), 200))
	if len(versions) != 2 {
		t.Error("more than one change appended")
	}
}

func TestW54Contract_WS05_TC12_PublishAndClaimOrder(t *testing.T) {
	for _, winner := range []string{"publish", "claim"} {
		t.Run(winner, func(t *testing.T) {
			f := w54Runtime(t, true)
			_, h := f.queued(t)
			id := w54ID(t, h, "task_id")
			f.noNewRuns(t)
			loser := "claim"
			if winner == "claim" {
				loser = "publish"
			}
			b := w54Arm(t, loser+".before_arbitration", f.issueID, loser, "pause")
			defer b.unblock()
			f.snapshot["goal"] = "New approved v2"
			if winner == "publish" {
				waiting := w54Async(t, "POST", "/api/daemon/runtimes/"+f.runtime+"/tasks/claim", map[string]any{}, f.token, "task-contract-v1")
				b.wait(t)
				f.publish(t, 1, 200)
				b.unblock()
				result := w54Await(t, waiting, 200, 409)
				if result["task"] != nil {
					t.Error("old task escaped after publication")
				}
				f.taskState(t, id, "cancelled", false)
				read := w54Body(t, authRequest(t, "GET", f.path()+"/handoffs/"+w54ID(t, h, "handoff_id"), nil), 200)
				if read["state"] != "stale" {
					t.Error("pending handoff not invalidated atomically")
				}
				var reason string
				f.fx.QueryRow(t, "SELECT COALESCE(error,'') FROM agent_task_queue WHERE id=$1", id).Scan(&reason)
				if reason == "" {
					t.Error("cancelled original task lost refusal reason")
				}
			} else {
				waiting := w54Async(t, "PUT", f.path(), f.putBody(t, 1), testToken, "")
				b.wait(t)
				task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
				f.taskState(t, id, "dispatched", false)
				b.unblock()
				result := w54Await(t, waiting, 409)
				if result["code"] != "contract_run_active" {
					t.Errorf("claim did not protect active version: %v", result)
				}
				if task["contract_admission"].(map[string]any)["receipt"] != nil {
					t.Error("claim minted accepted receipt")
				}
				if current := w54Body(t, authRequest(t, "GET", f.path(), nil), 200); current["version"] != float64(1) {
					t.Error("active snapshot changed")
				}
			}
		})
	}
}

func TestW54Contract_WS05_TC14_StartCommitAndCancellation(t *testing.T) {
	f := w54Runtime(t, true)
	_, h := f.queued(t)
	id := w54ID(t, h, "task_id")
	f.noNewRuns(t)
	task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
	b := w54Arm(t, "start.before_commit", f.issueID, "start", "pause")
	defer b.unblock()
	waiting := w54Async(t, "POST", "/api/daemon/tasks/"+id+"/start", w54StartBody(t, task), f.token, "task-contract-v1")
	e := b.wait(t)
	if e.Committed {
		t.Error("before_commit marked committed")
	}
	f.taskState(t, id, "dispatched", false)
	read := w54Body(t, authRequest(t, "GET", f.path()+"/handoffs/"+w54ID(t, h, "handoff_id"), nil), 200)
	if read["receipt"] != nil {
		t.Error("receipt visible before atomic start")
	}
	b.unblock()
	w54Await(t, waiting, 200)
	f.taskState(t, id, "running", true)
	f.snapshot["goal"] = "New goal while active"
	if result := f.publish(t, 1, 409); result["code"] != "contract_run_active" {
		t.Error("running task did not block publish")
	}
	w54Body(t, authRequest(t, "POST", "/api/tasks/"+id+"/cancel", map[string]any{}), 200)
	if result := f.publish(t, 1, 409); result["code"] != "contract_run_active" {
		t.Error("cancellation request mistaken for confirmed termination")
	}
}

func TestW54Contract_WS05_TC15_PublishRollbackAtEveryWrite(t *testing.T) {
	for _, point := range []string{"source_validation.read", "publish.persist_snapshot", "publish.persist_change", "publish.persist_pointer_adoption", "publish.invalidate_queued", "publish.commit"} {
		t.Run(point, func(t *testing.T) {
			f := w54Runtime(t, true)
			f.queued(t)
			f.noNewRuns(t)
			before := f.facts(t)
			b := w54Arm(t, point, f.issueID, "publish", "fail_once")
			f.snapshot["goal"] = "Must not partially publish"
			result := f.publish(t, 1, 503)
			if result["code"] != "validation_unavailable" {
				t.Errorf("wrong storage failure: %v", result)
			}
			b.wait(t)
			if after := f.facts(t); !reflect.DeepEqual(before, after) {
				t.Error("deterministic failure partially committed snapshot/change/adoption/queue/handoff")
			}
		})
	}
}

func TestW54Contract_WS05_TC15_HandoffRollbackAndSameKeyRecovery(t *testing.T) {
	for _, point := range []string{"handoff.persist_binding", "handoff.persist_dedup_dispatch", "handoff.commit"} {
		t.Run(point, func(t *testing.T) {
			f := w54Runtime(t, true)
			f.publish(t, 0, 201)
			b := w54Arm(t, point, f.issueID, "handoff", "fail_once")
			first := f.handoff(t, 1, "same-after-fault", 503)
			if first["code"] != "validation_unavailable" {
				t.Error("fault did not fail closed")
			}
			b.wait(t)
			var count int
			f.fx.QueryRow(t, "SELECT count(*) FROM agent_task_queue WHERE issue_id=$1", f.issueID).Scan(&count)
			if count != 0 {
				t.Error("failed handoff left an orphan run")
			}
			restored := f.handoff(t, 1, "same-after-fault", 200, 201)
			replay := f.handoff(t, 1, "same-after-fault", 200)
			if restored["handoff_id"] != replay["handoff_id"] || restored["task_id"] != replay["task_id"] {
				t.Error("retry created new identity")
			}
			f.fx.QueryRow(t, "SELECT count(*) FROM agent_task_queue WHERE issue_id=$1", f.issueID).Scan(&count)
			if count != 1 {
				t.Errorf("same key created %d runs", count)
			}
		})
	}
}

func TestW54Contract_WS05_TC15_RejectionAtomicWithCancellation(t *testing.T) {
	for _, point := range []string{"admission.persist_rejection", "admission.cancel_queued", "admission.commit"} {
		t.Run(point, func(t *testing.T) {
			f := w54Runtime(t, true)
			_, h := f.queued(t)
			id := w54ID(t, h, "task_id")
			task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
			f.noNewRuns(t)
			b := w54Arm(t, point, f.issueID, "start", "fail_once")
			request := w54StartBody(t, task)
			request["contract_admission"].(map[string]any)["version"] = 2
			failed := w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", request, "task-contract-v1"), 503)
			b.wait(t)
			if failed["code"] != "validation_unavailable" || failed["admission_id"] != nil {
				t.Error("unpersisted rejection presented as durable fact")
			}
			f.taskState(t, id, "dispatched", false)
			rejected := w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", request, "task-contract-v1"), 409)
			w54ID(t, rejected, "admission_id")
			f.taskState(t, id, "cancelled", false)
		})
	}
}

func TestW54Contract_WS05_TC15_AcceptAndExecutableCommitTogether(t *testing.T) {
	for _, point := range []string{"accept.persist_receipt", "accept.make_executable", "accept.commit"} {
		t.Run(point, func(t *testing.T) {
			f := w54Runtime(t, true)
			_, h := f.queued(t)
			id := w54ID(t, h, "task_id")
			task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
			f.noNewRuns(t)
			b := w54Arm(t, point, f.issueID, "start", "fail_once")
			w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", w54StartBody(t, task), "task-contract-v1"), 503)
			b.wait(t)
			f.taskState(t, id, "dispatched", false)
			read := w54Body(t, authRequest(t, "GET", f.path()+"/handoffs/"+w54ID(t, h, "handoff_id"), nil), 200)
			if read["receipt"] != nil {
				t.Error("receipt persisted without executable fact")
			}
			w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", w54StartBody(t, task), "task-contract-v1"), 200)
			f.taskState(t, id, "running", true)
		})
	}
}

func TestW54Contract_WS05_TC15_UnknownOutcomeReconcilesOriginalIdentity(t *testing.T) {
	for _, point := range []string{"operation.after_commit_before_response", "commit.outcome_unknown"} {
		for _, operation := range []string{"publish", "handoff", "start"} {
			t.Run(operation+"/"+point, func(t *testing.T) {
				f := w54Runtime(t, true)
				f.publish(t, 0, 201)
				var task map[string]any
				path, method := f.path(), "PUT"
				body := f.putBody(t, 1)
				body["goal"] = "Explicit reconciled v2"
				token := testToken
				if operation == "handoff" {
					path = f.path() + "/handoffs"
					method = "POST"
					body = map[string]any{"current_version": 1, "target": f.snapshot["next_owner"], "planned_action_ids": []any{"read-input"}, "resources": f.snapshot["allowed_actions"].([]any)[0].(map[string]any)["resources"], "idempotency_key": "unknown-result-same-key"}
				}
				if operation == "start" {
					f.handoff(t, 1, "start-unknown", 201)
					task = w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
					path = "/api/daemon/tasks/" + w54ID(t, task, "id") + "/start"
					method = "POST"
					body = w54StartBody(t, task)
					token = f.token
				}
				action := "drop_response"
				if point == "commit.outcome_unknown" {
					action = "outcome_unknown"
				}
				b := w54Arm(t, point, f.issueID, operation, action)
				first := <-w54Async(t, method, path, body, token, "task-contract-v1")
				b.wait(t)
				if first.response != nil {
					io.Copy(io.Discard, first.response.Body)
					first.response.Body.Close()
					if first.response.StatusCode != 503 {
						t.Errorf("injected unknown returned known HTTP %d", first.response.StatusCode)
					}
				}
				t.Log("W54_OUTCOME UNKNOWN: reconcile original expected version/key/task+generation; do not assert rollback")
				if operation == "publish" {
					current := w54Body(t, authRequest(t, "GET", f.path(), nil), 200)
					if current["version"] == float64(1) {
						w54Body(t, authRequest(t, method, path, body), 200)
					} else if current["version"] != float64(2) || current["goal"] != body["goal"] {
						t.Fatal("unknown publication has inconsistent durable state")
					}
					if list := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/versions", nil), 200)); len(list) != 2 {
						t.Error("unknown publication duplicated version/change")
					}
				} else {
					second := w54Await(t, w54Async(t, method, path, body, token, "task-contract-v1"), 200, 201)
					third := w54Await(t, w54Async(t, method, path, body, token, "task-contract-v1"), 200)
					key := "handoff_id"
					if operation == "start" {
						key = "contract_admission"
					}
					if !reflect.DeepEqual(second[key], third[key]) {
						t.Error("reconciliation changed original identity/receipt")
					}
					var count int
					f.fx.QueryRow(t, "SELECT count(*) FROM agent_task_queue WHERE issue_id=$1", f.issueID).Scan(&count)
					if count != 1 {
						t.Errorf("unknown outcome created %d runs", count)
					}
				}
			})
		}
	}
}

func TestW54Contract_WS05_TC12_UncommittedPublishIsInvisible(t *testing.T) {
	f := w54Runtime(t, true)
	_, h := f.queued(t)
	id := w54ID(t, h, "task_id")
	f.noNewRuns(t)
	before := f.facts(t)
	b := w54Arm(t, "publish.before_commit", f.issueID, "publish", "pause")
	defer b.unblock()
	f.snapshot["goal"] = "Only visible after commit"
	request := w54Async(t, "PUT", f.path(), f.putBody(t, 1), testToken, "")
	e := b.wait(t)
	if e.Committed {
		t.Error("before_commit reported committed")
	}
	if visible := f.facts(t); !reflect.DeepEqual(visible, before) {
		t.Error("uncommitted publication visible via API or DB")
	}
	b.unblock()
	w54Await(t, request, 200)
	f.taskState(t, id, "cancelled", false)
}

func TestW54Contract_WS05_TC12_ClaimCommitProtectsActiveVersion(t *testing.T) {
	f := w54Runtime(t, true)
	_, h := f.queued(t)
	id := w54ID(t, h, "task_id")
	f.noNewRuns(t)
	b := w54Arm(t, "claim.before_commit", f.issueID, "claim", "pause")
	defer b.unblock()
	request := w54Async(t, "POST", "/api/daemon/runtimes/"+f.runtime+"/tasks/claim", map[string]any{}, f.token, "task-contract-v1")
	e := b.wait(t)
	if e.Committed {
		t.Error("claim prematurely committed")
	}
	f.taskState(t, id, "queued", false)
	b.unblock()
	w54Await(t, request, 200)
	f.taskState(t, id, "dispatched", false)
	f.snapshot["goal"] = "Must not cross active claim"
	if result := f.publish(t, 1, 409); result["code"] != "contract_run_active" {
		t.Errorf("claim commit failed to fence publication: %v", result)
	}
}

func TestW54Contract_WS05_TC07_SourceRecheckedAfterClaim(t *testing.T) {
	f := w54Runtime(t, true)
	id := f.fx.Comment(t, f.issueID, "Synthetic approved decision")
	var revision int
	f.fx.QueryRow(t, "SELECT revision FROM comment WHERE id=$1", id).Scan(&revision)
	source := map[string]any{"kind": "comment", "locator": id, "version": revision, "digest": w54Digest("Synthetic approved decision"), "supplied_by": f.snapshot["return_to"]}
	f.snapshot["confirmed_facts"] = []any{map[string]any{"id": "decision1", "assertion": "Current fixed decision applies", "source_refs": []any{source}}}
	_, h := f.queued(t)
	taskID := w54ID(t, h, "task_id")
	f.noNewRuns(t)
	task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
	b := w54Arm(t, "start.before_arbitration", f.issueID, "start", "pause")
	defer b.unblock()
	request := w54Async(t, "POST", "/api/daemon/tasks/"+taskID+"/start", w54StartBody(t, task), f.token, "task-contract-v1")
	b.wait(t)
	w54Body(t, authRequest(t, "PUT", "/api/comments/"+id, map[string]any{"content": "Changed decision before start arbitration"}), 200)
	b.unblock()
	rejected := w54Await(t, request, 409)
	if rejected["code"] != "source_changed" {
		t.Errorf("cached claim approval bypassed source change: %v", rejected)
	}
	f.taskState(t, taskID, "cancelled", false)
}

func TestW54Contract_WS05_TC15_FirstPublishFailureCannotAdopt(t *testing.T) {
	for _, point := range []string{"publish.persist_snapshot", "publish.persist_change", "publish.persist_pointer_adoption", "publish.commit"} {
		t.Run(point, func(t *testing.T) {
			f := w54New(t)
			f.unchangedIssue(t)
			b := w54Arm(t, point, f.issueID, "publish", "fail_once")
			result := f.publish(t, 0, 503)
			b.wait(t)
			if result["code"] != "validation_unavailable" {
				t.Errorf("wrong storage error: %v", result)
			}
			issue := w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)
			if issue["contract_required"] == true {
				t.Error("failed first publication adopted task")
			}
			missing := w54Body(t, authRequest(t, "GET", f.path(), nil), 404)
			if missing["code"] != "contract_missing" {
				t.Error("partial snapshot exists after rollback")
			}
		})
	}
}
