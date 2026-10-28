## Summary

Raise the app-wide tooltip open delay from 200ms to 500ms. With truncation tooltips now on table cells, board column headers, label chips, and run rows, 200ms fires on any brief pause while the pointer passes over. The close delay stays 0, since tooltips hold nothing clickable.

- **One delay constant.** `TOOLTIP_OPEN_DELAY = 500` lives in `packages/ui/components/ui/tooltip.tsx`. It is the default for `TooltipProvider` and is also used by `DeferredTooltip` (and so `TruncatedTooltip`), which previously hard-coded 200. The theme provider no longer passes `delay={200}`, and the create-issue footer's nested `TooltipProvider delay={200}` is removed so it uses the root provider.
- **Instant hand-off for deferred tooltips.** Base UI's delay group opens the next tooltip without waiting when the pointer moves over within 400ms of one closing. `DeferredTooltip` drives a controlled tooltip with no Base UI trigger, so the group never saw it, and every truncated cell in a table would have waited the full 500ms. Tooltip popups now record when they mount and unmount. A deferred tooltip hovered while any tooltip is showing, or within `TOOLTIP_INSTANT_WINDOW` (400ms, Base UI's default, now set explicitly) of one closing, opens immediately.
- Local overrides are unchanged: DingTalk settings (0) and the editor bubble menu / zoom canvas (300).

Known gap: going the other way, from a deferred tooltip to a regular Base UI tooltip, still waits the full delay, because Base UI's group state is internal.

## Testing

- `packages/ui`: 29/29. New tests: a cold hover stays closed before 500ms and opens after; hovering a second deferred tooltip right after the first opens it at once. I confirmed the second test fails with the hand-off removed. Existing deferred and truncated tooltip tests now advance timers past 500ms.
- `packages/views`: full suite 6435/6435. `apps/desktop` renderer: 313/313.
- `tsc` and eslint are clean (only existing warnings in `stepper.tsx`, which this PR doesn't touch).
