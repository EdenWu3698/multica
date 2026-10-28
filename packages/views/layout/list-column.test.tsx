import { fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SidebarProvider } from "@multica/ui/components/ui/sidebar";
import { renderWithI18n } from "../test/i18n";
import { AppShellProvider, useAppShell } from "./app-shell-context";
import { ListColumn } from "./list-column";

vi.mock("../navigation", () => ({
  useNavigation: () => ({ pathname: "/acme/inbox" }),
}));

let shell: ReturnType<typeof useAppShell> = null;
function Reader() {
  shell = useAppShell();
  return null;
}

describe("ListColumn", () => {
  it("collapses the list column from its edge, like the space panel", () => {
    const { container } = renderWithI18n(
      <SidebarProvider>
        <AppShellProvider>
          <Reader />
          <ListColumn>list</ListColumn>
        </AppShellProvider>
      </SidebarProvider>,
    );

    expect(container.querySelector("[data-slot='list-column']")).toHaveClass(
      "w-(--sidebar-width)",
    );
    fireEvent.click(container.querySelector("[data-slot='space-panel-resize']")!);
    expect(shell?.listCollapsed).toBe(true);
  });

  it("renders without a resize edge outside the shell", () => {
    const { container } = renderWithI18n(<ListColumn>list</ListColumn>);
    expect(container.querySelector("[data-slot='space-panel-resize']")).toBeNull();
  });

  // Collapsing animates the width rather than unmounting, like the space panel.
  it("stays mounted and inert while collapsed", () => {
    const { container } = renderWithI18n(<ListColumn collapsed>list</ListColumn>);
    const column = container.querySelector("[data-slot='list-column']");
    expect(column).toHaveAttribute("data-state", "collapsed");
    expect(column).toHaveAttribute("inert");
    expect(column).toHaveClass("w-0", "transition-[width]");
    expect(column).toHaveTextContent("list");
  });
});
