package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Seed a complete synthetic server claim, then decode through the real Task
// type. Unknown wire fields must not silently downgrade an adopted task.
func w54ProtectedTask(t *testing.T, task Task) (Task, map[string]any, map[string]any) {
	t.Helper()
	digest := func(v any) string {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(raw)
		return hex.EncodeToString(h[:])
	}
	member := map[string]any{"type": "member", "id": uuid.NewString()}
	agent := map[string]any{"type": "agent", "id": task.AgentID}
	source := map[string]any{"kind": "issue", "locator": task.IssueID, "version": 7, "digest": digest(map[string]any{"title": "Synthetic task", "description": nil}), "supplied_by": member}
	resource := map[string]any{"kind": "issue", "locator": task.IssueID, "match": "exact"}
	plan := []any{map[string]any{"action_id": "read-input", "operation": "read", "resources": []any{resource}}}
	issueBasis := map[string]any{"revision": 7, "title_digest": digest("Synthetic task"), "description_digest": digest(nil)}
	stops := []any{}
	for _, code := range []string{"input_unavailable", "input_version_mismatch", "contract_out_of_scope", "permission_denied", "blocking_question"} {
		stops = append(stops, map[string]any{"code": code, "condition": "Stop and return to the owner", "return_to": member})
	}
	cid, hid := uuid.NewString(), uuid.NewString()
	snapshot := map[string]any{"schema_version": 1, "contract_id": cid, "issue_id": task.IssueID, "workspace_id": task.WorkspaceID, "version": 1, "contract_required": true,
		"goal": "Read approved synthetic input", "goal_sources": []any{source},
		"allowed_actions":  []any{map[string]any{"id": "read-input", "operation": "read", "resources": []any{resource}, "acceptance_ids": []any{"a1"}}},
		"prohibitions":     map[string]any{"text": "Do not alter product source", "denied_actions": []any{}},
		"acceptance":       []any{map[string]any{"id": "a1", "assertion": "Evidence identifies the fixed source", "evidence_required": []any{"test-log"}, "source_refs": []any{source}}},
		"effective_inputs": []any{map[string]any{"id": "input1", "source": source, "purpose": "Main requirement", "required": true}}, "confirmed_facts": []any{}, "assumptions": []any{},
		"versions":     map[string]any{"code_basis": []any{}, "code_not_applicable_reason": "Read synthetic issue only", "document_basis": []any{map[string]any{"source": source, "applicability": "Main requirement"}}, "issue_basis": issueBasis},
		"deliverables": []any{map[string]any{"id": "d1", "description": "Version evidence", "acceptance_ids": []any{"a1"}, "evidence_refs": []any{}}}, "open_questions": []any{}, "stop_conditions": stops, "next_owner": agent, "return_to": member,
		"decision_sources": []any{map[string]any{"source": source, "actor": member, "decided_at": "2026-10-29T00:00:00Z"}},
		"change":           map[string]any{"reason": "Create synthetic contract", "source_refs": []any{source}, "actor": member, "recorded_at": "2026-10-29T00:00:01Z", "from_version": 0, "to_version": 1, "changed_fields": []any{"/goal"}}}
	inputDigest := digest(map[string]any{"schema_version": 1, "contract_id": cid, "version": 1, "issue_basis": map[string]any{"title_digest": issueBasis["title_digest"], "description_digest": issueBasis["description_digest"]}, "sources": []any{map[string]any{"kind": source["kind"], "locator": source["locator"], "version": source["version"], "digest": source["digest"]}}, "code_basis": []any{}})
	admission := map[string]any{"status": "pending", "contract": snapshot, "handoff_id": hid, "version": 1, "input_digest": inputDigest, "planned_actions": plan, "code_basis": []any{}, "runtime_id": task.RuntimeID, "task_id": task.ID, "claim_generation": task.DispatchedAt, "receipt": nil}
	receipt := map[string]any{"handoff_id": hid, "version": 1, "target": agent, "task_id": task.ID, "runtime_id": task.RuntimeID, "claim_generation": task.DispatchedAt, "input_digest": inputDigest, "planned_actions": plan, "accepted_by": agent, "accepted_at": "2026-10-29T00:00:03Z"}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err = json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	wire["contract_admission"] = admission
	wire["contract_required"] = true
	raw, err = json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Task
	if err = json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	// Round trip expected objects to JSON numeric/array types used by the wire.
	raw, _ = json.Marshal(admission)
	json.Unmarshal(raw, &admission)
	raw, _ = json.Marshal(receipt)
	json.Unmarshal(raw, &receipt)
	return decoded, admission, receipt
}

func TestW54Runtime_WS03_TC17_Protected200RequiresBoundReceipt(t *testing.T) {
	for _, mode := range []string{"valid_control", "legacy_empty", "missing_admission", "missing_receipt", "null_receipt", "pending", "unknown_status", "wrong_handoff_id", "wrong_version", "wrong_task_id", "wrong_runtime_id", "wrong_claim_generation", "wrong_input_digest", "wrong_planned_actions", "wrong_target", "wrong_accepted_by", "missing_accepted_at", "resume_legacy_empty", "resume_wrong_version"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, "executions")
			source := filepath.Join(root, "source.txt")
			original := []byte("synthetic original\n")
			if err := os.WriteFile(source, original, 0600); err != nil {
				t.Fatal(err)
			}
			fake := filepath.Join(root, "fake-agent")
			writeTestExecutable(t, fake, []byte(`#!/bin/sh
printf x >> "$W54_PROCESS_MARKER"
printf changed > "$W54_SOURCE"
IFS= read -r _
printf '%s\n' '{"type":"system","session_id":"synthetic-session"}'
printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"session_id":"synthetic-session","result":"synthetic evidence"}'
`))
			if err := os.Chmod(fake, 0700); err != nil {
				t.Fatal(err)
			}
			task := Task{ID: uuid.NewString(), AgentID: uuid.NewString(), RuntimeID: uuid.NewString(), IssueID: uuid.NewString(), WorkspaceID: uuid.NewString(), AuthToken: "mat_w54_synthetic_test_only", StartClaimSupported: true, DispatchedAt: "2026-10-29T00:00:02.123456Z"}
			task.Agent = &AgentData{ID: task.AgentID, Name: "Synthetic", CustomEnv: map[string]string{"W54_PROCESS_MARKER": marker, "W54_SOURCE": source}}
			if strings.HasPrefix(mode, "resume_") {
				task.PriorSessionID = "synthetic-prior"
				task.PriorWorkDir = root
			}
			task, claim, receipt := w54ProtectedTask(t, task)
			admission := map[string]any{"status": "accepted", "receipt": receipt}
			response := map[string]any{"contract_admission": admission}
			variant := strings.TrimPrefix(mode, "resume_")
			switch variant {
			case "missing_admission":
				delete(response, "contract_admission")
			case "missing_receipt":
				delete(admission, "receipt")
			case "null_receipt":
				admission["receipt"] = nil
			case "pending":
				admission["status"] = "pending"
			case "unknown_status":
				admission["status"] = "future_success"
			case "wrong_version":
				receipt["version"] = float64(2)
			case "wrong_planned_actions":
				receipt["planned_actions"] = []any{}
			case "wrong_target", "wrong_accepted_by":
				receipt[strings.TrimPrefix(variant, "wrong_")] = map[string]any{"type": "agent", "id": uuid.NewString()}
			case "missing_accepted_at":
				delete(receipt, "accepted_at")
			default:
				if strings.HasPrefix(variant, "wrong_") {
					receipt[strings.TrimPrefix(variant, "wrong_")] = "different-binding"
				}
			}
			var starts, preReads, afterRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/start") {
					starts.Add(1)
					var actual map[string]any
					if err := json.NewDecoder(r.Body).Decode(&actual); err != nil {
						t.Error(err)
					}
					// Inspect the real client's request; no test-built start substitute.
					if actual["runtime_id"] != task.RuntimeID || actual["dispatched_at"] != task.DispatchedAt {
						t.Error("start lost runtime/generation")
					}
					if mode == "valid_control" {
						caps, _ := actual["capabilities"].([]any)
						found := false
						for _, c := range caps {
							found = found || c == "task-contract-v1"
						}
						if !found {
							t.Error("actual start lacks task-contract-v1 capability")
						}
						bound, ok := actual["contract_admission"].(map[string]any)
						if !ok {
							t.Error("actual start omitted protected admission")
						} else {
							for _, key := range []string{"handoff_id", "version", "input_digest", "planned_actions"} {
								if !reflect.DeepEqual(bound[key], claim[key]) {
									t.Errorf("actual start changed %s", key)
								}
							}
							if !reflect.DeepEqual(bound["code_observations"], []any{}) {
								t.Error("missing explicit empty code_observations")
							}
						}
					}
					if variant != "legacy_empty" {
						json.NewEncoder(w).Encode(response)
					}
					return
				}
				if starts.Load() == 0 {
					preReads.Add(1)
				} else {
					afterRequests.Add(1)
				}
				io.WriteString(w, `{}`)
			}))
			defer server.Close()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			d := &Daemon{client: NewClient(server.URL), logger: logger, workspaces: map[string]*workspaceState{}, activeEnvRoots: map[string]int{}, runtimeIndex: map[string]Runtime{task.RuntimeID: {ID: task.RuntimeID, Provider: "claude"}}, cfg: Config{WorkspacesRoot: filepath.Join(root, "workspaces"), AgentTimeout: 5 * time.Second, ServerBaseURL: server.URL, Agents: map[string]AgentEntry{"claude": {Path: fake}}}}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := d.runTask(ctx, task, "claude", 0, logger)
			if starts.Load() != 1 {
				t.Errorf("start attempts=%d want=1", starts.Load())
			}
			after, readErr := os.ReadFile(source)
			if readErr != nil {
				t.Fatal(readErr)
			}
			executions, markerErr := os.ReadFile(marker)
			if mode == "valid_control" {
				if err != nil || markerErr != nil || string(executions) != "x" || string(after) != "changed" {
					t.Errorf("positive probe run=%v marker=%v executions=%q source=%q", err, markerErr, executions, after)
				}
				return
			}
			if err == nil {
				t.Error("protected task accepted unbound HTTP 200")
			}
			if !os.IsNotExist(markerErr) {
				t.Errorf("N: business process executed %d time(s)", len(executions))
			}
			if sha256.Sum256(original) != sha256.Sum256(after) {
				t.Error("N: synthetic source changed without bound receipt")
			}
			if preReads.Load() != 0 || afterRequests.Load() != 0 {
				t.Errorf("N: resource/post-start requests=%d/%d", preReads.Load(), afterRequests.Load())
			}
			t.Logf("W54_LOCAL task=%s runtime=%s generation=%s starts=%d executions=%d", task.ID, task.RuntimeID, task.DispatchedAt, starts.Load(), len(executions))
		})
	}
}
