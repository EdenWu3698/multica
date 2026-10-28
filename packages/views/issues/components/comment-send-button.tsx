"use client";

import { useLayoutEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode, type SyntheticEvent } from "react";
import { AnimatePresence, motion, useReducedMotion, type Transition } from "motion/react";
import { ChevronDown, Loader2, TriangleAlert, Users } from "lucide-react";
import type { CommentTriggerOutcome } from "@multica/core/types";
import type { RecipientAction } from "@multica/core/issues/run-steering";
import { mentionLabelsByTarget } from "@multica/core/issues/comment-trigger-outcomes";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@multica/ui/components/ui/tooltip";
import { AVATAR_SIZE_PX } from "@multica/ui/lib/avatar-size";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../../i18n";
import { blockedReasonLabel, blockedShortReasonLabel } from "../blocked-trigger-copy";
import type { RecipientEntry } from "../hooks/use-recipient-actions";
import {
  actionDescription,
  actionLabel,
  sourceLabel,
  stateLabel,
  TriggerAgentAvatar,
  useTriggerPresenceLine,
  type IssuesT,
} from "./comment-trigger-chips";

// The send button is the one place a composer says what sending will do: who
// receives the message (avatars) and what happens to them (the verb). Its
// shape never changes; only the verb, the avatars and the fill do. The
// chevron opens one menu with a radio group per recipient.

type SendTone = "neutral" | "steer" | "restart";

const TONE_CLASS: Record<SendTone, string> = {
  neutral: "bg-primary text-primary-foreground",
  steer: "bg-brand text-brand-foreground",
  restart: "bg-destructive text-white",
};
// Stacked avatars are cut out of the fill, so the ring follows the tone.
const TONE_RING: Record<SendTone, string> = {
  neutral: "ring-primary",
  steer: "ring-brand",
  restart: "ring-destructive",
};

const AVATAR_SIZE = AVATAR_SIZE_PX.xs;
const MAX_HEADS = 3;
const EASE_OUT: [number, number, number, number] = [0.32, 0.72, 0, 1];

function sendIntent(recipients: RecipientEntry[], t: IssuesT): { label: string; tone: SendTone } {
  const active = recipients.filter((r) => r.action !== "skip");
  const tone: SendTone = active.some((r) => r.action === "restart")
    ? "restart"
    : active.some((r) => r.action === "steer") ? "steer" : "neutral";
  if (active.length === 0) return { label: t(($) => $.comment.send_action_comment), tone };
  if (active.length === 1) {
    const only = active[0]!;
    const label = only.action === "start" && only.state.kind !== "queued"
      ? t(($) => $.comment.send_action_start)
      : actionLabel(only.action, only.state, t);
    return { label, tone };
  }
  return {
    label: tone === "restart"
      ? t(($) => $.comment.restart_send)
      : t(($) => $.comment.send_action_count, { count: active.length }),
    tone,
  };
}

/** Keep the composer focused so sending does not dismiss Android's keyboard. */
function keepFocusInComposer(event: SyntheticEvent<HTMLButtonElement>) {
  event.preventDefault();
}

function useMeasuredWidth() {
  const ref = useRef<HTMLSpanElement>(null);
  const [width, setWidth] = useState<number | null>(null);
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    setWidth(el.offsetWidth);
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(() => setWidth(el.offsetWidth));
    observer.observe(el);
    return () => observer.disconnect();
  }, []);
  return [ref, width] as const;
}

interface CommentSendButtonProps {
  recipients: RecipientEntry[];
  /** Explicit mentions that will not trigger; listed with their reason in the menu. */
  blocked?: CommentTriggerOutcome[];
  hasAllMembersMention?: boolean;
  /** The draft markdown, used only to name blocked targets (see CommentTriggerChips). */
  draftContent?: string;
  onActionChange: (agentId: string, action: RecipientAction) => void;
  onSubmit: () => void;
  disabled?: boolean;
  loading?: boolean;
  /** Blocked on an upload the user started; see SubmitButton. */
  busy?: boolean;
  /** Why sending is blocked; shown as the tooltip instead of `tooltip`. */
  disabledReason?: ReactNode;
  tooltip?: ReactNode;
}

function CommentSendButton({
  recipients,
  blocked = [],
  hasAllMembersMention = false,
  draftContent = "",
  onActionChange,
  onSubmit,
  disabled,
  loading,
  busy,
  disabledReason,
  tooltip,
}: CommentSendButtonProps) {
  const { t } = useT("issues");
  const reduceMotion = useReducedMotion();
  const { label, tone } = sendIntent(recipients, t);
  const [contentRef, contentWidth] = useMeasuredWidth();
  const blockedLabels = useMemo(() => mentionLabelsByTarget(draftContent), [draftContent]);

  const isDisabled = !!(disabled || loading || busy);
  const showReason = isDisabled && !loading && disabledReason != null && disabledReason !== false;
  const hasMenu = recipients.length > 0 || blocked.length > 0 || hasAllMembersMention;
  const heads = recipients.slice(0, MAX_HEADS);
  const overflow = recipients.length - heads.length;
  const overlap = Math.round(AVATAR_SIZE * 0.3);
  const spinning = !!(loading || busy);
  const hasLead = spinning || heads.length > 0 || blocked.length > 0;

  const instant: Transition = { duration: 0 };
  const widthTransition: Transition = reduceMotion ? instant : { type: "spring", duration: 0.3, bounce: 0 };
  const swapIn: Transition = reduceMotion ? instant : { duration: 0.2, ease: EASE_OUT };
  const swapOut: Transition = reduceMotion ? instant : { duration: 0.14, ease: "easeIn" };
  const blur = (px: number) => (reduceMotion ? "blur(0px)" : `blur(${px}px)`);
  const nudge = (y: number) => (reduceMotion ? 0 : y);
  const headIn = { scale: reduceMotion ? 1 : 0.6, opacity: 0 };

  const ringed = cn("inline-flex rounded-full ring-2 transition-shadow duration-200", TONE_RING[tone]);
  const bubble = { width: AVATAR_SIZE, height: AVATAR_SIZE };
  const leadItems: { key: string; className: string; style?: CSSProperties; content: ReactNode }[] = spinning
    ? [{ key: "spinner", className: "inline-flex items-center justify-center", style: bubble, content: <Loader2 className="size-3.5 animate-spin" /> }]
    : [
        ...heads.map((entry, i) => ({
          key: entry.agent.id,
          className: ringed,
          style: { marginLeft: i === 0 ? 0 : -overlap },
          content: <TriggerAgentAvatar agent={entry.agent} suppressed={entry.action === "skip"} showDot={false} />,
        })),
        ...(overflow > 0 ? [{
          key: "overflow",
          className: cn(ringed, "items-center justify-center bg-muted font-medium tabular-nums text-muted-foreground"),
          style: { ...bubble, marginLeft: -overlap, fontSize: Math.max(9, Math.round(AVATAR_SIZE * 0.45)) },
          content: `+${overflow}`,
        }] : []),
        ...(blocked.length > 0 ? [{
          key: "blocked",
          className: cn(ringed, "items-center justify-center bg-warning text-foreground"),
          style: { ...bubble, marginLeft: heads.length > 0 ? -overlap : 0 },
          content: <TriangleAlert className="size-2.5" />,
        }] : []),
      ];

  const mainButton = (
    <button
      type="button"
      data-slot="comment-send"
      data-tone={tone}
      aria-label={label}
      disabled={isDisabled && !showReason}
      aria-disabled={showReason || (busy && !showReason) || undefined}
      aria-busy={spinning || undefined}
      onPointerDown={keepFocusInComposer}
      onMouseDown={keepFocusInComposer}
      onClick={isDisabled ? undefined : onSubmit}
      className={cn(
        "flex min-w-0 items-center rounded-l-full outline-none transition-colors duration-150 hover:bg-current/10 focus-visible:ring-3 focus-visible:ring-ring/50",
        isDisabled ? "cursor-not-allowed" : "cursor-pointer",
      )}
    >
      <motion.span
        aria-hidden
        className="block overflow-hidden"
        initial={false}
        animate={{ width: contentWidth ?? "auto" }}
        transition={widthTransition}
      >
        <span
          ref={contentRef}
          className={cn("relative flex w-max items-center gap-1.5 pr-2.5", hasLead ? "pl-1" : "pl-3")}
        >
          <span className="flex items-center empty:hidden">
            <AnimatePresence initial={false} mode="popLayout">
              {leadItems.map(({ key, className, style, content }) => (
                <motion.span
                  key={key}
                  style={style}
                  className={className}
                  initial={headIn}
                  animate={{ scale: 1, opacity: 1 }}
                  exit={{ ...headIn, transition: swapOut }}
                  transition={swapIn}
                >
                  {content}
                </motion.span>
              ))}
            </AnimatePresence>
          </span>
          <span className="relative flex items-center">
            <AnimatePresence initial={false} mode="popLayout">
              <motion.span
                key={label}
                className="block whitespace-nowrap text-label font-medium"
                initial={{ opacity: 0, y: nudge(6), filter: blur(2) }}
                animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
                exit={{ opacity: 0, y: nudge(-6), filter: blur(2), transition: swapOut }}
                transition={swapIn}
              >
                {label}
              </motion.span>
            </AnimatePresence>
          </span>
        </span>
      </motion.span>
    </button>
  );

  const tooltipContent = showReason ? disabledReason : tooltip;

  return (
    <div
      className={cn(
        "inline-flex h-[var(--button-height-sm)] shrink-0 items-stretch rounded-full transition-[background-color,color,opacity] duration-200 ease-out motion-reduce:transition-none",
        TONE_CLASS[tone],
        isDisabled && "opacity-50",
      )}
    >
      {tooltipContent ? (
        <Tooltip>
          <TooltipTrigger render={mainButton} />
          <TooltipContent side="top">{tooltipContent}</TooltipContent>
        </Tooltip>
      ) : mainButton}
      <span aria-hidden className="my-1.5 w-px bg-current opacity-25" />
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <button
              type="button"
              aria-label={t(($) => $.comment.send_choose_aria)}
              disabled={!hasMenu}
              className="flex items-center rounded-r-full pr-2 pl-1.5 outline-none transition-colors duration-150 hover:bg-current/10 focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 aria-expanded:bg-current/10"
            >
              <ChevronDown className="size-3.5" />
            </button>
          }
        />
        <DropdownMenuContent side="top" align="end" className="w-80">
          {recipients.map((entry, i) => (
            <RecipientGroup key={entry.agent.id} entry={entry} separated={i > 0} onActionChange={onActionChange} t={t} />
          ))}
          {hasAllMembersMention && (
            <>
              {recipients.length > 0 && <DropdownMenuSeparator />}
              <div className="flex items-center gap-1.5 px-1.5 py-1.5 text-caption text-muted-foreground">
                <Users className="size-3.5 shrink-0" />
                <span>{t(($) => $.comment.all_members_notice)}</span>
              </div>
            </>
          )}
          {blocked.length > 0 && (recipients.length > 0 || hasAllMembersMention) && <DropdownMenuSeparator />}
          {blocked.map((outcome) => {
            const name = blockedLabels.get(`${outcome.target_type}:${outcome.target_id}`);
            return (
              <div key={`${outcome.target_type}:${outcome.target_id}`} className="flex items-start gap-2 px-1.5 py-1.5 text-caption">
                <TriangleAlert className="mt-0.5 size-3.5 shrink-0 text-warning" />
                <span className="flex min-w-0 flex-col gap-0.5">
                  <span className="font-medium text-foreground">
                    {name ? `${name} · ${blockedShortReasonLabel(outcome.reason_code, t)}` : blockedShortReasonLabel(outcome.reason_code, t)}
                  </span>
                  <span className="text-muted-foreground">{blockedReasonLabel(outcome.reason_code, t)}</span>
                </span>
              </div>
            );
          })}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

function RecipientGroup({
  entry,
  separated,
  onActionChange,
  t,
}: {
  entry: RecipientEntry;
  separated: boolean;
  onActionChange: (agentId: string, action: RecipientAction) => void;
  t: IssuesT;
}) {
  const presenceLine = useTriggerPresenceLine(entry.agent.id, t);
  const state = stateLabel(entry.state, t);
  return (
    <>
      {separated && <DropdownMenuSeparator />}
      <DropdownMenuGroup>
        <DropdownMenuLabel className="flex min-w-0 items-center gap-2">
          <TriggerAgentAvatar agent={entry.agent} suppressed={false} />
          <span className="min-w-0 truncate text-foreground">{entry.agent.name}</span>
          <span className="shrink-0 font-normal">{state ?? sourceLabel(entry.agent.source, t)}</span>
        </DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={entry.action}
          onValueChange={(value) => onActionChange(entry.agent.id, value as RecipientAction)}
        >
          {entry.actions.map((action) => {
            const description = actionDescription(action, entry, presenceLine, t);
            return (
              <DropdownMenuRadioItem key={action} value={action} className="items-start py-1.5">
                <span className="flex min-w-0 flex-col gap-0.5">
                  <span className={cn(action === "restart" && "text-destructive")}>{actionLabel(action, entry.state, t)}</span>
                  {description && <span className="text-caption text-muted-foreground">{description}</span>}
                </span>
              </DropdownMenuRadioItem>
            );
          })}
        </DropdownMenuRadioGroup>
      </DropdownMenuGroup>
    </>
  );
}

export { CommentSendButton, type CommentSendButtonProps };
