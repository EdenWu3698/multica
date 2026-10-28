"use client";

import { useLayoutEffect, useMemo, useRef, useState, type ReactElement, type ReactNode } from "react";
import { AnimatePresence, motion, useIsPresent, useReducedMotion, type Transition } from "motion/react";
import { ChevronDown, TriangleAlert, Users } from "lucide-react";
import type { CommentTriggerPreviewAgent, CommentTriggerOutcome } from "@multica/core/types";
import type { AgentRunState, RecipientAction } from "@multica/core/issues/run-steering";
import { useAgentPresenceDetail } from "@multica/core/agents";
import { mentionLabelsByTarget } from "@multica/core/issues/comment-trigger-outcomes";
import { useCurrentWorkspace } from "@multica/core/paths";
import { ActorAvatar as ActorAvatarBase } from "@multica/ui/components/common/actor-avatar";
import { AVATAR_SIZE_PX } from "@multica/ui/lib/avatar-size";
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
import { cn } from "@multica/ui/lib/utils";
import { AgentStatusDot } from "../../common/actor-avatar";
import { useT } from "../../i18n";
import { blockedReasonLabel, blockedShortReasonLabel } from "../blocked-trigger-copy";
import type { RecipientEntry } from "../hooks/use-recipient-actions";

// The chips are the one place a composer says what sending will do; the send
// button never changes. Each recipient's chip says what Send will do to it,
// and its menu offers only what that recipient's current state allows: a
// running turn can take the message now, after the turn, or instead of it; an
// idle or queued agent can only start (or fold the message into its queued
// run) or be skipped. Up to two recipients get a chip each; three or more
// collapse to one avatar-stack chip whose menu groups every recipient's
// choices. A mention that will not trigger gets its own amber chip.
// The avatars render at the `xs` tier; the `+N` overflow chip and stack
// overlap reuse that tier's pixel diameter so the stack lines up exactly.
const AVATAR_SIZE = AVATAR_SIZE_PX.xs;
const MAX_STACK_HEADS = 4;
const COLLAPSE_AT = 3;
const EASE_OUT: [number, number, number, number] = [0.32, 0.72, 0, 1];

interface CommentTriggerChipsProps {
  recipients: RecipientEntry[];
  // Explicit @agent / @squad mentions that will NOT trigger if posted as-is
  // (MUL-4525 §2). Each renders as a named warning chip so the user sees WHICH
  // target won't run and why, not a silent no-op after sending.
  blocked?: CommentTriggerOutcome[];
  /** Whether the draft contains the structured @all member broadcast. */
  hasAllMembersMention?: boolean;
  // The draft markdown, used only to label each blocked target with the name the
  // user typed in its mention markup. The server omits blocked target names
  // (enumeration-safety); this is the user's own text, so it discloses nothing new.
  draftContent?: string;
  onActionChange: (agentId: string, action: RecipientAction) => void;
}

type IssuesT = ReturnType<typeof useT<"issues">>["t"];

function sourceLabel(source: string, t: IssuesT): string {
  switch (source) {
    case "issue_assignee":
      return t(($) => $.comment.trigger_source_issue_assignee);
    case "mention_agent":
      return t(($) => $.comment.trigger_source_mention_agent);
    case "mention_squad_leader":
      return t(($) => $.comment.trigger_source_mention_squad_leader);
    default:
      return t(($) => $.comment.trigger_source_unknown);
  }
}

// Presence is display metadata only — the trigger list itself is always the
// backend preview. Online-ish agents start right away; offline ones queue.
function useTriggerPresenceLine(agentId: string, t: IssuesT): string | null {
  const ws = useCurrentWorkspace();
  const detail = useAgentPresenceDetail(ws?.id, agentId);
  if (detail === "loading") return null;
  return detail.availability === "online" || detail.availability === "unstable"
    ? t(($) => $.comment.trigger_starts_now)
    : t(($) => $.comment.trigger_starts_when_online);
}

function stateLabel(state: AgentRunState, t: IssuesT): string | null {
  switch (state.kind) {
    case "running":
      return t(($) => $.comment.recipient_state_running);
    case "starting":
      return t(($) => $.comment.recipient_state_starting);
    case "queued":
      return t(($) => $.comment.recipient_state_queued);
    default:
      return null;
  }
}

function actionLabel(action: RecipientAction, state: AgentRunState, t: IssuesT): string {
  switch (action) {
    case "steer":
      return t(($) => $.comment.recipient_steer);
    case "after_run":
      return t(($) => $.comment.recipient_after_run);
    case "restart":
      return t(($) => $.comment.recipient_restart);
    case "skip":
      return t(($) => $.comment.trigger_wont_trigger);
    default:
      return state.kind === "queued"
        ? t(($) => $.comment.recipient_join)
        : t(($) => $.comment.trigger_will_start);
  }
}

// Only choices whose consequence is not obvious from the label carry a line.
function actionDescription(action: RecipientAction, entry: RecipientEntry, presenceLine: string | null, t: IssuesT): string | null {
  const name = entry.agent.name;
  switch (action) {
    case "steer":
      return t(($) => $.comment.recipient_steer_desc, { name });
    case "after_run":
      return t(($) => $.comment.recipient_after_run_desc);
    case "restart":
      return t(($) => $.comment.recipient_restart_desc);
    case "start":
      return entry.state.kind === "queued" ? t(($) => $.comment.recipient_join_desc, { name }) : presenceLine;
    default:
      return null;
  }
}

type ChipTone = "neutral" | "steer" | "restart" | "warn";

const CHIP_BASE =
  "inline-flex h-6 min-w-0 max-w-full items-center gap-1.5 rounded-md border border-transparent px-1.5 text-micro font-medium transition-[background-color,border-color,color] duration-200 ease-out motion-reduce:transition-none";

const CHIP_TONE: Record<ChipTone, string> = {
  neutral: "text-muted-foreground hover:bg-muted hover:text-foreground aria-expanded:bg-muted aria-expanded:text-foreground",
  steer: "border-brand/28 bg-brand/7 text-foreground hover:bg-brand/12 aria-expanded:bg-brand/12 dark:border-brand/45 dark:bg-brand/12 dark:hover:bg-brand/18",
  restart: "bg-destructive/8 text-destructive hover:bg-destructive/12 aria-expanded:bg-destructive/12 dark:bg-destructive/14 dark:hover:bg-destructive/20",
  warn: "bg-warning/14 text-foreground dark:bg-warning/18",
};

function actionTone(action: RecipientAction): ChipTone {
  if (action === "steer") return "steer";
  if (action === "restart") return "restart";
  return "neutral";
}

function groupTone(recipients: RecipientEntry[]): ChipTone {
  if (recipients.some((r) => r.action === "restart")) return "restart";
  if (recipients.some((r) => r.action === "steer")) return "steer";
  return "neutral";
}

function useChipMotion() {
  const reduceMotion = useReducedMotion();
  const instant: Transition = { duration: 0 };
  return {
    reduceMotion,
    enter: reduceMotion ? instant : ({ duration: 0.18, ease: EASE_OUT } satisfies Transition),
    exit: reduceMotion ? instant : ({ duration: 0.12, ease: "easeIn" } satisfies Transition),
    resize: reduceMotion ? instant : ({ type: "spring", duration: 0.3, bounce: 0 } satisfies Transition),
  };
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

// A chip's sentence changes in place when the user picks another choice: the
// old text lifts out, the new one rises in, and the chip's width follows.
function ChipLabel({ text }: { text: string }) {
  const { reduceMotion, enter, exit, resize } = useChipMotion();
  const [ref, width] = useMeasuredWidth();
  const nudge = reduceMotion ? 0 : 4;
  return (
    <motion.span
      className="block min-w-0 overflow-hidden"
      initial={false}
      animate={{ width: width ?? "auto" }}
      transition={resize}
    >
      <span ref={ref} className="relative block w-max max-w-60">
        <AnimatePresence initial={false} mode="popLayout">
          <motion.span
            key={text}
            className="block truncate"
            initial={{ opacity: 0, y: nudge }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -nudge, transition: exit }}
            transition={enter}
          >
            {text}
          </motion.span>
        </AnimatePresence>
      </span>
    </motion.span>
  );
}

export function CommentTriggerChips({
  recipients,
  blocked = [],
  hasAllMembersMention = false,
  draftContent = "",
  onActionChange,
}: CommentTriggerChipsProps) {
  const { t } = useT("issues");
  // Blocked outcomes carry no name (enumeration-safety); recover the label the
  // user typed from their own draft so each chip can say which target it is.
  const blockedLabels = useMemo(() => mentionLabelsByTarget(draftContent), [draftContent]);

  // Loading and errors render nothing: the preview is an enhancement, and
  // any interim chrome here reads as composer noise.
  const chips: { key: string; node: ReactNode }[] = [];
  if (hasAllMembersMention) {
    chips.push({
      key: "all-members",
      node: (
        <span className="inline-flex h-6 min-w-0 max-w-full items-center gap-1.5 rounded-md px-1.5 text-micro font-medium text-muted-foreground">
          <Users className="size-3 shrink-0" />
          <span className="truncate">{t(($) => $.comment.all_members_notice)}</span>
        </span>
      ),
    });
  }
  if (recipients.length >= COLLAPSE_AT) {
    chips.push({ key: "recipients", node: <CollapsedRecipientsChip recipients={recipients} onActionChange={onActionChange} t={t} /> });
  } else {
    for (const entry of recipients) {
      chips.push({ key: `agent:${entry.agent.id}`, node: <SingleRecipientChip entry={entry} onActionChange={onActionChange} t={t} /> });
    }
  }
  for (const outcome of blocked) {
    const key = `${outcome.target_type}:${outcome.target_id}`;
    chips.push({ key: `blocked:${key}`, node: <BlockedTriggerChip outcome={outcome} label={blockedLabels.get(key)} t={t} /> });
  }

  return (
    <div className="flex min-w-0 items-center gap-1.5 empty:hidden">
      <AnimatePresence initial={false} mode="popLayout">
        {chips.map(({ key, node }) => (
          <ChipSlot key={key}>{node}</ChipSlot>
        ))}
      </AnimatePresence>
    </div>
  );
}

// A chip on its way out is inert: its recipient is already gone, so a choice
// made in its menu would be dropped.
function ChipSlot({ children }: { children: ReactNode }) {
  const isPresent = useIsPresent();
  const { reduceMotion, enter, exit, resize } = useChipMotion();
  const hidden = { opacity: 0, scale: reduceMotion ? 1 : 0.94 };
  return (
    <motion.div
      layout="position"
      className="flex min-w-0 origin-left"
      initial={hidden}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ ...hidden, transition: exit }}
      transition={{ ...enter, layout: resize }}
      inert={!isPresent}
      aria-hidden={!isPresent || undefined}
    >
      {children}
    </motion.div>
  );
}

// One blocked mention: named like an allowed chip ("Go"), but amber, with a
// short reason ("Not found or no permission") instead of "will start", so a
// refused @mention reads as a specific "won't run" rather than a vague count.
function BlockedTriggerChip({
  outcome,
  label,
  t,
}: {
  outcome: CommentTriggerOutcome;
  label?: string;
  t: IssuesT;
}) {
  const shortReason = blockedShortReasonLabel(outcome.reason_code, t);
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            className={cn(CHIP_BASE, CHIP_TONE.warn)}
            aria-label={
              label
                ? t(($) => $.comment.trigger_blocked_chip_aria, { name: label, reason: shortReason })
                : shortReason
            }
          >
            <TriangleAlert className="size-3 shrink-0 text-warning" />
            {label ? (
              <span className="inline-flex min-w-0 items-center gap-1">
                <span className="truncate">{label}</span>
                <span className="shrink-0">·</span>
                <span className="shrink-0">{shortReason}</span>
              </span>
            ) : (
              <span className="truncate">{shortReason}</span>
            )}
          </span>
        }
      />
      <TooltipContent side="top" className="max-w-72 text-caption">
        {blockedReasonLabel(outcome.reason_code, t)}
      </TooltipContent>
    </Tooltip>
  );
}

// One recipient's choices: a header naming the agent and its state, then a
// radio item per choice. Shared by the single chip's menu and the collapsed
// chip's grouped menu so both read the same.
function RecipientChoices({
  entry,
  onActionChange,
  t,
}: {
  entry: RecipientEntry;
  onActionChange: (agentId: string, action: RecipientAction) => void;
  t: IssuesT;
}) {
  const presenceLine = useTriggerPresenceLine(entry.agent.id, t);
  const state = stateLabel(entry.state, t);
  return (
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
            <div key={action}>
              {action === "skip" && <DropdownMenuSeparator />}
              <DropdownMenuRadioItem value={action} className="items-start py-1.5">
                <span className="flex min-w-0 flex-col gap-0.5">
                  <span className={cn(action === "restart" && "text-destructive")}>{actionLabel(action, entry.state, t)}</span>
                  {description && <span className="text-caption text-muted-foreground">{description}</span>}
                </span>
              </DropdownMenuRadioItem>
            </div>
          );
        })}
      </DropdownMenuRadioGroup>
    </DropdownMenuGroup>
  );
}

function ChipMenu({ trigger, children }: { trigger: ReactElement; children: ReactNode }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={trigger} />
      <DropdownMenuContent side="top" align="start" className="w-80">
        {children}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function SingleRecipientChip({
  entry,
  onActionChange,
  t,
}: {
  entry: RecipientEntry;
  onActionChange: (agentId: string, action: RecipientAction) => void;
  t: IssuesT;
}) {
  const label = actionLabel(entry.action, entry.state, t);
  // The avatar carries "who"; the sentence carries only the outcome, so it
  // stays short and never truncates on long agent names.
  return (
    <ChipMenu
      trigger={
        <button
          type="button"
          aria-label={t(($) => $.comment.trigger_chip_aria, { name: entry.agent.name, state: label })}
          data-tone={actionTone(entry.action)}
          className={cn(CHIP_BASE, "cursor-pointer", CHIP_TONE[actionTone(entry.action)])}
        >
          <TriggerAgentAvatar agent={entry.agent} suppressed={entry.action === "skip"} />
          <ChipLabel text={label} />
          <ChevronDown aria-hidden className="size-3 shrink-0 text-faint-foreground" />
        </button>
      }
    >
      <RecipientChoices entry={entry} onActionChange={onActionChange} t={t} />
    </ChipMenu>
  );
}

function CollapsedRecipientsChip({
  recipients,
  onActionChange,
  t,
}: {
  recipients: RecipientEntry[];
  onActionChange: (agentId: string, action: RecipientAction) => void;
  t: IssuesT;
}) {
  const activeCount = recipients.filter((r) => r.action !== "skip").length;
  const heads = recipients.slice(0, MAX_STACK_HEADS);
  const overflow = recipients.length - heads.length;
  // Mirror AgentAvatarStack: ~30% overlap reads as "stacked" without
  // obscuring the next avatar.
  const overlap = Math.round(AVATAR_SIZE * 0.3);
  const sentence =
    activeCount === 0
      ? t(($) => $.comment.trigger_none_will_trigger)
      : t(($) => $.comment.recipient_count, { count: activeCount });
  const tone = groupTone(recipients);

  return (
    <ChipMenu
      trigger={
        <button
          type="button"
          data-tone={tone}
          className={cn(CHIP_BASE, "cursor-pointer", CHIP_TONE[tone])}
        >
          <span className="inline-flex items-center">
            {heads.map((entry, i) => (
              <span
                key={entry.agent.id}
                style={{ marginLeft: i === 0 ? 0 : -overlap }}
                className="inline-flex rounded-full ring-2 ring-background"
              >
                <TriggerAgentAvatar agent={entry.agent} suppressed={entry.action === "skip"} showDot={false} />
              </span>
            ))}
            {overflow > 0 && (
              <span
                style={{
                  marginLeft: -overlap,
                  width: AVATAR_SIZE,
                  height: AVATAR_SIZE,
                  fontSize: Math.max(9, Math.round(AVATAR_SIZE * 0.45)),
                }}
                className="inline-flex items-center justify-center rounded-full bg-muted font-medium tabular-nums text-muted-foreground ring-2 ring-background"
              >
                +{overflow}
              </span>
            )}
          </span>
          <ChipLabel text={sentence} />
          <ChevronDown aria-hidden className="size-3 shrink-0 text-faint-foreground" />
        </button>
      }
    >
      {recipients.map((entry, i) => (
        <div key={entry.agent.id}>
          {i > 0 && <DropdownMenuSeparator />}
          <RecipientChoices entry={entry} onActionChange={onActionChange} t={t} />
        </div>
      ))}
    </ChipMenu>
  );
}

function TriggerAgentAvatar({
  agent,
  suppressed,
  showDot = true,
}: {
  agent: CommentTriggerPreviewAgent;
  suppressed: boolean;
  showDot?: boolean;
}) {
  return (
    <span
      className={cn(
        "relative inline-flex shrink-0 transition-[opacity,filter] duration-200 motion-reduce:transition-none",
        suppressed && "opacity-40 grayscale",
      )}
    >
      <ActorAvatarBase
        name={agent.name}
        initials=""
        avatarUrl={agent.avatar_url}
        isAgent
        size="xs"
      />
      {showDot && !suppressed && <AgentStatusDot agentId={agent.id} size="xs" />}
    </span>
  );
}
