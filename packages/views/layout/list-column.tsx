"use client";

import { useRef } from "react";
import { useSidebarSafe } from "@multica/ui/components/ui/sidebar";
import { cn } from "@multica/ui/lib/utils";
import { useAppShell } from "./app-shell-context";
import { PanelResizeHandle } from "./panel-resize-handle";

/**
 * The list column of a list-and-detail page (Inbox, Chat, Settings). It takes
 * the space panel's width, resize edge and collapse motion, so the column's
 * divider stays put when switching between spaces and one drag resizes them
 * all. It stays mounted while collapsed, keeping the list's scroll position.
 */
export function ListColumn({
  collapsed = false,
  className,
  innerClassName,
  children,
}: {
  collapsed?: boolean;
  className?: string;
  innerClassName?: string;
  children: React.ReactNode;
}) {
  const ref = useRef<HTMLElement>(null);
  const sidebar = useSidebarSafe();
  const shell = useAppShell();
  const onToggle = shell?.toggleNav ?? sidebar?.toggleSidebar;

  return (
    <aside
      ref={ref}
      data-slot="list-column"
      data-state={collapsed ? "collapsed" : "expanded"}
      inert={collapsed}
      className={cn(
        "relative h-full shrink-0 transition-[width] duration-200 ease-out motion-reduce:transition-none",
        collapsed ? "w-0 overflow-hidden" : "w-(--sidebar-width)",
        className,
      )}
    >
      <div
        className={cn(
          "flex h-full w-(--sidebar-width) flex-col overflow-hidden border-r",
          innerClassName,
        )}
      >
        {children}
      </div>
      {!collapsed && sidebar && !sidebar.isCompact && onToggle && (
        <PanelResizeHandle panelRef={ref} onToggle={onToggle} />
      )}
    </aside>
  );
}
