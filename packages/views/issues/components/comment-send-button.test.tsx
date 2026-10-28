import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen, within } from "@testing-library/react";
import type { AgentTask, CommentTriggerOutcome, CommentTriggerPreviewAgent } from "@multica/core/types";
import {
  recipientActions,
  resolveRecipientAction,
  type AgentRunState,
  type RecipientAction,
} from "@multica/core/issues/run-steering";
import type { RecipientEntry } from "../hooks/use-recipient-actions";
import { renderWithI18n } from "../../test/i18n";
import { CommentSendButton } from "./comment-send-button";

vi.mock("@multica/core/agents", () => ({
  useAgentPresenceDetail: () => ({ availability: "online", workload: "idle" }),
}));

vi.mock("@multica/core/paths", () => ({
  useCurrentWorkspace: () => ({ id: "ws-1" }),
}));

vi.mock("../../common/actor-avatar", () => ({
  AgentStatusDot: () => null,
}));

const walt: CommentTriggerPreviewAgent = { id: "agent-1", name: "Walt", source: "issue_assignee", reason: "" };
const bob: CommentTriggerPreviewAgent = { id: "agent-2", name: "Bob", source: "mention_agent", reason: "" };

function turn(status: AgentTask["status"]): AgentTask {
  return { id: `turn-${status}`, agent_id: "agent-1", issue_id: "issue", status, priority: 0,
    created_at: "2026-09-23T00:00:00Z", dispatched_at: null, started_at: null, completed_at: null,
    result: null, error: null } as AgentTask;
}
const running: AgentRunState = { kind: "running", task: turn("running"), steerable: true };
const queued: AgentRunState = { kind: "queued", task: turn("queued") };
const idle: AgentRunState = { kind: "idle" };

function entry(agent: CommentTriggerPreviewAgent, state: AgentRunState, chosen?: RecipientAction): RecipientEntry {
  const opts = { canSteer: true, canRestart: true, steerByDefault: true };
  return { agent, state, action: resolveRecipientAction(state, chosen, opts), actions: recipientActions(state, opts) };
}

function renderButton(recipients: RecipientEntry[], extra: { blocked?: CommentTriggerOutcome[]; onActionChange?: () => void } = {}) {
  return renderWithI18n(
    <CommentSendButton recipients={recipients} blocked={extra.blocked} onActionChange={extra.onActionChange ?? vi.fn()} onSubmit={vi.fn()} />,
  );
}

function sendButton(): HTMLElement {
  return document.querySelector<HTMLElement>('[data-slot="comment-send"]')!;
}

describe("CommentSendButton", () => {
  it("is a plain comment with no menu when nobody is triggered", () => {
    renderButton([]);
    expect(sendButton()).toHaveAccessibleName("Comment");
    expect(sendButton()).toHaveAttribute("data-tone", "neutral");
    expect(screen.getByRole("button", { name: "Choose what each agent does with this message" })).toBeDisabled();
  });

  it("names what sending does to a single recipient", () => {
    const { rerender } = renderButton([entry(walt, idle)]);
    expect(sendButton()).toHaveAccessibleName("Send and start");
    rerender(<CommentSendButton recipients={[entry(walt, queued)]} onActionChange={vi.fn()} onSubmit={vi.fn()} />);
    expect(sendButton()).toHaveAccessibleName("Include when it starts");
    rerender(<CommentSendButton recipients={[entry(walt, running)]} onActionChange={vi.fn()} onSubmit={vi.fn()} />);
    expect(sendButton()).toHaveAccessibleName("Add to current run");
    expect(sendButton()).toHaveAttribute("data-tone", "steer");
    rerender(<CommentSendButton recipients={[entry(walt, running, "restart")]} onActionChange={vi.fn()} onSubmit={vi.fn()} />);
    expect(sendButton()).toHaveAccessibleName("Stop and start over");
    expect(sendButton()).toHaveAttribute("data-tone", "restart");
  });

  it("counts recipients, and a stop anywhere turns the whole send red", () => {
    const { rerender } = renderButton([entry(walt, running), entry(bob, idle)]);
    expect(sendButton()).toHaveAccessibleName("Send to 2 agents");
    expect(sendButton()).toHaveAttribute("data-tone", "steer");
    rerender(<CommentSendButton recipients={[entry(walt, running, "restart"), entry(bob, idle)]} onActionChange={vi.fn()} onSubmit={vi.fn()} />);
    expect(sendButton()).toHaveAccessibleName("Stop and send");
    expect(sendButton()).toHaveAttribute("data-tone", "restart");
  });

  it("falls back to a plain comment when every recipient is skipped", () => {
    renderButton([entry(walt, idle, "skip")]);
    expect(sendButton()).toHaveAccessibleName("Comment");
  });

  it("groups each recipient's choices in one menu", async () => {
    const onActionChange = vi.fn();
    renderButton([entry(walt, running), entry(bob, idle)], { onActionChange });
    fireEvent.click(screen.getByRole("button", { name: "Choose what each agent does with this message" }));
    const menu = await screen.findByRole("menu");
    expect(within(menu).getAllByRole("menuitemradio")).toHaveLength(6);
    expect(within(menu).getByText("Walt")).toBeInTheDocument();
    expect(within(menu).getByText("Bob")).toBeInTheDocument();
    fireEvent.click(within(menu).getByRole("menuitemradio", { name: /Stop and start over/ }));
    expect(onActionChange).toHaveBeenCalledWith("agent-1", "restart");
  });

  it("lists a blocked mention with its reason", async () => {
    renderButton([entry(walt, idle)], {
      blocked: [{ target_type: "agent", target_id: "agent-9", status: "blocked", reason_code: "runtime_offline" } as CommentTriggerOutcome],
    });
    fireEvent.click(screen.getByRole("button", { name: "Choose what each agent does with this message" }));
    const menu = await screen.findByRole("menu");
    expect(within(menu).getByText("Runtime offline")).toBeInTheDocument();
    expect(within(menu).getByText("This target's runtime is offline")).toBeInTheDocument();
  });
});
