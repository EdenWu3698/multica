import { expect, test } from "@playwright/test";
import { TestApiClient } from "./fixtures";
import { assertIsolatedPreview } from "../tests/task-contract/preflight.mjs";

test.describe("W54 TC-16/18 isolated Web entry", () => {
  test("login, create, comment and refresh preserve the same synthetic task", async ({ page }) => {
    const manifest = assertIsolatedPreview();
    const health = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/health`, { signal: AbortSignal.timeout(5000) });
    expect(health.status).toBe(200);
    expect((await health.json()).commit).toBe(manifest.source_sha);
    const api = new TestApiClient();
    const run = `${Date.now()}-${process.pid}`;
    try {
      await api.login(`w54-${run}@example.invalid`, "W54 synthetic member");
      const workspace = await api.ensureWorkspace("W54 synthetic workspace", `w54-${run}`);
      await api.markUserOnboarded();
      const issue = await api.createIssue("W54 synthetic Web task", { status: "backlog" });
      await api.createComment(issue.id, "W54 synthetic comment survives refresh");
      const token = api.getToken();
      expect(token).toBeTruthy();
      await page.addInitScript((value) => localStorage.setItem("multica_token", value!), token);
      await page.goto(`/${workspace.slug}/issues/${issue.id}`);
      await expect(page.getByText("W54 synthetic Web task", { exact: true }).first()).toBeVisible();
      await expect(page.getByText("W54 synthetic comment survives refresh", { exact: true })).toBeVisible();
      await page.reload();
      await expect(page.getByText("W54 synthetic Web task", { exact: true }).first()).toBeVisible();
      await expect(page.getByText("W54 synthetic comment survives refresh", { exact: true })).toBeVisible();
    } finally { await api.cleanup(); }
  });

  test("TC-16 loaded task exposes an explicit contract authoring action", async ({ page }) => {
    const manifest = assertIsolatedPreview();
    const health = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/health`, { signal: AbortSignal.timeout(5000) });
    expect(health.status).toBe(200);
    expect((await health.json()).commit).toBe(manifest.source_sha);
    const api = new TestApiClient();
    const run = `${Date.now()}-${process.pid}`;
    try {
      await api.login(`w54-contract-${run}@example.invalid`, "W54 synthetic contract author");
      const workspace = await api.ensureWorkspace("W54 synthetic contract workspace", `w54-contract-${run}`);
      await api.markUserOnboarded();
      const issue = await api.createIssue("W54 contract authoring task", { status: "backlog" });
      expect(issue.id).toBeTruthy();
      await page.addInitScript((value) => localStorage.setItem("multica_token", value!), api.getToken());
      await page.goto(`/${workspace.slug}/issues/${issue.id}`);
      await expect(page.getByText("W54 contract authoring task", { exact: true }).first()).toBeVisible();
      const contractAction = page.getByRole("button", { name: /(?:create|edit|task).*contract|合同/i })
        .or(page.getByRole("link", { name: /(?:create|edit|task).*contract|合同/i }));
      await expect(contractAction.first()).toBeVisible();
      await contractAction.first().click();
      await expect(page.getByRole("textbox", { name: /goal|目标/i })).toBeVisible();
    } finally { await api.cleanup(); }
  });
});
