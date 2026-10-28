"use client";

import { useRef } from "react";
import type { AppSpaceId } from "@multica/core/paths";
import { cn } from "@multica/ui/lib/utils";
import { useSidebar } from "@multica/ui/components/ui/sidebar";
import { useAppShell } from "./app-shell-context";
import { PanelResizeHandle } from "./panel-resize-handle";
import { AutopilotsPanel } from "./panels/autopilots-panel";
import { TeamPanel } from "./panels/team-panel";
import { WorkPanel } from "./panels/work-panel";

const PANELS: Partial<Record<AppSpaceId, () => React.ReactNode>> = {
  work: WorkPanel,
  team: TeamPanel,
  autopilots: AutopilotsPanel,
};

/** The current space's panel content, or null for spaces without one. */
export function SpacePanelContent({ space }: { space: AppSpaceId | null }) {
  const Panel = space ? PANELS[space] : undefined;
  return Panel ? <Panel /> : null;
}

/**
 * The space panel inside the content card: a column on the card's leading
 * edge, separated from the page by one hairline. Spaces without a shell
 * panel (Inbox, Chat, Settings) render their own `ListColumn` at the same
 * width instead, so nothing mounts for them. While collapsed, hovering a rail space previews
 * its panel over the page.
 */
export function SpacePanel() {
  const shell = useAppShell();
  const { open, isCompact } = useSidebar();
  const panelRef = useRef<HTMLElement>(null);
  if (!shell || isCompact) return null;

  const { space, hasPanel, peekSpace } = shell;
  const peeking = !open && peekSpace !== null;

  return (
    <>
      {hasPanel && (
        <aside
          // A new space mounts at its final width; only toggling animates.
          key={space}
          ref={panelRef}
          data-slot="space-panel"
          data-state={open ? "expanded" : "collapsed"}
          inert={!open}
          className={cn(
            "relative hidden shrink-0 transition-[width] duration-200 ease-out motion-reduce:transition-none md:block",
            open ? "w-(--sidebar-width)" : "w-0 overflow-hidden",
          )}
        >
          <div className="flex h-full w-(--sidebar-width) flex-col overflow-hidden border-r">
            <SpacePanelContent space={space} />
          </div>
          {open && <PanelResizeHandle panelRef={panelRef} onToggle={shell.toggleNav} />}
        </aside>
      )}
      {peeking && PANELS[peekSpace] && (
        <aside
          data-slot="space-panel-peek"
          onPointerEnter={shell.holdPeek}
          onPointerLeave={shell.releasePeek}
          className="absolute inset-y-0 left-0 z-30 hidden w-(--sidebar-width) flex-col overflow-hidden rounded-xl bg-page-canvas shadow-[var(--floating-shadow)] ring-1 ring-surface-border md:flex"
        >
          <SpacePanelContent space={peekSpace} />
        </aside>
      )}
    </>
  );
}
