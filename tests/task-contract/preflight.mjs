import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";

export function assertIsolatedPreview(env = process.env) {
  for (const key of ["NEXT_PUBLIC_API_URL", "PLAYWRIGHT_BASE_URL", "DATABASE_URL"]) {
    if (!env[key]) throw new Error(`ENVIRONMENT_BLOCKED: ${key} must explicitly name the isolated preview`);
    const url = new URL(env[key]);
    if (!["localhost", "127.0.0.1", "[::1]"].includes(url.hostname)) throw new Error(`ENVIRONMENT_BLOCKED: ${key} is not loopback`);
    if (key === "DATABASE_URL" && !/^\/multica_wush54_[a-z0-9_]+$/.test(url.pathname)) throw new Error("ENVIRONMENT_BLOCKED: database must use the dedicated multica_wush54_ namespace");
  }
  if (env.W54_SYNTHETIC_ONLY !== "1") throw new Error("ENVIRONMENT_BLOCKED: synthetic-only preview has not been attested");
  const path = resolve(env.W54_PREVIEW_MANIFEST ?? "tests/task-contract/preview-manifest.json");
  if (!existsSync(path)) throw new Error("ENVIRONMENT_BLOCKED: fixed-SHA preview manifest is missing (TARS/Data handoff)");
  const manifest = JSON.parse(readFileSync(path, "utf8"));
  if (!/^[a-f0-9]{40}$/.test(manifest.source_sha ?? "") || !manifest.synthetic_only || !manifest.no_real_agents || !manifest.stop_instructions) throw new Error("ENVIRONMENT_BLOCKED: preview identity/isolation evidence is incomplete");
  return manifest;
}

if (process.argv[1] && resolve(process.argv[1]) === new URL(import.meta.url).pathname) {
  try { assertIsolatedPreview(); console.log("Isolated preview preconditions present"); }
  catch (error) { console.error(error.message); process.exitCode = 2; }
}
