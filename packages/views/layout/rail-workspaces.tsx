"use client";

import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { MoreHorizontal } from "lucide-react";
import { api } from "@multica/core/api";
import {
  inboxUnreadSummaryEnabled,
  inboxUnreadSummaryOptions,
  unreadWorkspaceIds,
} from "@multica/core/inbox/queries";
import { paths, useCurrentWorkspace } from "@multica/core/paths";
import type { Invitation, Workspace } from "@multica/core/types";
import {
  myInvitationListOptions,
  workspaceKeys,
  workspaceListOptions,
} from "@multica/core/workspace/queries";
import { TooltipHint } from "@multica/ui/components/common/tooltip-hint";
import { Button } from "@multica/ui/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@multica/ui/components/ui/popover";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../i18n";
import { AppLink, useNavigation } from "../navigation";
import { WorkspaceAvatar } from "../workspace/workspace-avatar";
import { RAIL_ICON_BUTTON_CLASS } from "./rail-styles";

// Stable empty defaults: an inline `= []` is a new array every render.
const EMPTY_WORKSPACES: Workspace[] = [];
const EMPTY_INVITATIONS: Invitation[] = [];
const EMPTY_INBOX_SUMMARY: Awaited<ReturnType<typeof api.getInboxUnreadSummary>> = [];

/** Workspaces shown as tiles; the rest go behind the overflow menu. */
const MAX_RAIL_WORKSPACES = 4;

const TILE_CLASS =
  "relative inline-flex size-[34px] shrink-0 items-center justify-center rounded-[10px] outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring";

function UnreadDot() {
  return (
    <span
      aria-hidden
      className="absolute -top-0.5 -right-0.5 size-2 rounded-full bg-brand ring-2 ring-app-shell"
    />
  );
}

function WorkspaceTile({
  workspace,
  current,
  unread,
}: {
  workspace: Workspace;
  current: boolean;
  unread: boolean;
}) {
  return (
    <TooltipHint label={workspace.name} side="right">
      <AppLink
        href={paths.workspace(workspace.slug).issues()}
        aria-label={workspace.name}
        aria-current={current ? "page" : undefined}
        className={cn(
          TILE_CLASS,
          // The active-workspace marker hangs off the rail's leading edge.
          current &&
            "before:absolute before:top-[7px] before:bottom-[7px] before:-left-[15px] before:w-1 before:rounded-r-sm before:bg-foreground",
        )}
      >
        <WorkspaceAvatar
          name={workspace.name}
          avatarUrl={workspace.avatar_url}
          className={cn(
            "h-[34px] w-[34px] rounded-[10px] border-surface-border bg-surface-raised text-body shadow-[var(--surface-shadow)]",
            current ? "text-foreground" : "text-muted-foreground",
          )}
        />
        {unread && <UnreadDot />}
      </AppLink>
    </TooltipHint>
  );
}

function useInvitationActions(invitations: Invitation[]) {
  const queryClient = useQueryClient();
  const { push } = useNavigation();
  const accept = useMutation({
    mutationFn: (id: string) => api.acceptInvitation(id),
    // Land inside the joined workspace; staying put and just adding a tile
    // reads as nothing happened (MUL-820).
    onSuccess: async (_, invitationId) => {
      const invitation = invitations.find((i) => i.id === invitationId);
      queryClient.invalidateQueries({ queryKey: workspaceKeys.myInvitations() });
      // staleTime: 0 forces a real fetch — the joined workspace has to be in
      // the list before its slug can be resolved.
      const list = await queryClient.fetchQuery({ ...workspaceListOptions(), staleTime: 0 });
      const joined = invitation ? list.find((w) => w.id === invitation.workspace_id) : null;
      if (joined) push(paths.workspace(joined.slug).issues());
    },
    // "Invitation is not pending": it was concluded on another surface while
    // this tile was on screen. Refetch so the stale tile drops.
    onError: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.myInvitations() });
    },
  });
  const decline = useMutation({
    mutationFn: (id: string) => api.declineInvitation(id),
    // Success drops the declined tile; failure means it was concluded
    // elsewhere. Either way the list is stale.
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.myInvitations() });
    },
  });
  return { accept, decline };
}

function InvitationTile({
  invitation,
  actions,
}: {
  invitation: Invitation;
  actions: ReturnType<typeof useInvitationActions>;
}) {
  const { t } = useT("layout");
  const name = invitation.workspace_name ?? t(($) => $.sidebar.invitation_workspace_fallback);
  const pending = actions.accept.isPending || actions.decline.isPending;
  return (
    <Popover>
      <TooltipHint label={t(($) => $.rail.invitation_tooltip, { workspace: name })} side="right">
        <PopoverTrigger
          aria-label={t(($) => $.rail.invitation_tooltip, { workspace: name })}
          className={cn(
            TILE_CLASS,
            "cursor-pointer border-[1.5px] border-dashed border-faint-foreground text-body font-semibold text-muted-foreground transition-colors hover:border-muted-foreground hover:text-foreground data-popup-open:border-muted-foreground",
          )}
        >
          {name.charAt(0).toUpperCase()}
          <UnreadDot />
        </PopoverTrigger>
      </TooltipHint>
      <PopoverContent side="right" align="start" sideOffset={10} className="w-64 gap-3 p-3">
        <div className="flex items-center gap-2.5">
          <WorkspaceAvatar name={name} size="lg" className="rounded-lg" />
          <div className="min-w-0 flex-1">
            <p className="truncate text-body font-medium">{name}</p>
            <p className="truncate text-caption text-muted-foreground">
              {invitation.inviter_name
                ? t(($) => $.rail.invitation_from, { name: invitation.inviter_name })
                : t(($) => $.sidebar.pending_invitations_label)}
            </p>
          </div>
        </div>
        <div className="flex justify-end gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={pending}
            onClick={() => actions.decline.mutate(invitation.id)}
          >
            {t(($) => $.sidebar.invitation_decline)}
          </Button>
          <Button size="sm" disabled={pending} onClick={() => actions.accept.mutate(invitation.id)}>
            {t(($) => $.sidebar.invitation_join)}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function OverflowWorkspaces({
  workspaces,
  unreadIds,
}: {
  workspaces: Workspace[];
  unreadIds: Set<string>;
}) {
  const { t } = useT("layout");
  const anyUnread = workspaces.some((ws) => unreadIds.has(ws.id));
  return (
    <DropdownMenu>
      <TooltipHint label={t(($) => $.rail.more_workspaces)} side="right">
        <DropdownMenuTrigger
          aria-label={t(($) => $.rail.more_workspaces)}
          className={cn(RAIL_ICON_BUTTON_CLASS, "size-[34px]")}
        >
          <MoreHorizontal />
          {anyUnread && <UnreadDot />}
        </DropdownMenuTrigger>
      </TooltipHint>
      <DropdownMenuContent side="right" align="start" sideOffset={10} className="w-auto min-w-56">
        <DropdownMenuGroup>
          <DropdownMenuLabel className="text-caption text-muted-foreground">
            {t(($) => $.sidebar.workspaces_label)}
          </DropdownMenuLabel>
          {workspaces.map((ws) => (
            <DropdownMenuItem
              key={ws.id}
              render={<AppLink href={paths.workspace(ws.slug).issues()} />}
            >
              <WorkspaceAvatar name={ws.name} avatarUrl={ws.avatar_url} size="sm" />
              <span className="flex-1 truncate">{ws.name}</span>
              {unreadIds.has(ws.id) && <span className="size-2 rounded-full bg-brand" />}
            </DropdownMenuItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/**
 * Workspace stack at the top of the rail: one tile per workspace (the
 * current one marked), then pending invitations as dashed tiles. Creating a
 * workspace is rare enough to live in the account menu.
 * A dot on a tile means that workspace has unread inbox items; the current
 * workspace's unread shows on the Inbox space instead.
 */
export function RailWorkspaces() {
  const workspace = useCurrentWorkspace();
  const wsId = workspace?.id;
  const { data: workspaces = EMPTY_WORKSPACES } = useQuery(workspaceListOptions());
  const { data: invitations = EMPTY_INVITATIONS } = useQuery(myInvitationListOptions());
  // One cross-workspace cache entry; gated on an active workspace since the
  // endpoint resolves through the workspace-member middleware.
  const { data: unreadSummary = EMPTY_INBOX_SUMMARY } = useQuery({
    ...inboxUnreadSummaryOptions(),
    enabled: inboxUnreadSummaryEnabled(wsId),
  });
  const unreadIds = useMemo(() => unreadWorkspaceIds(unreadSummary), [unreadSummary]);
  const actions = useInvitationActions(invitations);

  const { tiles, overflow } = useMemo(() => {
    if (workspaces.length <= MAX_RAIL_WORKSPACES) {
      return { tiles: workspaces, overflow: EMPTY_WORKSPACES };
    }
    // Keep list order for muscle memory, but never hide the current one.
    let shown = workspaces.slice(0, MAX_RAIL_WORKSPACES - 1);
    const current = workspaces.find((ws) => ws.id === wsId);
    if (current && !shown.includes(current)) {
      shown = [...shown.slice(0, MAX_RAIL_WORKSPACES - 2), current];
    }
    return { tiles: shown, overflow: workspaces.filter((ws) => !shown.includes(ws)) };
  }, [workspaces, wsId]);

  return (
    <div className="flex flex-col items-center gap-2">
      {tiles.map((ws) => (
        <WorkspaceTile
          key={ws.id}
          workspace={ws}
          current={ws.id === wsId}
          unread={ws.id !== wsId && unreadIds.has(ws.id)}
        />
      ))}
      {overflow.length > 0 && <OverflowWorkspaces workspaces={overflow} unreadIds={unreadIds} />}
      {invitations.map((invitation) => (
        <InvitationTile key={invitation.id} invitation={invitation} actions={actions} />
      ))}
    </div>
  );
}
