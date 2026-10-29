import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { assertIsolatedPreview } from "./preflight.mjs";

test("preview preflight refuses remote API and non-test databases before any network access", () => {
  const local = { NEXT_PUBLIC_API_URL: "http://127.0.0.1:18988", PLAYWRIGHT_BASE_URL: "http://127.0.0.1:13908", DATABASE_URL: "postgres://synthetic@localhost:5432/multica_wush54_fixture" };
  assert.throws(() => assertIsolatedPreview({ ...local, NEXT_PUBLIC_API_URL: "https://api.example.invalid" }), /not loopback/);
  assert.throws(() => assertIsolatedPreview({ ...local, DATABASE_URL: "postgres://synthetic@localhost:5432/multica" }), /dedicated/);
  assert.throws(() => assertIsolatedPreview(local), /synthetic-only/);
});

test("preview preflight requires a fixed source identity and isolation handoff", (t) => {
  const dir = mkdtempSync(join(tmpdir(), "w54-preflight-"));
  t.after(() => rmSync(dir, { recursive: true }));
  const manifest = join(dir, "preview.json");
  const env = { NEXT_PUBLIC_API_URL: "http://127.0.0.1:18988", PLAYWRIGHT_BASE_URL: "http://127.0.0.1:13908", DATABASE_URL: "postgres://synthetic@localhost:5432/multica_wush54_fixture", W54_SYNTHETIC_ONLY: "1", W54_PREVIEW_MANIFEST: manifest };
  assert.throws(() => assertIsolatedPreview(env), /manifest is missing/);
  writeFileSync(manifest, JSON.stringify({ source_sha: "main" }));
  assert.throws(() => assertIsolatedPreview(env), /incomplete/);
  writeFileSync(manifest, JSON.stringify({ source_sha: "1".repeat(40), synthetic_only: true, no_real_agents: true, stop_instructions: "Stop the synthetic fixture" }));
  assert.equal(assertIsolatedPreview(env).synthetic_only, true);
});
