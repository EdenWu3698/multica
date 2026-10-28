import type { ReactNode } from "react";
import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithI18n } from "../test/i18n";
import { RailWorkspaces } from "./rail-workspaces";

type Ws = { id: string; name: string; slug: string; avatar_url: string | null };

const state = vi.hoisted(() => ({
  workspaces: [] as Ws[],
  invitations: [] as { id: string; workspace_id: string; workspace_name?: string; inviter_name?: string }[],
  summary: [] as { workspace_id: string; count: number }[],
  creationDisabled: false,
  accept: vi.fn(),
  decline: vi.fn(),
  invalidateQueries: vi.fn(),
  mutations: [] as Array<Record<string, unknown>>,
}));

vi.mock("../navigation", () => ({
  AppLink: ({ children, href, ...props }: { children: ReactNode; href: string } & React.ComponentProps<"a">) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
  useNavigation: () => ({ pathname: "/active/issues", push: vi.fn() }),
}));
vi.mock("@multica/core/paths", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@multica/core/paths")>()),
  useCurrentWorkspace: () => ({ id: "ws-1", name: "Active WS", slug: "active" }),
}));
vi.mock("@multica/core/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@multica/core/api")>();
  return {
    ...actual,
    api: {
      ...actual.api,
      getBaseUrl: () => "http://127.0.0.1:8080",
      acceptInvitation: state.accept,
      declineInvitation: state.decline,
    },
  };
});
vi.mock("@multica/core/config", () => ({
  useConfigStore: (selector: (s: { workspaceCreationDisabled: boolean }) => unknown) =>
    selector({ workspaceCreationDisabled: state.creationDisabled }),
}));
vi.mock("@multica/core/inbox/queries", () => ({
  inboxUnreadSummaryOptions: () => ({ queryKey: ["inbox", "unread-summary"] }),
  inboxUnreadSummaryEnabled: () => true,
  unreadWorkspaceIds: (entries: { workspace_id: string; count: number }[]) =>
    new Set(entries.filter((s) => s.count > 0).map((s) => s.workspace_id)),
}));
vi.mock("@multica/core/workspace/queries", () => ({
  myInvitationListOptions: () => ({ queryKey: ["invitations"] }),
  workspaceKeys: { myInvitations: () => ["invitations"] },
  workspaceListOptions: () => ({ queryKey: ["workspaces"] }),
}));
vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@tanstack/react-query")>()),
  useMutation: (options: Record<string, unknown> = {}) => {
    state.mutations.push(options);
    return { isPending: false, mutate: vi.fn() };
  },
  useQuery: ({ queryKey }: { queryKey: readonly unknown[] }) => {
    if (queryKey[0] === "workspaces") return { data: state.workspaces };
    if (queryKey[0] === "invitations") return { data: state.invitations };
    if (queryKey[0] === "inbox") return { data: state.summary };
    return { data: undefined };
  },
  useQueryClient: () => ({ fetchQuery: vi.fn(), invalidateQueries: state.invalidateQueries }),
}));
vi.mock("@multica/ui/components/common/tooltip-hint", () => ({
  TooltipHint: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

const ws = (n: number): Ws => ({ id: `ws-${n}`, name: n === 1 ? "Active WS" : `WS ${n}`, slug: n === 1 ? "active" : `ws${n}`, avatar_url: null });
const tile = (name: string) => screen.queryByRole("link", { name });
const hasDot = (name: string) => !!tile(name)?.querySelector(".bg-brand");

beforeEach(() => {
  state.workspaces = [ws(1), ws(2)];
  state.invitations = [];
  state.summary = [];
  state.creationDisabled = false;
  state.mutations.length = 0;
});

describe("RailWorkspaces", () => {
  it("renders a tile per workspace and marks the current one", () => {
    renderWithI18n(<RailWorkspaces />);
    expect(tile("Active WS")).toHaveAttribute("aria-current", "page");
    expect(tile("Active WS")).toHaveAttribute("href", "/active/issues");
    expect(tile("WS 2")).not.toHaveAttribute("aria-current");
  });

  it("dots another workspace with unread inbox items", () => {
    state.summary = [{ workspace_id: "ws-2", count: 3 }];
    renderWithI18n(<RailWorkspaces />);
    expect(hasDot("WS 2")).toBe(true);
    expect(hasDot("Active WS")).toBe(false);
  });

  // The current workspace's unread is the Inbox space's count.
  it("never dots the current workspace", () => {
    state.summary = [{ workspace_id: "ws-1", count: 5 }];
    renderWithI18n(<RailWorkspaces />);
    expect(hasDot("Active WS")).toBe(false);
  });

  it("does not dot a workspace whose unread count is zero", () => {
    state.summary = [{ workspace_id: "ws-2", count: 0 }];
    renderWithI18n(<RailWorkspaces />);
    expect(hasDot("WS 2")).toBe(false);
  });

  it("moves extra workspaces behind an overflow menu but keeps the current one", () => {
    state.workspaces = [ws(2), ws(3), ws(4), ws(5), ws(6), ws(1)];
    renderWithI18n(<RailWorkspaces />);
    expect(tile("Active WS")).not.toBeNull();
    expect(tile("WS 2")).not.toBeNull();
    expect(tile("WS 3")).not.toBeNull();
    expect(tile("WS 4")).toBeNull();
    expect(screen.getByRole("button", { name: "More workspaces" })).toBeInTheDocument();
  });

  it("offers create unless workspace creation is disabled", () => {
    const { unmount } = renderWithI18n(<RailWorkspaces />);
    expect(screen.getByRole("link", { name: "Create workspace" })).toHaveAttribute("href", "/workspaces/new");
    unmount();

    state.creationDisabled = true;
    renderWithI18n(<RailWorkspaces />);
    expect(screen.queryByRole("link", { name: "Create workspace" })).toBeNull();
  });

  it("shows pending invitations as their own tiles", () => {
    state.invitations = [{ id: "inv-1", workspace_id: "ws-9", workspace_name: "Copilot Hub" }];
    renderWithI18n(<RailWorkspaces />);
    expect(screen.getByRole("button", { name: "Invitation to Copilot Hub" })).toBeInTheDocument();
  });
});

describe("pending invitation self-heal", () => {
  beforeEach(() => {
    state.accept.mockReset().mockRejectedValue(new Error("invitation is not pending"));
    state.decline.mockReset().mockRejectedValue(new Error("invitation is not pending"));
    state.invalidateQueries.mockClear();
  });

  // "Invitation is not pending": it was concluded on another surface. Both
  // mutations must refetch the list on failure so the stale tile drops.
  it("invalidates the pending list when accept or decline fails", async () => {
    renderWithI18n(<RailWorkspaces />);
    expect(state.mutations).toHaveLength(2);

    for (const options of state.mutations as Array<{
      mutationFn: (id: string) => Promise<unknown>;
      onError?: (...args: unknown[]) => unknown;
      onSettled?: (...args: unknown[]) => unknown;
    }>) {
      const settle = options.onError ?? options.onSettled;
      expect(settle).toBeTypeOf("function");
      state.invalidateQueries.mockClear();
      await expect(options.mutationFn("inv-1")).rejects.toThrow("invitation is not pending");
      await settle!(new Error("invitation is not pending"), "inv-1", undefined, undefined);
      expect(state.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["invitations"] });
    }
    expect(state.accept).toHaveBeenCalledTimes(1);
    expect(state.decline).toHaveBeenCalledTimes(1);
  });
});
