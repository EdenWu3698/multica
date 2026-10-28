/**
 * The rail sits on the app shell, where the page surface tokens are nearly
 * invisible (`--surface-hover` is within a hair of `--app-shell` in light
 * mode), so rail controls use the sidebar accent instead.
 */
export const RAIL_ICON_BUTTON_CLASS =
  "relative inline-flex size-9 shrink-0 cursor-pointer items-center justify-center rounded-lg text-muted-foreground transition-colors outline-none hover:bg-sidebar-accent/70 hover:text-sidebar-accent-foreground focus-visible:outline-2 focus-visible:outline-ring data-popup-open:bg-sidebar-accent data-popup-open:text-sidebar-accent-foreground aria-[current=page]:bg-sidebar-accent aria-[current=page]:text-sidebar-accent-foreground [&_svg:not([class*='size-'])]:size-[18px]";
