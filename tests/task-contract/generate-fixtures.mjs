import { createHash, randomUUID } from "node:crypto";
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve, join } from "node:path";

const destination = resolve(process.argv[2] ?? "tests/task-contract/generated");
mkdirSync(destination, { recursive: false });
const repo = join(destination, "synthetic-repo");
mkdirSync(repo);
const git = (...args) => execFileSync("git", args, { cwd: repo, encoding: "utf8", env: {
  ...process.env, GIT_AUTHOR_NAME: "Synthetic Fixture", GIT_COMMITTER_NAME: "Synthetic Fixture",
  GIT_AUTHOR_EMAIL: "fixture@example.invalid", GIT_COMMITTER_EMAIL: "fixture@example.invalid",
  GIT_AUTHOR_DATE: "2026-10-29T00:00:00Z", GIT_COMMITTER_DATE: "2026-10-29T00:00:00Z",
}}).trim();
const digest = (text) => createHash("sha256").update(text).digest("hex");
git("init", "--quiet");
mkdirSync(join(repo, "src")); mkdirSync(join(repo, "docs"));
writeFileSync(join(repo, "src", "synthetic.txt"), "synthetic source A\n");
const d1 = "Synthetic requirement D1: read the approved input.\n";
const d2 = "Synthetic requirement D2: include source digest in evidence.\n";
writeFileSync(join(repo, "docs", "requirement.md"), d1);
git("add", "."); git("-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "test: 合成基线 A");
const commitA = git("rev-parse", "HEAD");
writeFileSync(join(repo, "src", "synthetic.txt"), "synthetic source B\n");
writeFileSync(join(repo, "docs", "requirement.md"), d2);
git("add", "."); git("-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "test: 合成基线 B");
const commitB = git("rev-parse", "HEAD");
writeFileSync(join(repo, "src", "synthetic.txt"), "synthetic source B plus uncommitted change\n");
const patch = git("diff", "--binary") + "\n";
writeFileSync(join(destination, "dirty.patch"), patch);
const ids = Object.fromEntries(["w1", "w2", "initiator", "receiver", "returnTo", "outsider", "callableAgent", "privateAgent", "runtime", "legacyRuntime", "ordinaryIssue", "protectedIssue", "comment", "readableAttachment", "expiredAttachment"].map((name) => [name, randomUUID()]));
const manifest = {
  spec: "W54-SPEC-0.1", generatedAt: "2026-10-29T00:00:00Z", ids,
  identities: [
    { id: ids.initiator, workspace: ids.w1, kind: "member" },
    { id: ids.receiver, workspace: ids.w1, kind: "member" },
    { id: ids.returnTo, workspace: ids.w1, kind: "member" },
    { id: ids.outsider, workspace: ids.w2, kind: "member" },
  ],
  agents: [
    { id: ids.callableAgent, workspace: ids.w1, callable: true, executable: "TEST_CREATED_FAKE_ONLY" },
    { id: ids.privateAgent, workspace: ids.w1, callable: false, executable: "TEST_CREATED_FAKE_ONLY" },
  ],
  runtimes: [{ id: ids.runtime, protocol: "task-contract-v1" }, { id: ids.legacyRuntime, protocol: "legacy" }],
  code: { commitA, commitB, dirtyPatch: { file: "dirty.patch", sha256: digest(patch) } },
  documents: [{ id: "D1", path: "docs/requirement.md", commit: commitA, sha256: digest(d1) }, { id: "D2", path: "docs/requirement.md", commit: commitB, sha256: digest(d2) }],
  comments: [{ id: ids.comment, revision: 1, content: "Synthetic approved decision C1", sha256: digest("Synthetic approved decision C1") }],
  attachments: [{ id: ids.readableAttachment, available: true }, { id: ids.expiredAttachment, available: false }],
  outOfScopePlans: ["unknown-operation", "unapproved-action", "../outside", "/absolute", "src/../private", "denied-overlap", "wrong-receiver"],
  history: Array.from({ length: 200 }, (_, i) => ({ version: i + 1, goal: `Synthetic historical goal ${i + 1}` })),
  repeatedEvents: Array.from({ length: 10 }, () => ({ idempotency_key: "same-synthetic-event", version: 1 })),
  note: "IDs are synthetic and unseeded. The supported runtime capability is task-contract-v1. Use existing fixtures/TestApiClient to seed isolated storage; never send to a live workspace.",
};
writeFileSync(join(destination, "manifest.json"), JSON.stringify(manifest, null, 2) + "\n");
console.log(JSON.stringify({ output: destination, commitA, commitB, patchSha256: digest(patch) }));
