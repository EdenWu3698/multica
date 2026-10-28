"use client";

import { useRef } from "react";
import { useSidebarSafe } from "@multica/ui/components/ui/sidebar";
import { cn } from "@multica/ui/lib/utils";
import { useAppShell } from "./app-shell-context";
import { PanelResizeHandle } from "./panel-resize-handle";
import { NAV_PEEK_CLASS } from "./rail-styles";

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
  // Hovering this space's rail item while collapsed floats the column itself
  // over the page. Only the inner box moves, so the column keeps its width
  // (no collapse motion) and the list keeps its state.
  const peeking = collapsed && !!shell?.peekSpace && shell.peekSpace === shell.space;

  return (
    <aside
      ref={ref}
      data-slot="list-column"
      data-state={collapsed ? "collapsed" : "expanded"}
      inert={collapsed && !peeking}
      className={cn(
        "relative h-full shrink-0 transition-[width] duration-200 ease-out motion-reduce:transition-none",
        collapsed ? "w-0" : "w-(--sidebar-width)",
        collapsed && !peeking && "overflow-hidden",
        className,
      )}
    >
      <div
        data-slot={peeking ? "list-column-peek" : undefined}
        onPointerEnter={peeking ? shell.holdPeek : undefined}
        onPointerLeave={peeking ? shell.releasePeek : undefined}
        className={cn(
          "flex h-full w-(--sidebar-width) flex-col overflow-hidden",
          peeking ? NAV_PEEK_CLASS : cn("border-r", innerClassName),
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
