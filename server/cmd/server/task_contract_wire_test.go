package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/analytics"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/realtime"
)

// The fixture projection only contains ASCII keys and integral numbers below
// 2^53. encoding/json with HTML escaping disabled is JCS for this subset. This
// is an independent expected-value calculator, never an admission validator.
func w54JSON(t *testing.T, v any) []byte {
	t.Helper()
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(v); err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n"))
}

func w54JSONDigest(t *testing.T, v any) string { t.Helper(); return w54Digest(string(w54JSON(t, v))) }

func w54Copy(t *testing.T, v map[string]any) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(w54JSON(t, v), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func (f w54Fixture) putBody(t *testing.T, version int) map[string]any {
	t.Helper()
	b := w54Copy(t, f.snapshot)
	b["expected_version"] = version
	b["expected_issue_revision"] = w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)["revision"]
	return b
}

func w54Field(t *testing.T, b map[string]any, code, path, reason string) {
	t.Helper()
	if b["code"] != code {
		t.Errorf("code=%v want=%s", b["code"], code)
	}
	errors, ok := b["field_errors"].([]any)
	if !ok {
		t.Fatalf("missing field_errors: %v", b)
	}
	for _, v := range errors {
		e := v.(map[string]any)
		if e["path"] == path && e["code"] == code && (reason == "" || e["reason"] == reason) {
			return
		}
	}
	t.Errorf("missing diagnostic %s %s %s: %v", code, path, reason, b)
}

func w54ID(t *testing.T, b map[string]any, key string) string {
	t.Helper()
	s, ok := b[key].(string)
	if !ok || s == "" {
		t.Fatalf("missing durable %s: %v", key, b)
	}
	return s
}

func w54Items(t *testing.T, b map[string]any) []any {
	t.Helper()
	a, ok := b["items"].([]any)
	if !ok {
		t.Fatalf("missing items: %v", b)
	}
	if _, ok := b["next_cursor"]; !ok {
		t.Fatal("missing nullable next_cursor")
	}
	return a
}

func w54InputDigest(t *testing.T, snapshot map[string]any) string {
	t.Helper()
	sources := map[string]map[string]any{}
	var visit func(any)
	visit = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if _, found := x["supplied_by"]; found && x["kind"] != nil && x["digest"] != nil {
				s := map[string]any{"kind": x["kind"], "locator": x["locator"], "version": x["version"], "digest": x["digest"]}
				key := string(w54JSON(t, []any{x["kind"], x["locator"], x["version"]}))
				if old, ok := sources[key]; ok && !reflect.DeepEqual(old, s) {
					t.Fatal("fixture has conflicting Source identities")
				}
				sources[key] = s
			}
			for _, child := range x {
				visit(child)
			}
		case []any:
			for _, child := range x {
				visit(child)
			}
		}
	}
	visit(snapshot)
	ordered := []any{}
	for _, s := range sources {
		ordered = append(ordered, s)
	}
	sort.Slice(ordered, func(i, j int) bool { return bytes.Compare(w54JSON(t, ordered[i]), w54JSON(t, ordered[j])) < 0 })
	versions := snapshot["versions"].(map[string]any)
	basis := versions["issue_basis"].(map[string]any)
	code := []any{}
	for _, raw := range versions["code_basis"].([]any) {
		c := raw.(map[string]any)
		var dirty any
		if s, ok := c["dirty_snapshot"].(map[string]any); ok {
			dirty = map[string]any{"kind": s["kind"], "locator": s["locator"], "version": s["version"], "digest": s["digest"]}
		}
		code = append(code, map[string]any{"repository": c["repository"], "commit_sha": c["commit_sha"], "dirty_snapshot": dirty})
	}
	sort.Slice(code, func(i, j int) bool {
		return code[i].(map[string]any)["repository"].(string) < code[j].(map[string]any)["repository"].(string)
	})
	return w54JSONDigest(t, map[string]any{"schema_version": 1, "contract_id": snapshot["contract_id"], "version": snapshot["version"],
		"issue_basis": map[string]any{"title_digest": basis["title_digest"], "description_digest": basis["description_digest"]}, "sources": ordered, "code_basis": code})
}

func (f w54Fixture) unchangedIssue(t *testing.T) {
	t.Helper()
	f.noNewRuns(t)
	var before string
	query := `SELECT jsonb_build_array(i.status,i.assignee_id,i.assignee_type,(SELECT count(*) FROM comment WHERE workspace_id=$2),(SELECT count(*) FROM inbox_item WHERE workspace_id=$2))::text FROM issue i WHERE i.id=$1`
	f.fx.QueryRow(t, query, f.issueID, testWorkspaceID).Scan(&before)
	t.Cleanup(func() {
		var after string
		f.fx.QueryRow(t, query, f.issueID, testWorkspaceID).Scan(&after)
		if before != after {
			t.Errorf("N: status/assignee/comments/inbox changed: %s -> %s", before, after)
		}
	})
}

func TestW54Contract_WS01_TC01_CompleteSnapshotNoOpCASAndChange(t *testing.T) {
	f := w54New(t)
	f.unchangedIssue(t)
	issue := w54Body(t, authRequest(t, "GET", "/api/issues/"+f.issueID, nil), 200)
	first := f.publish(t, 0, 201)
	for k, v := range f.snapshot {
		if k == "versions" || k == "change" {
			continue
		}
		if !bytes.Equal(w54JSON(t, first[k]), w54JSON(t, v)) {
			t.Errorf("snapshot changed %s", k)
		}
	}
	if first["issue_id"] != f.issueID || first["workspace_id"] != testWorkspaceID || first["contract_required"] != true || first["version"] != float64(1) {
		t.Fatalf("wrong binding: %v", first)
	}
	w54ID(t, first, "contract_id")
	basis := first["versions"].(map[string]any)["issue_basis"].(map[string]any)
	if basis["revision"] != issue["revision"] || basis["title_digest"] != w54JSONDigest(t, issue["title"]) || basis["description_digest"] != w54JSONDigest(t, issue["description"]) {
		t.Errorf("issue_basis not derived from seeded issue: %v", basis)
	}
	for _, k := range []string{"expected_version", "expected_issue_revision", "snapshot", "data"} {
		if _, ok := first[k]; ok {
			t.Errorf("response contains %s", k)
		}
	}
	read := w54Body(t, authRequest(t, "GET", f.path(), nil), 200)
	if !reflect.DeepEqual(first, read) {
		t.Fatal("GET differs from durable PUT result")
	}
	f.snapshot["change"].(map[string]any)["reason"] = "No-op must retain old reason"
	if got := f.publish(t, 1, 200); !reflect.DeepEqual(first, got) {
		t.Error("no-op replaced snapshot/change")
	}
	w54Field(t, f.publish(t, 0, 409), "contract_version_conflict", "/expected_version", "")
	f.snapshot["goal"] = "New approved goal"
	second := f.publish(t, 1, 200)
	change := second["change"].(map[string]any)
	if change["from_version"] != float64(1) || change["to_version"] != float64(2) || !reflect.DeepEqual(change["actor"], f.snapshot["return_to"]) || !reflect.DeepEqual(change["changed_fields"], []any{"/goal"}) {
		t.Errorf("wrong change: %v", change)
	}
	if second["contract_id"] != first["contract_id"] {
		t.Error("contract identity changed")
	}
	history := w54Body(t, authRequest(t, "GET", f.path()+"/versions/1", nil), 200)
	if !reflect.DeepEqual(first, history) {
		t.Error("history rewritten")
	}
	page := w54Body(t, authRequest(t, "GET", f.path()+"/versions?limit=1", nil), 200)
	items := w54Items(t, page)
	if len(items) != 1 || items[0].(map[string]any)["version"] != float64(2) {
		t.Fatalf("bad version ordering: %v", page)
	}
	if _, ok := items[0].(map[string]any)["goal"]; ok {
		t.Error("list leaked historical full snapshot")
	}
	cursor := w54ID(t, page, "next_cursor")
	last := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/versions?limit=1&cursor="+url.QueryEscape(cursor), nil), 200))
	if len(last) != 1 || last[0].(map[string]any)["version"] != float64(1) {
		t.Errorf("unstable cursor: %v", last)
	}
}

func TestW54Contract_WS01_TC02_ReadOnlyAndUnknownFields(t *testing.T) {
	for _, field := range []string{"contract_id", "issue_id", "workspace_id", "version", "contract_required", "snapshot", "data", "unexpected"} {
		t.Run(field, func(t *testing.T) {
			f := w54New(t)
			f.unchangedIssue(t)
			b := f.putBody(t, 0)
			b[field] = 1
			reason := "read_only_field"
			if field == "unexpected" || field == "data" || field == "snapshot" {
				reason = "unknown_field"
			}
			w54Field(t, w54Body(t, authRequest(t, "PUT", f.path(), b), 400), "invalid_request", "/"+field, reason)
		})
	}
	for _, tc := range []struct {
		name, path, reason string
		status             int
		mutate             func(map[string]any)
	}{
		{"readonly_issue_basis", "/versions/issue_basis", "read_only_field", 400, func(s map[string]any) { s["versions"].(map[string]any)["issue_basis"] = map[string]any{} }},
		{"null_expected", "/expected_version", "invalid_type", 400, func(s map[string]any) { s["expected_version"] = nil }},
		{"fractional_version", "/expected_version", "invalid_type", 400, func(s map[string]any) { s["expected_version"] = 0.5 }},
		{"string_reference", "/acceptance/0/source_refs/0", "invalid_type", 400, func(s map[string]any) { s["acceptance"].([]any)[0].(map[string]any)["source_refs"] = []any{"input1"} }},
		{"missing_goal", "/goal", "required", 422, func(s map[string]any) { delete(s, "goal") }},
		{"empty_acceptance", "/acceptance", "required", 422, func(s map[string]any) { s["acceptance"] = []any{} }},
		{"unknown_acceptance", "/allowed_actions/0/acceptance_ids/0", "unresolved_reference", 409, func(s map[string]any) {
			s["allowed_actions"].([]any)[0].(map[string]any)["acceptance_ids"] = []any{"missing-a"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := w54New(t)
			f.unchangedIssue(t)
			b := f.putBody(t, 0)
			tc.mutate(b)
			code := map[int]string{400: "invalid_request", 422: "contract_incomplete", 409: "contract_conflict"}[tc.status]
			w54Field(t, w54Body(t, authRequest(t, "PUT", f.path(), b), tc.status), code, tc.path, tc.reason)
		})
	}
}

func TestW54Contract_WS01_TC02_DuplicateJSONKey(t *testing.T) {
	f := w54New(t)
	f.unchangedIssue(t)
	raw := w54JSON(t, f.putBody(t, 0))
	raw = append([]byte(`{"goal":"duplicate",`), raw[1:]...)
	req, err := http.NewRequest("PUT", testServer.URL+f.path(), bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b := w54Body(t, resp, 400)
	if b["code"] != "invalid_request" {
		t.Errorf("duplicate key accepted: %v", b)
	}
}

func TestW54Contract_WS01_TC04_SourceShapeAndConflict(t *testing.T) {
	for _, field := range []string{"kind", "locator", "version", "digest", "supplied_by"} {
		for _, variant := range []string{"missing", "null"} {
			t.Run(field+"/"+variant, func(t *testing.T) {
				f := w54New(t)
				f.unchangedIssue(t)
				b := f.putBody(t, 0)
				s := b["effective_inputs"].([]any)[0].(map[string]any)["source"].(map[string]any)
				if variant == "missing" {
					delete(s, field)
				} else {
					s[field] = nil
				}
				w54Field(t, w54Body(t, authRequest(t, "PUT", f.path(), b), 400), "invalid_request", "/effective_inputs/0/source/"+field, "")
			})
		}
	}
	t.Run("contradictory_digest", func(t *testing.T) {
		f := w54New(t)
		f.unchangedIssue(t)
		b := f.putBody(t, 0)
		s := w54Copy(t, b["goal_sources"].([]any)[0].(map[string]any))
		s["digest"] = w54Digest("different actual bytes")
		b["confirmed_facts"] = []any{map[string]any{"id": "fact1", "assertion": "Synthetic sourced fact", "source_refs": []any{b["goal_sources"].([]any)[0], s}}}
		w54Field(t, w54Body(t, authRequest(t, "PUT", f.path(), b), 409), "contract_conflict", "/confirmed_facts/0/source_refs/1/digest", "")
	})
}

func TestW54Contract_WS01_TC07_CommentBytesAndRevision(t *testing.T) {
	for _, content := range []string{"abc", "abc\n", "abc\r\n", "\ufeffabc", "e\u0301", "é"} {
		t.Run(fmt.Sprintf("bytes_%x", content), func(t *testing.T) {
			f := w54New(t)
			id := f.fx.Comment(t, f.issueID, content)
			var revision int
			f.fx.QueryRow(t, "SELECT revision FROM comment WHERE id=$1", id).Scan(&revision)
			s := map[string]any{"kind": "comment", "locator": id, "version": revision, "digest": w54Digest(content), "supplied_by": f.snapshot["return_to"]}
			f.snapshot["confirmed_facts"] = []any{map[string]any{"id": "fact1", "assertion": "This fixed decision applies", "source_refs": []any{s}}}
			f.noNewRuns(t)
			f.publish(t, 0, 201)
			f.handoff(t, 1, "before-edit", 201)
			// Editing back to identical text must still invalidate the old revision.
			w54Body(t, authRequest(t, "PUT", "/api/comments/"+id, map[string]any{"content": content + " edited"}), 200)
			w54Body(t, authRequest(t, "PUT", "/api/comments/"+id, map[string]any{"content": content}), 200)
			body := f.handoff(t, 1, "after-edit", 409)
			if body["code"] != "source_changed" {
				t.Errorf("edited decision accepted: %v", body)
			}
		})
	}
}

func TestW54Contract_WS02_TC01_ReceiptAndDurableRead(t *testing.T) {
	f := w54New(t)
	f.unchangedIssue(t)
	snapshot := f.publish(t, 0, 201)
	h := f.handoff(t, 1, "member-wire", 201)
	id := w54ID(t, h, "handoff_id")
	for _, field := range []string{"task_id", "dispatch", "receipt", "latest_admission_id"} {
		v, ok := h[field]
		if !ok || v != nil {
			t.Errorf("pending member %s=%v present=%t", field, v, ok)
		}
	}
	if h["state"] != "pending" {
		t.Errorf("handoff prematurely accepted: %v", h)
	}
	r := w54Body(t, authRequest(t, "POST", f.path()+"/handoffs/"+id+"/accept", map[string]any{"current_version": 1}), 200)
	if r["input_digest"] != w54InputDigest(t, snapshot) {
		t.Errorf("digest not bound to seeded snapshot: %v", r)
	}
	for _, k := range []string{"runtime_id", "task_id", "claim_generation"} {
		v, ok := r[k]
		if !ok || v != nil {
			t.Errorf("member %s not explicitly null", k)
		}
	}
	if !reflect.DeepEqual(r["accepted_by"], f.snapshot["next_owner"]) || r["handoff_id"] != id || !reflect.DeepEqual(r["planned_actions"], h["planned_actions"]) {
		t.Errorf("receipt binding: %v", r)
	}
	replay := w54Body(t, authRequest(t, "POST", f.path()+"/handoffs/"+id+"/accept", map[string]any{"current_version": 1}), 200)
	if !reflect.DeepEqual(r, replay) {
		t.Error("accept replay changed receipt")
	}
	read := w54Body(t, authRequest(t, "GET", f.path()+"/handoffs/"+id, nil), 200)
	if read["state"] != "accepted" || !reflect.DeepEqual(read["receipt"], r) {
		t.Errorf("durable receipt missing: %v", read)
	}
	list := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/handoffs", nil), 200))
	if len(list) != 1 {
		t.Errorf("duplicate handoffs: %v", list)
	}
	admissions := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/admissions?handoff_id="+id, nil), 200))
	if len(admissions) != 1 || admissions[0].(map[string]any)["outcome"] != "accepted" || admissions[0].(map[string]any)["operation"] != "member_accept" {
		t.Fatalf("accept admission: %v", admissions)
	}
	// Rebuild the real router against the same isolated DB.
	// This is service reconstruction, not an OS-process restart claim.
	router := NewRouter(testPool, realtime.NewHub(), events.New(), analytics.NoopClient{}, nil)
	restarted := httptest.NewServer(router)
	defer restarted.Close()
	req, _ := http.NewRequest("GET", restarted.URL+f.path()+"/handoffs/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if got := w54Body(t, resp, 200); !reflect.DeepEqual(got, read) {
		t.Error("fresh router lost persisted receipt")
	}
}

func TestW54Contract_WS02_TC11_RejectionBeforeHandoffIsQueryableAndReplayStable(t *testing.T) {
	f := w54New(t)
	f.unchangedIssue(t)
	f.publish(t, 0, 201)
	b := map[string]any{"current_version": 1, "planned_action_ids": []any{"unapproved-edit"}, "resources": f.snapshot["allowed_actions"].([]any)[0].(map[string]any)["resources"], "target": f.snapshot["next_owner"], "idempotency_key": "durable-refusal"}
	first := w54Body(t, authRequest(t, "POST", f.path()+"/handoffs", b), 409)
	w54Field(t, first, "contract_out_of_scope", "/planned_action_ids/0", "")
	id := w54ID(t, first, "admission_id")
	for i := 0; i < 2; i++ {
		got := w54Body(t, authRequest(t, "POST", f.path()+"/handoffs", b), 409)
		if got["admission_id"] != id {
			t.Error("refusal replay duplicated admission")
		}
	}
	a := w54Body(t, authRequest(t, "GET", f.path()+"/admissions/"+id, nil), 200)
	for _, k := range []string{"admission_id", "issue_id", "request_id", "operation", "outcome", "code", "handoff_state", "field_errors", "expected_version", "checked_version", "current_version", "checked_at", "return_to", "handoff_id", "task_id", "claim_generation", "runtime_id", "input_digest", "planned_actions", "code_observations"} {
		if _, ok := a[k]; !ok {
			t.Errorf("admission missing %s", k)
		}
	}
	if a["outcome"] != "rejected" || a["operation"] != "handoff" || a["handoff_id"] != nil || a["task_id"] != nil {
		t.Errorf("wrong pre-handoff rejection: %v", a)
	}
	if list := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/admissions", nil), 200)); len(list) != 1 {
		t.Errorf("duplicate rejection: %v", list)
	}
	if list := w54Items(t, w54Body(t, authRequest(t, "GET", f.path()+"/handoffs", nil), 200)); len(list) != 0 {
		t.Errorf("rejection created handoff: %v", list)
	}
}

func TestW54Contract_WS02_TC10_AcceptRejectsForgedBinding(t *testing.T) {
	for _, field := range []string{"task_id", "runtime_id", "claim_generation", "accepted_at", "receipt"} {
		t.Run(field, func(t *testing.T) {
			f := w54New(t)
			f.unchangedIssue(t)
			f.publish(t, 0, 201)
			h := f.handoff(t, 1, "forged", 201)
			b := map[string]any{"current_version": 1, field: "forged"}
			w54Field(t, w54Body(t, authRequest(t, "POST", f.path()+"/handoffs/"+w54ID(t, h, "handoff_id")+"/accept", b), 400), "invalid_request", "/"+field, "")
		})
	}
}

func TestW54Contract_WS01_TC09_InvalidRepositoryPaths(t *testing.T) {
	for _, path := range []string{"", "/absolute", "src/../private", "src/./file", "src//file", "src/", "src\\file", "src/*", "src/\x00file"} {
		t.Run(fmt.Sprintf("path_%x", path), func(t *testing.T) {
			f := w54New(t)
			f.unchangedIssue(t)
			f.snapshot["allowed_actions"].([]any)[0].(map[string]any)["resources"] = []any{map[string]any{"kind": "repository", "locator": map[string]any{"repository": "https://example.invalid/synthetic.git", "path": path}, "match": "exact"}}
			w54Field(t, f.publish(t, 0, 400), "invalid_request", "/allowed_actions/0/resources/0/locator/path", "")
		})
	}
}

func TestW54Contract_WS04_TC10_ReadProjectionRevocation(t *testing.T) {
	f := w54New(t)
	f.unchangedIssue(t)
	f.publish(t, 0, 201)
	h := f.handoff(t, 1, "private-projection", 201)
	member := f.fx.User(t, "Synthetic projection reader", "projection@example.invalid")
	membership := f.fx.Member(t, testWorkspaceID, member, "member")
	token, err := generateTestJWT(member, "projection@example.invalid", "Synthetic projection reader")
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{f.path(), f.path() + "/versions", f.path() + "/versions/1", f.path() + "/handoffs", f.path() + "/handoffs/" + w54ID(t, h, "handoff_id"), f.path() + "/admissions"}
	for _, path := range paths {
		w54Body(t, w54As(t, token, testWorkspaceID, "GET", path, nil), 200)
	}
	f.fx.Exec(t, "DELETE FROM member WHERE id=$1", membership)
	for _, path := range paths {
		b := w54Body(t, w54As(t, token, testWorkspaceID, "GET", path, nil), 403, 404)
		for _, field := range []string{"items", "goal", "target", "return_to", "version", "current_version", "receipt"} {
			if _, ok := b[field]; ok {
				t.Errorf("revoked read leaked %s", field)
			}
		}
		if strings.Contains(string(w54JSON(t, b)), "Synthetic approved requirement") {
			t.Error("private source content leaked")
		}
	}
}
