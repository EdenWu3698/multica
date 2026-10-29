package main

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/multica-ai/multica/server/internal/analytics"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/internal/storage"
	"github.com/multica-ai/multica/server/internal/testutil"
)

// Only the object-store I/O boundary is observed/faulted. Authorization,
// contract validation, transactions and API handlers remain real.
type w54ReadSpy struct {
	storage.Storage
	mu        sync.Mutex
	reads     map[string]int
	failKey   string
	faultHits int
}

func (s *w54ReadSpy) GetReader(ctx context.Context, key string) (io.ReadCloser, error) {
	s.mu.Lock()
	s.reads[key]++
	fail := s.failKey == key
	if fail {
		s.faultHits++
		s.failKey = ""
	}
	s.mu.Unlock()
	if fail {
		return nil, errors.New("W54 injected source storage failure")
	}
	return s.Storage.GetReader(ctx, key)
}
func (s *w54ReadSpy) count(key string) int { s.mu.Lock(); defer s.mu.Unlock(); return s.reads[key] }

func w54WithStorage(t *testing.T) *w54ReadSpy {
	t.Helper()
	t.Setenv("LOCAL_UPLOAD_DIR", t.TempDir())
	t.Setenv("LOCAL_UPLOAD_BASE_URL", "http://127.0.0.1")
	local := storage.NewLocalStorageFromEnv()
	if local == nil {
		t.Fatal("local synthetic object store unavailable")
	}
	spy := &w54ReadSpy{Storage: local, reads: map[string]int{}}
	hub, bus := realtime.NewHub(), events.New()
	registerListeners(bus, hub)
	router, h := NewRouterWithOptions(testPool, hub, bus, analytics.NoopClient{}, nil, RouterOptions{})
	h.Storage = spy
	h.TaskService.SourceContextStorage = spy
	previous := testServer
	server := httptest.NewServer(router)
	testServer = server
	t.Cleanup(func() { testServer = previous; server.Close() })
	return spy
}

func w54Artifact(t *testing.T, f w54Fixture, spy *w54ReadSpy, content []byte, workspace string) (map[string]any, string) {
	t.Helper()
	key := "w54/" + f.issueID + "/" + w54Digest(string(content)) + ".txt"
	location, err := spy.Upload(context.Background(), key, content, "text/plain", "synthetic.txt")
	if err != nil {
		t.Fatal(err)
	}
	id := f.fx.Insert(t, "attachment", testutil.Cols{"workspace_id": workspace, "issue_id": f.issueID, "uploader_type": "member", "uploader_id": testUserID, "filename": "synthetic.txt", "url": location, "content_type": "text/plain", "size_bytes": len(content)})
	source := map[string]any{"kind": "artifact", "locator": id, "version": id, "digest": w54Digest(string(content)), "supplied_by": f.snapshot["return_to"]}
	return source, key
}

func TestW54Contract_WS04_TC10_ResourceReadSpyBaseline(t *testing.T) {
	spy := w54WithStorage(t)
	f := w54New(t)
	content := []byte("synthetic attachment bytes\r\n")
	s, key := w54Artifact(t, f, spy, content, testWorkspaceID)
	response := authRequest(t, "GET", "/api/attachments/"+s["locator"].(string)+"/content", nil)
	raw, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || string(raw) != string(content) || spy.count(key) == 0 {
		t.Fatalf("read spy positive control failed: status=%d reads=%d body=%q", response.StatusCode, spy.count(key), raw)
	}
	other := f.fx.Workspace(t, "Synthetic private resource workspace", "w54-private-"+f.issueID)
	private, privateKey := w54Artifact(t, f, spy, []byte("private synthetic source"), other)
	response = authRequest(t, "GET", "/api/attachments/"+private["locator"].(string)+"/content", nil)
	io.Copy(io.Discard, response.Body)
	response.Body.Close()
	if response.StatusCode != 403 && response.StatusCode != 404 {
		t.Errorf("foreign resource status=%d", response.StatusCode)
	}
	if spy.count(privateKey) != 0 {
		t.Error("unauthorized resource body read")
	}
	t.Logf("W54_READ_SPY authorized_reads=%d unauthorized_reads=%d", spy.count(key), spy.count(privateKey))
}

func TestW54Contract_WS01_TC04_ArtifactAvailabilityAndBytes(t *testing.T) {
	for _, mode := range []string{"valid", "digest_mismatch", "deleted", "cross_workspace", "storage_fault"} {
		t.Run(mode, func(t *testing.T) {
			spy := w54WithStorage(t)
			f := w54New(t)
			workspace := testWorkspaceID
			if mode == "cross_workspace" {
				workspace = f.fx.Workspace(t, "Synthetic foreign input", "w54-foreign-"+f.issueID)
			}
			s, key := w54Artifact(t, f, spy, []byte("abc\r\n"), workspace)
			f.snapshot["effective_inputs"] = append(f.snapshot["effective_inputs"].([]any), map[string]any{"id": "artifact-input", "source": s, "purpose": "Fixed supplemental evidence", "required": true})
			if mode == "digest_mismatch" {
				s["digest"] = w54Digest("abc\n")
			}
			if mode == "deleted" {
				spy.Delete(context.Background(), key)
			}
			if mode == "storage_fault" {
				spy.failKey = key
			}
			f.unchangedIssue(t)
			status, code := 201, ""
			switch mode {
			case "digest_mismatch":
				status, code = 409, "source_changed"
			case "deleted", "cross_workspace":
				status, code = 422, "input_unavailable"
			case "storage_fault":
				status, code = 503, "validation_unavailable"
			}
			result := f.publish(t, 0, status)
			if code != "" {
				if result["code"] != code {
					t.Errorf("artifact refusal=%v want=%s", result, code)
				}
			} else {
				f.handoff(t, 1, "artifact-valid", 201)
			}
			if mode == "cross_workspace" {
				if spy.count(key) != 0 {
					t.Error("N: foreign attachment bytes read")
				}
			} else if spy.count(key) == 0 {
				t.Error("resource validation never read actual artifact bytes")
			}
			if mode == "storage_fault" && spy.faultHits != 1 {
				t.Errorf("fault hits=%d; unhit fault is not verified", spy.faultHits)
			}
			t.Logf("W54_SOURCE mode=%s reads=%d fault_hits=%d", mode, spy.count(key), spy.faultHits)
		})
	}
}

func w54GitCorpus(t *testing.T) (string, string, []byte) {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) []byte {
		cmd := exec.Command("git", append([]string{"-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Synthetic Fixture", "GIT_COMMITTER_NAME=Synthetic Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid", "GIT_COMMITTER_EMAIL=fixture@example.invalid", "GIT_AUTHOR_DATE=2026-10-29T00:00:00Z", "GIT_COMMITTER_DATE=2026-10-29T00:00:00Z")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("synthetic Git %v: %v %s", args, err, out)
		}
		return out
	}
	write := func(content string) {
		if err := os.WriteFile(filepath.Join(root, "input.txt"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "--quiet")
	write("approved A\n")
	git("add", "input.txt")
	git("commit", "--quiet", "-m", "test: 合成版本 A")
	a := strings.TrimSpace(string(git("rev-parse", "HEAD")))
	write("unapproved B\n")
	git("add", "input.txt")
	git("commit", "--quiet", "-m", "test: 合成版本 B")
	b := strings.TrimSpace(string(git("rev-parse", "HEAD")))
	write("uncommitted synthetic input\n")
	patch := git("diff", "--binary", a)
	if a == b || len(patch) == 0 {
		t.Fatal("synthetic Git corpus missing drift")
	}
	return a, b, patch
}

func TestW54Contract_WS03_TC08_CodeObservationMatchesActualSeed(t *testing.T) {
	for _, mode := range []string{"clean_valid", "sha_drift", "missing_repository", "extra_repository", "duplicate_repository", "dirty_valid", "dirty_missing", "dirty_mismatch"} {
		t.Run(mode, func(t *testing.T) {
			spy := w54WithStorage(t)
			f := w54Runtime(t, true)
			a, b, patch := w54GitCorpus(t)
			t.Logf("W54_CODE_SEED actual_A=%s actual_B=%s dirty_patch_sha256=%s", a, b, w54Digest(string(patch)))
			repository := "https://example.invalid/synthetic-" + f.issueID + ".git"
			var dirty any
			if strings.HasPrefix(mode, "dirty_") {
				source, _ := w54Artifact(t, f.w54Fixture, spy, patch, testWorkspaceID)
				dirty = source
			}
			versions := f.snapshot["versions"].(map[string]any)
			delete(versions, "code_not_applicable_reason")
			versions["code_basis"] = []any{map[string]any{"repository": repository, "commit_sha": a, "branch": "fixture-A", "dirty_snapshot": dirty}}
			_, h := f.queued(t)
			id := w54ID(t, h, "task_id")
			f.noNewRuns(t)
			task := w54Claimed(t, f.claim(t, "single", "task-contract-v1"), "single")
			body := w54StartBody(t, task)
			observation := map[string]any{"repository": repository, "commit_sha": a, "dirty_snapshot": dirty}
			observations := []any{observation}
			status, code, path := 200, "", ""
			switch mode {
			case "sha_drift":
				observation["commit_sha"] = b
				status, code, path = 409, "input_version_mismatch", "/contract_admission/code_observations/0/commit_sha"
			case "missing_repository":
				observations = []any{}
				status, code, path = 409, "input_version_mismatch", "/contract_admission/code_observations"
			case "extra_repository":
				extra := w54Copy(t, observation)
				extra["repository"] = "https://example.invalid/extra.git"
				observations = append(observations, extra)
				status, code, path = 409, "input_version_mismatch", "/contract_admission/code_observations/1/repository"
			case "duplicate_repository":
				observations = append(observations, observation)
				status, code, path = 409, "input_version_mismatch", "/contract_admission/code_observations/1/repository"
			case "dirty_missing":
				observation["dirty_snapshot"] = nil
				status, code, path = 409, "input_version_mismatch", "/contract_admission/code_observations/0/dirty_snapshot"
			case "dirty_mismatch":
				changed := w54Copy(t, dirty.(map[string]any))
				changed["digest"] = w54Digest("different patch bytes")
				observation["dirty_snapshot"] = changed
				status, code, path = 409, "input_version_mismatch", "/contract_admission/code_observations/0/dirty_snapshot"
			}
			body["contract_admission"].(map[string]any)["code_observations"] = observations
			result := w54Body(t, f.request(t, "/api/daemon/tasks/"+id+"/start", body, "task-contract-v1"), status)
			if status == 200 {
				f.taskState(t, id, "running", true)
			} else {
				w54Field(t, result, code, path, "")
				f.taskState(t, id, "cancelled", false)
			}
			t.Logf("W54_CODE actual_A=%s actual_B=%s dirty_patch_sha256=%s", a, b, w54Digest(string(patch)))
		})
	}
}
