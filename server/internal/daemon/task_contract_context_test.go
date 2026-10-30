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
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func w54ContextDigest(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

// Recompute the specified commitment from the synthetic snapshot, without
// validating it or substituting for any product admission decision.
func w54ContextTask(t *testing.T, task Task, admission, receipt map[string]any) Task {
	t.Helper()
	snapshot := admission["contract"].(map[string]any)
	versions := snapshot["versions"].(map[string]any)
	unique := map[string]map[string]any{}
	var visit func(any)
	visit = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			if v["kind"] != nil && v["locator"] != nil && v["version"] != nil && v["digest"] != nil && v["supplied_by"] != nil {
				source := map[string]any{"kind": v["kind"], "locator": v["locator"], "version": v["version"], "digest": v["digest"]}
				raw, _ := json.Marshal(source)
				unique[string(raw)] = source
			}
			for _, child := range v {
				visit(child)
			}
		case []any:
			for _, child := range v {
				visit(child)
			}
		}
	}
	visit(snapshot)
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	sources := make([]any, 0, len(keys))
	for _, key := range keys {
		sources = append(sources, unique[key])
	}
	code := []any{}
	for _, v := range versions["code_basis"].([]any) {
		basis := v.(map[string]any)
		var dirty any
		if source, ok := basis["dirty_snapshot"].(map[string]any); ok {
			dirty = map[string]any{"kind": source["kind"], "locator": source["locator"], "version": source["version"], "digest": source["digest"]}
		}
		code = append(code, map[string]any{"repository": basis["repository"], "commit_sha": basis["commit_sha"], "dirty_snapshot": dirty})
	}
	issue := versions["issue_basis"].(map[string]any)
	digest := w54ContextDigest(t, map[string]any{"schema_version": 1, "contract_id": snapshot["contract_id"], "version": snapshot["version"], "issue_basis": map[string]any{"title_digest": issue["title_digest"], "description_digest": issue["description_digest"]}, "sources": sources, "code_basis": code})
	admission["input_digest"], receipt["input_digest"] = digest, digest
	admission["code_basis"] = versions["code_basis"]
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err = json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	wire["contract_admission"], wire["contract_required"] = admission, true
	raw, err = json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Task
	if err = json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

type w54ContextProbe struct {
	task                                                   Task
	daemonID, root, marker, input, args, brief, executable string
}

func w54NewContextProbe(t *testing.T) w54ContextProbe {
	t.Helper()
	p := w54ContextProbe{root: t.TempDir(), daemonID: uuid.NewString()}
	p.marker, p.input, p.args, p.brief, p.executable = filepath.Join(p.root, "executions"), filepath.Join(p.root, "input"), filepath.Join(p.root, "args"), filepath.Join(p.root, "brief"), filepath.Join(p.root, "fake-agent")
	writeTestExecutable(t, p.executable, []byte(`#!/bin/sh
printf x >> "$W54_MARKER"
printf '%s\n' "$@" > "$W54_ARGS"
IFS= read -r request
printf '%s' "$request" > "$W54_INPUT"
for file in CLAUDE.md AGENTS.md; do
  if [ -f "$file" ]; then cat "$file" >> "$W54_BRIEF"; fi
done
if [ -n "$W54_SOURCE" ]; then printf changed > "$W54_SOURCE"; fi
printf '%s\n' '{"type":"system","session_id":"synthetic-session"}'
printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"session_id":"synthetic-session","result":"synthetic evidence"}'
`))
	if err := os.Chmod(p.executable, 0700); err != nil {
		t.Fatal(err)
	}
	p.task = Task{ID: uuid.NewString(), AgentID: uuid.NewString(), RuntimeID: uuid.NewString(), IssueID: uuid.NewString(), WorkspaceID: uuid.NewString(), AuthToken: "mat_w54_synthetic_test_only", StartClaimSupported: true, DispatchedAt: "2026-10-30T00:00:02.123456Z"}
	p.task.Agent = &AgentData{ID: p.task.AgentID, Name: "Synthetic", CustomEnv: map[string]string{"W54_MARKER": p.marker, "W54_INPUT": p.input, "W54_ARGS": p.args, "W54_BRIEF": p.brief}}
	return p
}

func (p w54ContextProbe) run(t *testing.T, task Task, handler http.HandlerFunc) error {
	t.Helper()
	server := httptest.NewServer(handler)
	defer server.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{DaemonID: p.daemonID, ServerBaseURL: server.URL, WorkspacesRoot: filepath.Join(p.root, "workspaces"), AgentTimeout: 5 * time.Second, Agents: map[string]AgentEntry{"claude": {Path: p.executable}}}, logger)
	d.executionEnvironmentCommand = nil
	d.runtimeIndex[task.RuntimeID] = Runtime{ID: task.RuntimeID, Provider: "claude"}
	defer d.client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := d.runTask(ctx, task, "claude", 0, logger)
	return err
}

func (p w54ContextProbe) executions(t *testing.T) int {
	t.Helper()
	raw, err := os.ReadFile(p.marker)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return len(raw)
}

func TestW54Runtime_WS03_TC17_CurrentContextAtProcessBoundary(t *testing.T) {
	for _, mode := range []string{"fresh", "resumed", "large_current_contract"} {
		t.Run(mode, func(t *testing.T) {
			p := w54NewContextProbe(t)
			if mode == "resumed" {
				p.task.PriorSessionID = "synthetic-old-session"
				p.task.PriorWorkDir = p.root
			}
			_, admission, receipt := w54ProtectedTask(t, p.task)
			snapshot := admission["contract"].(map[string]any)
			snapshot["goal"] = "CURRENT_GOAL_" + uuid.NewString()
			snapshot["prohibitions"].(map[string]any)["text"] = "CURRENT_PROHIBITION_" + uuid.NewString()
			acceptance := snapshot["acceptance"].([]any)[0].(map[string]any)
			acceptance["assertion"] = "CURRENT_ACCEPTANCE_" + uuid.NewString()
			if mode == "large_current_contract" {
				acceptance["assertion"] = acceptance["assertion"].(string) + strings.Repeat("approved detail ", 8192) + "CURRENT_CONTRACT_TAIL"
			}
			stops := snapshot["stop_conditions"].([]any)
			for i, v := range stops {
				v.(map[string]any)["condition"] = "CURRENT_STOP_" + string(rune('A'+i)) + "_" + uuid.NewString()
			}
			task := w54ContextTask(t, p.task, admission, receipt)
			var starts atomic.Int32
			err := p.run(t, task, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/start") {
					starts.Add(1)
					json.NewEncoder(w).Encode(map[string]any{"contract_admission": map[string]any{"status": "accepted", "receipt": receipt}})
					return
				}
				io.WriteString(w, `{}`)
			})
			// A size refusal must stop before launching the business executable.
			sizeRefusal := err != nil && strings.Contains(strings.ToLower(err.Error()), "contract") && (strings.Contains(strings.ToLower(err.Error()), "size") || strings.Contains(strings.ToLower(err.Error()), "large") || strings.Contains(strings.ToLower(err.Error()), "load"))
			if mode == "large_current_contract" && sizeRefusal && p.executions(t) == 0 {
				t.Log("large current contract refused before local execution")
				return
			}
			if err != nil || p.executions(t) != 1 || starts.Load() != 1 {
				t.Fatalf("process probe err=%v executions=%d starts=%d", err, p.executions(t), starts.Load())
			}
			var collected strings.Builder
			for _, path := range []string{p.input, p.args, p.brief} {
				raw, readErr := os.ReadFile(path)
				if readErr != nil && !os.IsNotExist(readErr) {
					t.Fatal(readErr)
				}
				collected.Write(raw)
			}
			// Normalize JSON escaping in stdin while retaining argv/runtime briefs.
			var input any
			raw, _ := os.ReadFile(p.input)
			if json.Unmarshal(raw, &input) == nil {
				var walk func(any)
				walk = func(v any) {
					switch x := v.(type) {
					case string:
						collected.WriteString(x)
					case map[string]any:
						for _, child := range x {
							walk(child)
						}
					case []any:
						for _, child := range x {
							walk(child)
						}
					}
				}
				walk(input)
			}
			actual := collected.String()
			required := []string{snapshot["goal"].(string), snapshot["prohibitions"].(map[string]any)["text"].(string), acceptance["assertion"].(string), snapshot["next_owner"].(map[string]any)["id"].(string), snapshot["return_to"].(map[string]any)["id"].(string), admission["handoff_id"].(string), admission["input_digest"].(string)}
			for _, stop := range stops {
				required = append(required, stop.(map[string]any)["condition"].(string))
			}
			for i, expected := range required {
				if !strings.Contains(actual, expected) {
					t.Errorf("process startup omitted current contract item=%d bytes=%d", i, len(expected))
				}
			}
			t.Logf("W54_CONTEXT mode=%s executions=%d startup_bytes=%d contract_assertion_bytes=%d", mode, p.executions(t), len(actual), len(acceptance["assertion"].(string)))
		})
	}
}

func TestW54Runtime_WS03_TC08_ActualGitObservationBeforeExecution(t *testing.T) {
	for _, mode := range []string{"clean_accepted", "clean_rejected", "sha_drift", "dirty_without_fixed_evidence", "dirty_fixed_accepted", "dirty_fixed_rejected"} {
		t.Run(mode, func(t *testing.T) {
			p := w54NewContextProbe(t)
			repo := filepath.Join(p.root, "repo")
			if err := os.Mkdir(repo, 0700); err != nil {
				t.Fatal(err)
			}
			realGit, err := exec.LookPath("git")
			if err != nil {
				t.Fatal(err)
			}
			git := func(args ...string) []byte {
				cmd := exec.Command(realGit, append([]string{"-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false"}, args...)...)
				cmd.Dir = repo
				cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Synthetic", "GIT_COMMITTER_NAME=Synthetic", "GIT_AUTHOR_EMAIL=fixture@example.invalid", "GIT_COMMITTER_EMAIL=fixture@example.invalid", "GIT_AUTHOR_DATE=2026-10-30T00:00:00Z", "GIT_COMMITTER_DATE=2026-10-30T00:00:00Z")
				out, e := cmd.CombinedOutput()
				if e != nil {
					t.Fatalf("fixture git %v: %v %s", args, e, out)
				}
				return out
			}
			source := filepath.Join(repo, "input.txt")
			write := func(content string) {
				if err := os.WriteFile(source, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			git("init", "--quiet")
			write("approved synthetic A\r\n")
			git("add", "input.txt")
			git("commit", "--quiet", "-m", "test: 合成批准输入")
			approved := strings.TrimSpace(string(git("rev-parse", "HEAD")))
			repository := "https://example.invalid/synthetic-" + p.task.IssueID + ".git"
			git("remote", "add", "origin", repository)
			if mode == "sha_drift" {
				write("unapproved synthetic B\n")
				git("add", "input.txt")
				git("commit", "--quiet", "-m", "test: 合成版本漂移")
			}
			actualSHA := strings.TrimSpace(string(git("rev-parse", "HEAD")))
			if strings.HasPrefix(mode, "dirty_") {
				write("uncommitted synthetic source\r\n")
			}
			original, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			patch := git("diff", "--binary", "HEAD")
			p.task.ProjectID = uuid.NewString()
			ref, _ := json.Marshal(localDirectoryRef{LocalPath: repo, DaemonID: p.daemonID, ExecutionMode: "in_place"})
			p.task.ProjectResources = []ProjectResourceData{{ID: uuid.NewString(), ResourceType: "local_directory", ResourceRef: ref}}
			p.task.Repos = []RepoData{{URL: repository, Ref: approved}}
			p.task.Agent.CustomEnv["W54_SOURCE"] = source
			_, admission, receipt := w54ProtectedTask(t, p.task)
			snapshot := admission["contract"].(map[string]any)
			versions := snapshot["versions"].(map[string]any)
			delete(versions, "code_not_applicable_reason")
			var dirty any
			artifactID := uuid.NewString()
			if strings.HasPrefix(mode, "dirty_fixed_") {
				sum := sha256.Sum256(patch)
				dirty = map[string]any{"kind": "artifact", "locator": artifactID, "version": artifactID, "digest": hex.EncodeToString(sum[:]), "supplied_by": snapshot["return_to"]}
			}
			versions["code_basis"] = []any{map[string]any{"repository": repository, "commit_sha": approved, "branch": "synthetic-approved", "dirty_snapshot": dirty}}
			task := w54ContextTask(t, p.task, admission, receipt)
			// Spy on the real Git executable; do not fake metadata or results.
			bin := filepath.Join(p.root, "bin")
			if err := os.Mkdir(bin, 0700); err != nil {
				t.Fatal(err)
			}
			gitLog, hookLog := filepath.Join(p.root, "git-commands"), filepath.Join(p.root, "hooks")
			writeTestExecutable(t, filepath.Join(bin, "git"), []byte("#!/bin/sh\nprintf '%s\\0' \"$@\" >> \"$W54_GIT_LOG\"\nprintf '\\n' >> \"$W54_GIT_LOG\"\nexec \"$W54_REAL_GIT\" \"$@\"\n"))
			if err := os.Chmod(filepath.Join(bin, "git"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("W54_GIT_LOG", gitLog)
			t.Setenv("W54_REAL_GIT", realGit)
			t.Setenv("W54_HOOK_LOG", hookLog)
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			for _, name := range []string{"post-checkout", "post-commit", "pre-commit"} {
				if err := os.WriteFile(filepath.Join(repo, ".git", "hooks", name), []byte("#!/bin/sh\nprintf x >> \"$W54_HOOK_LOG\"\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			accepted := strings.HasSuffix(mode, "_accepted")
			var starts, artifactReads, uploads atomic.Int32
			runErr := p.run(t, task, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && strings.Contains(r.URL.Path, "attachments") {
					uploads.Add(1)
				}
				if r.URL.Path == "/api/attachments/"+artifactID+"/content" {
					artifactReads.Add(1)
					w.Write(patch)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/start") {
					starts.Add(1)
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					bound, _ := body["contract_admission"].(map[string]any)
					observations, _ := bound["code_observations"].([]any)
					if len(observations) != 1 {
						t.Errorf("actual Git metadata observation count=%d want=1", len(observations))
					} else {
						observation, _ := observations[0].(map[string]any)
						if observation["repository"] != repository || observation["commit_sha"] != actualSHA {
							t.Errorf("actual Git metadata does not match local repository/SHA: %v", observation)
						}
						if mode == "dirty_without_fixed_evidence" && observation["dirty_snapshot"] == nil {
							t.Error("actual Git metadata falsely reports unapproved dirty source as clean")
						}
						if mode != "dirty_without_fixed_evidence" && !reflect.DeepEqual(observation["dirty_snapshot"], dirty) {
							t.Error("actual Git metadata changed fixed dirty evidence")
						}
					}
					current, readErr := os.ReadFile(source)
					if readErr != nil || string(current) != string(original) {
						t.Error("N: source changed before admission response")
					}
					if accepted {
						json.NewEncoder(w).Encode(map[string]any{"contract_admission": map[string]any{"status": "accepted", "receipt": receipt}})
					} else {
						w.WriteHeader(409)
						io.WriteString(w, `{"code":"input_version_mismatch","error":"synthetic refusal"}`)
					}
					return
				}
				io.WriteString(w, `{}`)
			})
			count := p.executions(t)
			if accepted {
				if runErr != nil || count != 1 || starts.Load() != 1 {
					t.Errorf("accepted metadata control err=%v executions=%d starts=%d", runErr, count, starts.Load())
				}
			} else {
				if runErr == nil || count != 0 {
					t.Errorf("N: rejected metadata err=%v executions=%d", runErr, count)
				}
				after, err := os.ReadFile(source)
				if err != nil || sha256.Sum256(after) != sha256.Sum256(original) {
					t.Error("N: rejected metadata changed synthetic source")
				}
			}
			if got := strings.TrimSpace(string(git("rev-parse", "HEAD"))); got != actualSHA {
				t.Error("metadata observation changed repository HEAD")
			}
			if raw, _ := os.ReadFile(hookLog); len(raw) != 0 {
				t.Error("metadata observation executed project Git hooks")
			}
			commands, _ := os.ReadFile(gitLog)
			for _, arg := range strings.FieldsFunc(string(commands), func(r rune) bool { return r == 0 || r == '\n' }) {
				switch arg {
				case "fetch", "checkout", "switch", "commit", "push", "pull", "reset", "apply", "am":
					t.Errorf("metadata observation invoked mutating Git command %s", arg)
				}
			}
			if uploads.Load() != 0 {
				t.Error("metadata observation uploaded source without admission")
			}
			t.Logf("W54_METADATA mode=%s approved=%s actual=%s starts=%d executions=%d artifact_reads=%d uploads=%d", mode, approved, actualSHA, starts.Load(), count, artifactReads.Load(), uploads.Load())
		})
	}
}
