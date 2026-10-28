import { act, cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SidebarProvider, useSidebar } from "@multica/ui/components/ui/sidebar";
import {
  AppShellProvider,
  useAppShell,
  useListColumnCollapsed,
} from "./app-shell-context";

const navigation = vi.hoisted(() => ({ pathname: "/acme/issues" }));

vi.mock("../navigation", () => ({
  useNavigation: () => ({ pathname: navigation.pathname }),
}));

let shell: ReturnType<typeof useAppShell> = null;
function Reader() {
  shell = useAppShell();
  return null;
}

let sidebarState: string | null = null;
let listColumnCollapsed = false;
function NavReader() {
  shell = useAppShell();
  sidebarState = useSidebar().state;
  listColumnCollapsed = useListColumnCollapsed();
  return null;
}

function renderNavShell() {
  return render(
    <SidebarProvider>
      <AppShellProvider>
        <NavReader />
      </AppShellProvider>
    </SidebarProvider>,
  );
}

function renderShell() {
  return render(
    <AppShellProvider>
      <Reader />
    </AppShellProvider>,
  );
}

beforeEach(() => {
  vi.useFakeTimers();
  navigation.pathname = "/acme/issues";
  shell = null;
});

afterEach(() => {
  vi.useRealTimers();
});

describe("AppShellProvider", () => {
  it("is absent outside the shell", () => {
    render(<Reader />);
    expect(shell).toBeNull();
  });

  it("gives panels to work, team and autopilots only", () => {
    const cases: [string, string | null, boolean][] = [
      ["/acme/my-issues", "work", true],
      ["/acme/squads/s-1", "team", true],
      ["/acme/autopilots", "autopilots", true],
      ["/acme/inbox", "inbox", false],
      ["/acme/chat", "chat", false],
      ["/acme/settings", "settings", false],
      ["/acme/attachments/a-1", null, false],
    ];
    for (const [pathname, space, hasPanel] of cases) {
      navigation.pathname = pathname;
      const { unmount } = renderShell();
      expect(shell?.space).toBe(space);
      expect(shell?.hasPanel).toBe(hasPanel);
      unmount();
    }
  });

  it("opens a peek after the pointer settles and switches instantly once open", () => {
    renderShell();
    act(() => shell!.requestPeek("team"));
    act(() => vi.advanceTimersByTime(100));
    expect(shell!.peekSpace).toBeNull();
    act(() => vi.advanceTimersByTime(100));
    expect(shell!.peekSpace).toBe("team");

    act(() => shell!.requestPeek("work"));
    expect(shell!.peekSpace).toBe("work");
  });

  it("closes after release unless the pointer reaches the preview", () => {
    renderShell();
    act(() => shell!.requestPeek("team"));
    act(() => vi.advanceTimersByTime(200));

    act(() => shell!.releasePeek());
    act(() => shell!.holdPeek());
    act(() => vi.advanceTimersByTime(500));
    expect(shell!.peekSpace).toBe("team");

    act(() => shell!.releasePeek());
    act(() => vi.advanceTimersByTime(250));
    expect(shell!.peekSpace).toBeNull();
  });

  it("closes the peek once navigation lands", () => {
    const { rerender } = renderShell();
    act(() => shell!.requestPeek("team"));
    act(() => vi.advanceTimersByTime(200));
    expect(shell!.peekSpace).toBe("team");

    navigation.pathname = "/acme/agents";
    rerender(
      <AppShellProvider>
        <Reader />
      </AppShellProvider>,
    );
    expect(shell!.peekSpace).toBeNull();
  });

  it("toggles the panel in panel spaces and the list column elsewhere", () => {
    renderNavShell();
    act(() => shell!.toggleNav());
    expect(sidebarState).toBe("collapsed");
    expect(listColumnCollapsed).toBe(false);
    act(() => shell!.toggleNav());
    expect(sidebarState).toBe("expanded");
    cleanup();

    navigation.pathname = "/acme/inbox";
    renderNavShell();
    act(() => shell!.toggleNav());
    expect(sidebarState).toBe("expanded");
    expect(listColumnCollapsed).toBe(true);
  });
});
