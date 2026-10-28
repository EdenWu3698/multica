"use client";

import { useId, useRef } from "react";
import { cn } from "@multica/ui/lib/utils";
import { useScrollFade } from "@multica/ui/hooks/use-scroll-fade";
import { AppLink } from "../navigation";

/**
 * Building blocks of the space panels. Rows share the Settings navigation
 * styling: Settings renders its own column in the same card, and switching
 * between it and a shell panel must not change how a nav row looks.
 */

/** A section root stays lit on its child routes (a project on its detail page). */
export function isPathWithin(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(href + "/");
}

export function PanelHeader({
  title,
  children,
}: {
  title: string;
  /** Trailing actions (icon buttons). */
  children?: React.ReactNode;
}) {
  return (
    <div className="flex h-12 shrink-0 items-center gap-1 pl-5 pr-2">
      <h2 className="min-w-0 flex-1 truncate text-title-sm font-semibold">{title}</h2>
      {children}
    </div>
  );
}

export function PanelBody({ children }: { children: React.ReactNode }) {
  const ref = useRef<HTMLDivElement>(null);
  const fadeStyle = useScrollFade(ref, 24);
  return (
    <div
      ref={ref}
      style={fadeStyle}
      className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-2 pb-4"
    >
      {children}
    </div>
  );
}

export function PanelSection({
  label,
  action,
  children,
}: {
  label: string;
  /** Rendered at the end of the label row (e.g. a create button). */
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  const id = useId();
  return (
    <section aria-labelledby={id} className="mt-4">
      <div className="flex h-7 items-center gap-1 pl-3 pr-1">
        <h3 id={id} className="min-w-0 flex-1 truncate text-caption font-medium text-muted-foreground">
          {label}
        </h3>
        {action}
      </div>
      {children}
    </section>
  );
}

export function PanelList({
  children,
  className,
  ...props
}: React.ComponentProps<"ul">) {
  return (
    <ul className={cn("space-y-px", className)} {...props}>
      {children}
    </ul>
  );
}

export const PANEL_ROW_CLASS =
  "flex min-h-8 w-full items-center gap-2.5 rounded-lg px-3 py-1.5 text-left text-body transition-colors outline-none focus-visible:outline-2 focus-visible:outline-ring";

export function panelRowStateClass(active: boolean) {
  return active
    ? "bg-surface-selected font-medium text-surface-selected-foreground hover:bg-surface-selected"
    : "text-muted-foreground hover:bg-surface-hover hover:text-foreground";
}

export function PanelCount({ value, className }: { value: number; className?: string }) {
  if (value <= 0) return null;
  return (
    <span className={cn("shrink-0 text-caption tabular-nums text-faint-foreground", className)}>
      {value}
    </span>
  );
}

interface PanelLinkProps {
  href: string;
  label: string;
  active: boolean;
  /** Leading visual; icons are sized by the row. */
  icon?: React.ReactNode;
  /** Second line under the label (agent activity, schedule). */
  description?: React.ReactNode;
  /** End of the row: a count, a status, a tag. */
  trailing?: React.ReactNode;
  /** Runs before navigation, e.g. to select a list scope the page reads. */
  onNavigate?: () => void;
  className?: string;
}

export function PanelLink({
  href,
  label,
  active,
  icon,
  description,
  trailing,
  onNavigate,
  className,
}: PanelLinkProps) {
  return (
    <li>
      <AppLink
        href={href}
        newTabTitle={label}
        aria-current={active ? "page" : undefined}
        onClick={onNavigate}
        className={cn(
          PANEL_ROW_CLASS,
          panelRowStateClass(active),
          "[&_svg:not([class*='size-'])]:size-4 [&_svg]:shrink-0",
          className,
        )}
      >
        {icon}
        {description ? (
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="truncate">{label}</span>
            <span className="truncate text-caption font-normal text-muted-foreground">
              {description}
            </span>
          </span>
        ) : (
          <span className="min-w-0 flex-1 truncate">{label}</span>
        )}
        {trailing}
      </AppLink>
    </li>
  );
}
