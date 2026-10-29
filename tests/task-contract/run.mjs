import { spawnSync } from "node:child_process";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { resolve, join } from "node:path";

const root = resolve(new URL("../..", import.meta.url).pathname);
const scope = process.argv[2] ?? "all";
const out = resolve(process.argv[3] ?? `tests/task-contract/evidence/run-${Date.now()}`);
mkdirSync(out, { recursive: true });
const env = { ...process.env };
const selections = {
  core: ["pnpm", ["--filter", "@multica/core", "exec", "vitest", "run", "api/task-contract-acceptance.test.ts", "--reporter=json", `--outputFile=${join(out, "core.json")}`]],
  runtime: ["bash", ["scripts/go-test-with-agent-cli-guard.sh", "--", env.W54_GO ?? "go", "-C", "server", "test", "./internal/daemon", "-run", "^TestW54Runtime", "-count=1", "-json"]],
  server: ["bash", ["scripts/go-test-with-agent-cli-guard.sh", "--", env.W54_GO ?? "go", "-C", "server", "test", "./cmd/server", "-run", "^TestW54Contract", "-count=1", "-json"]],
  views: ["pnpm", ["--filter", "@multica/views", "exec", "vitest", "run", "issues/components/revision-conflict-compare.test.tsx", "issues/components/source-context-viewer.test.tsx", "issues/actions/run-confirm-gate.test.ts", "--reporter=json", `--outputFile=${join(out, "views.json")}`]],
};
if (scope !== "all" && !Object.hasOwn(selections, scope)) throw new Error(`Unknown scope: ${scope}`);
const summary = [];
for (const [name, [command, args]] of Object.entries(selections)) {
  if (scope !== "all" && scope !== name) continue;
  if (name === "server") {
    const url = env.DATABASE_URL ? new URL(env.DATABASE_URL) : null;
    if (!url || !["localhost", "127.0.0.1", "[::1]"].includes(url.hostname) || !/^\/multica_wush54_[a-z0-9_]+$/.test(url.pathname)) {
      summary.push({ name, status: "environment_blocked", reason: "explicit isolated multica_wush54_ DATABASE_URL required" });
      continue;
    }
  }
  const result = spawnSync(command, args, { cwd: root, env, encoding: "utf8", timeout: 180000, maxBuffer: 16*1024*1024 });
  const stdout = result.stdout ?? "";
  writeFileSync(join(out, `${name}.stdout`), stdout.replaceAll(root, "<checkout>"));
  writeFileSync(join(out, `${name}.stderr`), (result.stderr ?? "").replaceAll(root, "<checkout>"));
  let counts;
  try {
    if (["server", "runtime"].includes(name)) {
      const events = stdout.split("\n").filter(Boolean).map((line) => JSON.parse(line));
      const results = events.filter((event) => event.Test && ["pass", "fail", "skip"].includes(event.Action));
      const leaves = results.filter((event) => !results.some((other) => other.Test.startsWith(event.Test + "/")));
      counts = Object.fromEntries(["pass", "fail", "skip"].map((action) => [action, leaves.filter((event) => event.Action === action).length]));
    } else {
      const report = JSON.parse(readFileSync(join(out, `${name}.json`), "utf8"));
      counts = { pass: report.numPassedTests, fail: report.numFailedTests, skip: report.numPendingTests };
    }
  } catch { /* A parse failure is never counted as a product red. */ }
  const ran = counts && counts.pass + counts.fail > 0;
  summary.push({ name, command: [command, ...args], exit: result.status, counts,
    status: !ran || result.error ? "environment_or_harness_error" : counts.skip ? "incomplete_with_skips" : result.status === 0 ? "baseline_pass" : "failed_requires_classification" });
}
const head = spawnSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8" }).stdout.trim();
writeFileSync(join(out, "summary.json"), JSON.stringify({ sourceHead: head, scope, results: summary }, null, 2) + "\n");
console.log(JSON.stringify(summary, null, 2));
if (summary.some((entry) => entry.status !== "baseline_pass")) process.exitCode = 1;
