import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const hash = (bytes) => createHash("sha256").update(bytes).digest("hex");
// Only the approved example's ASCII keys/integral numbers are in this oracle.
const canonical = (value) => JSON.stringify(value, (_key, item) =>
  item && typeof item === "object" && !Array.isArray(item)
    ? Object.fromEntries(Object.entries(item).sort(([a], [b]) => a < b ? -1 : a > b ? 1 : 0)) : item);
const spec = readFileSync(new URL("specs/W54-SPEC-0.1-wire-supplement.md", import.meta.url), "utf8");

test("approved supplement fingerprint is unchanged", () => {
  assert.equal(hash(spec), "59190fd9a3f1b54e108d3437b28c32c4118192d43f6d38263cc9f84b5a1de71c");
});

for (const [bytes, expected] of [
  ["abc", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"],
  ["abc\n", "edeaaff3f1774ad2888673770c6d64097e391bc362d7d6fb34982ddf0efd18cb"],
  ["abc\r\n", "552bab6864c7a7b69a502ed1854b9245c0e1a30f008aaa0b281da62585fdb025"],
  ['{"description":null,"title":"abc"}', "8f32d28e013043aaa07ef670f216566e07c1643ff1a0cea8fcf644b66543ca59"],
  ['"abc"', "6cc43f858fbb763301637b5af970e2a46b46f461f27e5a0f41e009c59b827b25"],
  ["null", "74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b"],
]) test(`WS-01 byte vector ${JSON.stringify(bytes)}`, () => assert.equal(hash(bytes), expected));

test("WS-03 independently reproduces the supplement input_digest", () => {
  const examples = [...spec.matchAll(/```json\n([\s\S]*?)\n```/g)].map((match) => JSON.parse(match[1]));
  const put = examples.find((item) => item.goal && item.expected_version === 0);
  const generated = examples.find((item) => item.contract_id);
  const { expected_version: _v, expected_issue_revision: _r, ...business } = put;
  const snapshot = { ...business, ...generated, versions: { ...business.versions, ...generated.versions }, change: { ...business.change, ...generated.change } };
  const sources = new Map();
  const visit = (value) => {
    if (!value || typeof value !== "object") return;
    if (value.kind && value.supplied_by && value.digest) {
      const { kind, locator, version, digest } = value;
      sources.set(canonical([kind, locator, version]), { kind, locator, version, digest });
    }
    Object.values(value).forEach(visit);
  };
  visit(snapshot);
  const projection = {
    schema_version: 1, contract_id: snapshot.contract_id, version: snapshot.version,
    issue_basis: { title_digest: snapshot.versions.issue_basis.title_digest, description_digest: snapshot.versions.issue_basis.description_digest },
    sources: [...sources.values()].sort((a, b) => Buffer.compare(Buffer.from(canonical(a)), Buffer.from(canonical(b)))),
    code_basis: [],
  };
  assert.equal(hash(canonical(projection)), "eefb7d3a7502508494dadeb5c2940d515bd0baededed7468e179170122137094");
});
