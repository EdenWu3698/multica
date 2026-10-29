// @vitest-environment node
import { describe, expect, it } from "vitest";
import { IssueSchema, ListIssuesResponseSchema } from "./schemas";
import { parseWithFallback } from "./schema";

const issue = {
  id: "11111111-1111-4111-8111-111111111111",
  workspace_id: "22222222-2222-4222-8222-222222222222",
  identifier: "SYN-1", number: 1, title: "Synthetic ordinary issue",
  description: null, status: "backlog", priority: "none", assignee_id: null,
  assignee_type: null, creator_type: "member", creator_id: "33333333-3333-4333-8333-333333333333",
  parent_issue_id: null, project_id: null, position: 0, stage: null,
  start_date: null, due_date: null, metadata: {}, properties: {}, revision: 1,
  created_at: "2026-10-29T00:00:00Z", updated_at: "2026-10-29T00:00:00Z",
};

describe("W54 TC-17 existing response compatibility", () => {
  it("keeps an ordinary issue from an older server readable", () => {
    const parsed = IssueSchema.parse(issue);
    expect(parsed.id).toBe(issue.id);
    expect(parsed.title).toBe("Synthetic ordinary issue");
    expect(parsed.contract_required).not.toBe(true);
  });

  it("preserves an explicit protected issue marker", () => {
    const parsed = IssueSchema.parse({ ...issue, contract_required: true });
    expect(parsed.contract_required).toBe(true);
  });

  it("does not blank a mixed ordinary/protected issue list", () => {
    const fallback = { issues: [], total: 0 };
    const parsed = parseWithFallback({ issues: [issue, { ...issue, id: "protected", contract_required: true }], total: 2 }, ListIssuesResponseSchema, fallback, { endpoint: "GET /api/issues" });
    expect(parsed.total).toBe(2);
    expect(parsed.issues).toHaveLength(2);
  });
});

describe("W54 TC-17 adoption marker must be schema validated", () => {
  it.each(["false", "true", 0, 1, {}, []])("rejects malformed contract_required=%j instead of treating unknown input as validated", (value) => {
    // A loose unknown field currently bypasses validation entirely. This red
    // assertion concerns validation, not the eventual UI fallback design.
    const result = IssueSchema.safeParse({ ...issue, contract_required: value });
    expect(result.success).toBe(false);
  });
});
