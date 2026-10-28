"use client";

import { useCallback, useEffect, useRef } from "react";
import type { AppSpaceId } from "@multica/core/paths";
import { cn } from "@multica/ui/lib/utils";
import {
  clampSidebarWidth,
  useSidebar,
  useSidebarResize,
} from "@multica/ui/components/ui/sidebar";
import { useT } from "../i18n";
import { useAppShell } from "./app-shell-context";
import { AutopilotsPanel } from "./panels/autopilots-panel";
import { TeamPanel } from "./panels/team-panel";
import { WorkPanel } from "./panels/work-panel";

const PANELS: Partial<Record<AppSpaceId, () => React.ReactNode>> = {
  work: WorkPanel,
  team: TeamPanel,
  autopilots: AutopilotsPanel,
};

const RESIZE_DRAG_THRESHOLD = 2;

/** The current space's panel content, or null for spaces without one. */
export function SpacePanelContent({ space }: { space: AppSpaceId | null }) {
  const Panel = space ? PANELS[space] : undefined;
  return Panel ? <Panel /> : null;
}

/**
 * The panel's trailing edge: drag to resize, click to collapse. The preview
 * writes the width straight onto the panel element and commits once on
 * release, so a drag re-renders nothing until the pointer comes up.
 */
function PanelResizeHandle({ panelRef }: { panelRef: React.RefObject<HTMLElement | null> }) {
  const { t } = useT("ui");
  const { toggleSidebar } = useSidebar();
  const { commitWidth } = useSidebarResize();
  const cancelRef = useRef<(() => void) | null>(null);
  const draggedRef = useRef(false);

  useEffect(() => () => cancelRef.current?.(), []);

  const onPointerDown = useCallback(
    (e: React.PointerEvent<HTMLButtonElement>) => {
      const panel = panelRef.current;
      if (e.button !== 0 || e.isPrimary === false || !panel) return;
      e.preventDefault();
      cancelRef.current?.();
      draggedRef.current = false;
      const handle = e.currentTarget;
      const pointerId = e.pointerId;
      const startX = e.clientX;
      const startWidth = clampSidebarWidth(panel.getBoundingClientRect().width);
      let latestWidth = startWidth;
      document.documentElement.setAttribute("data-sidebar-resizing", "true");

      const finish = (mode: "commit" | "cancel") => {
        document.removeEventListener("pointermove", onMove);
        document.removeEventListener("pointerup", onUp);
        document.removeEventListener("pointercancel", onCancel);
        window.removeEventListener("blur", onBlur);
        document.documentElement.removeAttribute("data-sidebar-resizing");
        if (mode === "commit" && draggedRef.current) commitWidth(latestWidth);
        panel.style.removeProperty("--sidebar-width");
        cancelRef.current = null;
        if (handle.hasPointerCapture?.(pointerId)) handle.releasePointerCapture?.(pointerId);
      };
      const onMove = (event: PointerEvent) => {
        if (event.pointerId !== pointerId) return;
        const delta = event.clientX - startX;
        if (!draggedRef.current && Math.abs(delta) < RESIZE_DRAG_THRESHOLD) return;
        draggedRef.current = true;
        const next = clampSidebarWidth(startWidth + delta);
        if (next === latestWidth) return;
        latestWidth = next;
        panel.style.setProperty("--sidebar-width", `${next}px`);
      };
      const onUp = (event: PointerEvent) => {
        if (event.pointerId === pointerId) finish("commit");
      };
      const onCancel = (event: PointerEvent) => {
        if (event.pointerId === pointerId) finish("cancel");
      };
      const onBlur = () => finish("cancel");

      document.addEventListener("pointermove", onMove);
      document.addEventListener("pointerup", onUp);
      document.addEventListener("pointercancel", onCancel);
      window.addEventListener("blur", onBlur);
      cancelRef.current = () => finish("cancel");
      handle.setPointerCapture?.(pointerId);
    },
    [commitWidth, panelRef],
  );

  return (
    <button
      type="button"
      data-slot="space-panel-resize"
      aria-label={t(($) => $.toggle_sidebar)}
      title={t(($) => $.toggle_sidebar)}
      tabIndex={-1}
      onPointerDown={onPointerDown}
      onClick={() => {
        if (!draggedRef.current) toggleSidebar();
      }}
      className="absolute inset-y-0 -right-1 z-20 w-2 cursor-ew-resize touch-none after:absolute after:inset-y-0 after:left-1/2 after:w-px after:-translate-x-1/2 after:transition-colors hover:after:bg-ring/60"
    />
  );
}

/**
 * The space panel inside the content card: a column on the card's leading
 * edge, separated from the page by one hairline. Spaces without a shell
 * panel (Inbox, Chat, Settings) render their own list column instead, so
 * nothing mounts for them. While collapsed, hovering a rail space previews
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
          {open && <PanelResizeHandle panelRef={panelRef} />}
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
