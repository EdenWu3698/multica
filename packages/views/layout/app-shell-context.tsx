"use client";

import { createContext, use, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { spaceForPath, type AppSpaceId } from "@multica/core/paths";
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
  const space = spaceForPath(pathname);
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

  const value = useMemo<AppShellContextValue>(
    () => ({
      space,
      hasPanel: space !== null && PANEL_SPACES.has(space),
      peekSpace,
      requestPeek,
      holdPeek,
      releasePeek,
      closePeek,
    }),
    [space, peekSpace, requestPeek, holdPeek, releasePeek, closePeek],
  );

  return <AppShellContext.Provider value={value}>{children}</AppShellContext.Provider>;
}

/** Null outside the dashboard shell (standalone pages, isolated tests). */
export function useAppShell(): AppShellContextValue | null {
  return use(AppShellContext);
}
