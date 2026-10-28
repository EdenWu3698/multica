import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Drives the issue timeline's "Jump to latest" affordance (MUL-8324).
 *
 * `visible` turns on once the bottom of the scroll container — where the
 * newest timeline entry sits — is more than one viewport away, so the button
 * only shows when reaching the latest activity would take real scrolling.
 *
 * `jump` drives scrollTop to the bottom directly instead of asking Virtuoso
 * for the last index: with `customScrollParent`, writing scrollTop is what
 * makes Virtuoso materialize the tail rows, and those rows then grow as their
 * real heights (markdown, code, images) land. The rAF loop chases the moving
 * bottom until it holds for two frames, the same stabilization the scroll
 * restore and deep-link landing use.
 */
export function useJumpToLatest(scrollContainerEl: HTMLElement | null) {
  const [visible, setVisible] = useState(false);
  const cancelJumpRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (!scrollContainerEl) {
      setVisible(false);
      return;
    }
    let frame = 0;
    const measure = () => {
      frame = 0;
      setVisible(distanceFromBottom(scrollContainerEl) > scrollContainerEl.clientHeight);
    };
    const schedule = () => {
      if (frame === 0) frame = requestAnimationFrame(measure);
    };
    measure();
    scrollContainerEl.addEventListener("scroll", schedule, { passive: true });
    // Content height changes without a scroll event (rows measured, comments
    // streamed in, threads expanded), so observe the content too.
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(schedule);
    observer?.observe(scrollContainerEl);
    for (const child of Array.from(scrollContainerEl.children)) observer?.observe(child);
    return () => {
      scrollContainerEl.removeEventListener("scroll", schedule);
      observer?.disconnect();
      if (frame !== 0) cancelAnimationFrame(frame);
    };
  }, [scrollContainerEl]);

  useEffect(() => () => cancelJumpRef.current?.(), []);

  const jump = useCallback(() => {
    const el = scrollContainerEl;
    if (!el) return;
    cancelJumpRef.current?.();

    let cancelled = false;
    let attempts = 0;
    let stableFrames = 0;
    let frame = 0;
    const toBottom = () => {
      el.scrollTop = el.scrollHeight - el.clientHeight;
    };
    const tick = () => {
      if (cancelled || !el.isConnected) return;
      attempts += 1;
      if (distanceFromBottom(el) <= 1) {
        stableFrames += 1;
      } else {
        stableFrames = 0;
        toBottom();
      }
      if (stableFrames >= 2 || attempts >= 30) {
        cancelJumpRef.current?.();
        return;
      }
      frame = requestAnimationFrame(tick);
    };

    // The reader taking the wheel mid-chase wins over the loop.
    const stop = () => cancelJumpRef.current?.();
    const userInputs = ["wheel", "touchstart", "keydown"] as const;
    for (const type of userInputs) el.addEventListener(type, stop, { passive: true });

    toBottom();
    frame = requestAnimationFrame(tick);
    cancelJumpRef.current = () => {
      cancelled = true;
      cancelAnimationFrame(frame);
      for (const type of userInputs) el.removeEventListener(type, stop);
      cancelJumpRef.current = null;
    };
  }, [scrollContainerEl]);

  return { visible, jump };
}

function distanceFromBottom(el: HTMLElement) {
  return el.scrollHeight - el.scrollTop - el.clientHeight;
}
