"use client";

import { createContext, use, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { spaceForPath, type AppSpaceId } from "@multica/core/paths";
import { useSidebarSafe } from "@multica/ui/components/ui/sidebar";
import { useNavigation } from "../navigation";

/**
 * Spaces whose navigation lives in the shell panel. Inbox, Chat and Settings
 * are list-and-detail pages that already render their own list column in the
 * same card, so the shell adds no panel for them.
 */
export const PANEL_SPACES: ReadonlySet<AppSpaceId> = new Set(["work", "team", "autopilots"]);

// Opening waits for the pointer to settle so sweeping across the rail does
// not flash every panel; closing waits long enough to cross the gap between
// the rail and the preview.
const PEEK_OPEN_DELAY_MS = 150;
const PEEK_CLOSE_DELAY_MS = 200;

interface AppShellContextValue {
  space: AppSpaceId | null;
  /** The current space has a shell panel (collapsible with the sidebar toggle). */
  hasPanel: boolean;
  /**
   * Inbox, Chat and Settings keep their own list column, which the sidebar
   * toggle collapses instead of the shell panel. Independent of the panel's
   * open state so the md–xl auto-collapse never hides a list the page needs.
   */
  listCollapsed: boolean;
  setListCollapsed: (collapsed: boolean) => void;
  /** The sidebar toggle: the space panel, the list column, or the compact sheet. */
  toggleNav: () => void;
  /** Space previewed over the content while the panel is collapsed. */
  peekSpace: AppSpaceId | null;
  requestPeek: (space: AppSpaceId) => void;
  holdPeek: () => void;
  releasePeek: () => void;
  closePeek: () => void;
}

const AppShellContext = createContext<AppShellContextValue | null>(null);

export function AppShellProvider({ children }: { children: React.ReactNode }) {
  const { pathname } = useNavigation();
  const sidebar = useSidebarSafe();
  const space = spaceForPath(pathname);
  const hasPanel = space !== null && PANEL_SPACES.has(space);
  const [listCollapsed, setListCollapsed] = useState(false);
  const [peekSpace, setPeekSpace] = useState<AppSpaceId | null>(null);
  const openTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const peekRef = useRef(peekSpace);
  useEffect(() => {
    peekRef.current = peekSpace;
  }, [peekSpace]);

  const clearTimers = useCallback(() => {
    if (openTimer.current) clearTimeout(openTimer.current);
    if (closeTimer.current) clearTimeout(closeTimer.current);
    openTimer.current = null;
    closeTimer.current = null;
  }, []);

  const requestPeek = useCallback(
    (next: AppSpaceId) => {
      clearTimers();
      if (peekRef.current !== null) {
        setPeekSpace(next);
        return;
      }
      openTimer.current = setTimeout(() => setPeekSpace(next), PEEK_OPEN_DELAY_MS);
    },
    [clearTimers],
  );
  const holdPeek = useCallback(() => {
    if (closeTimer.current) clearTimeout(closeTimer.current);
    closeTimer.current = null;
  }, []);
  const releasePeek = useCallback(() => {
    clearTimers();
    closeTimer.current = setTimeout(() => setPeekSpace(null), PEEK_CLOSE_DELAY_MS);
  }, [clearTimers]);
  const closePeek = useCallback(() => {
    clearTimers();
    setPeekSpace(null);
  }, [clearTimers]);

  // Following a link out of the preview lands on its page; the preview has
  // done its job.
  useEffect(() => {
    closePeek();
  }, [pathname, closePeek]);
  useEffect(() => clearTimers, [clearTimers]);

  const toggleSidebar = sidebar?.toggleSidebar;
  const isCompact = sidebar?.isCompact ?? false;
  const toggleNav = useCallback(() => {
    if (isCompact || hasPanel || space === null) toggleSidebar?.();
    else setListCollapsed((collapsed) => !collapsed);
  }, [hasPanel, isCompact, space, toggleSidebar]);

  const value = useMemo<AppShellContextValue>(
    () => ({
      space,
      hasPanel,
      listCollapsed,
      setListCollapsed,
      toggleNav,
      peekSpace,
      requestPeek,
      holdPeek,
      releasePeek,
      closePeek,
    }),
    [space, hasPanel, listCollapsed, toggleNav, peekSpace, requestPeek, holdPeek, releasePeek, closePeek],
  );

  return <AppShellContext.Provider value={value}>{children}</AppShellContext.Provider>;
}

/** Null outside the dashboard shell (standalone pages, isolated tests). */
export function useAppShell(): AppShellContextValue | null {
  return use(AppShellContext);
}

/**
 * Whether a list-and-detail page (Inbox, Chat, Settings) hides its list
 * column. Below `md` those pages manage the list themselves.
 */
export function useListColumnCollapsed(): boolean {
  const shell = useAppShell();
  const sidebar = useSidebarSafe();
  return Boolean(shell?.listCollapsed && !shell.hasPanel && !sidebar?.isCompact);
}
