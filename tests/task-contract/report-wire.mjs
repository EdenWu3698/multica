import { readFileSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";

const folder = resolve(process.argv[2]);
const summary = JSON.parse(readFileSync(join(folder, "summary.json"), "utf8"));
const rows = [];
let semanticHookHits = 0;
for (const suite of summary.results) {
  if (["server", "runtime", "recovery"].includes(suite.name)) {
    const events = readFileSync(join(folder, `${suite.name}.stdout`), "utf8").split("\n").filter(Boolean).map((line) => JSON.parse(line));
    semanticHookHits += events.filter((event) => event.Output?.includes("W54_SEAM_HIT")).length;
    const results = events.filter((e) => e.Test && ["pass", "fail", "skip"].includes(e.Action));
    for (const result of results.filter((e) => !results.some((other) => other.Test.startsWith(e.Test + "/")))) {
      const output = events.filter((e) => e.Test === result.Test && e.Output).map((e) => e.Output).join("");
      const classification = result.Action === "pass" ? "baseline_or_test_infrastructure" :
        output.includes("LOCAL_PENDING") ? "product_seam_mapping_pending" :
        output.includes("404 page not found") ? "missing_contract_route_later_assertions_unreached" :
        output.includes("actual start omitted protected admission") ? "missing_runtime_admission_wire" :
        output.includes("protected task accepted unbound HTTP 200") ? "unsafe_runtime_acceptance_and_observed_local_effects" : "needs_manual_classification";
      rows.push({ suite: suite.name, test: result.Test, tc: result.Test.match(/TC\d\d/)?.[0] ?? null,
        ws: result.Test.match(/WS\d\d/)?.[0] ?? null, result: result.Action, classification,
        firstFailure: result.Action === "fail" ? output.split("\n").find((line) => /\.go:\d+:/.test(line) && !/W54_(PERF|CODE_SEED|READ_SPY|RESTART)/.test(line))?.trim() ?? null : null,
        evidence: `${suite.name}.stdout`, sourceSHA: summary.sourceHead });
    }
  } else {
    const report = JSON.parse(readFileSync(join(folder, `${suite.name}.json`), "utf8"));
    for (const file of report.testResults) for (const result of file.assertionResults) {
      rows.push({ suite: suite.name, test: result.fullName, tc: result.fullName.match(/TC-?\d\d/)?.[0] ?? null,
        result: { passed: "pass", failed: "fail", pending: "skip" }[result.status],
        classification: result.status === "passed" ? "existing_baseline" : suite.name === "core" ? "unvalidated_adoption_marker" : "missing_web_authoring_entry",
        firstFailure: result.failureMessages?.[0] ?? null, evidence: `${suite.name}.json`, sourceSHA: summary.sourceHead });
    }
  }
}
const counts = Object.fromEntries(["pass", "fail", "skip"].map((key) => [key, rows.filter((r) => r.result === key).length]));
const classificationCounts = {};
for (const row of rows) classificationCounts[row.classification] = (classificationCounts[row.classification] ?? 0) + 1;
const output = { sourceSHA: summary.sourceHead, counts, classificationCounts, semanticHookHits,
  semanticHookConclusion: semanticHookHits === 0 ? "No semantic point was reached. No transaction ordering, rollback, or commit-unknown guarantee is verified." : "Inspect each scenario's hit and product assertions; hit counts alone do not prove correctness.", rows };
writeFileSync(join(folder, "leaf-results.json"), JSON.stringify(output, null, 2) + "\n");
console.log(JSON.stringify({ sourceSHA: output.sourceSHA, counts, classificationCounts }, null, 2));
if (rows.some((row) => row.classification === "needs_manual_classification")) process.exitCode = 1;
