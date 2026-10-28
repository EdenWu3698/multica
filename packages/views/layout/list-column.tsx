"use client";

import { useRef } from "react";
import { useSidebarSafe } from "@multica/ui/components/ui/sidebar";
import { cn } from "@multica/ui/lib/utils";
import { useAppShell } from "./app-shell-context";
import { PanelResizeHandle } from "./panel-resize-handle";

/**
 * The list column of a list-and-detail page (Inbox, Chat, Settings). It takes
 * the space panel's width and resize edge, so the column's divider stays put
 * when switching between spaces and one drag resizes them all.
 */
export function ListColumn({
  className,
  children,
}: {
  className?: string;
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
      className={cn("relative flex h-full w-(--sidebar-width) shrink-0 flex-col border-r", className)}
    >
      {children}
      {sidebar && !sidebar.isCompact && onToggle && (
        <PanelResizeHandle panelRef={ref} onToggle={onToggle} />
      )}
    </aside>
  );
}
