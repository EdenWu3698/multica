"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useStore } from "zustand";
import { Bot, CircleUser, Layers, ListTodo, Plus, SquarePen } from "lucide-react";
import { IconButton } from "@multica/ui/components/common/icon-button";
import { useAuthStore } from "@multica/core/auth";
import { useCurrentWorkspace, useWorkspacePaths } from "@multica/core/paths";
import { myIssuesViewStore } from "@multica/core/issues/stores";
import { useIssueDraftStore } from "@multica/core/issues/stores/draft-store";
import { openCreateIssueWithPreference } from "@multica/core/issues/stores/create-mode-store";
import { issueViewListOptions } from "@multica/core/issue-views/queries";
import {
  issueViewContainerKey,
  useActiveIssueViewStore,
} from "@multica/core/issue-views/active-view-store";
import { projectListOptions } from "@multica/core/projects/queries";
import { useModalStore } from "@multica/core/modals";
import { useShortcut } from "@multica/core/shortcuts";
import type { IssueView } from "@multica/core/api/schemas";
import type { Project } from "@multica/core/types";
import { useNavigation } from "../../navigation";
import { ProjectIcon } from "../../projects/components/project-icon";
import { ShortcutKeycaps } from "../../common/shortcut-keycaps";
import { useT } from "../../i18n";
import {
  PanelBody,
  PanelCount,
  PanelHeader,
  PanelLink,
  PanelList,
  PanelSection,
  isPathWithin,
} from "../panel-nav";
import { PinnedSection, usePanelPins } from "./pinned-section";

const EMPTY_PROJECTS: Project[] = [];
const EMPTY_VIEWS: IssueView[] = [];
const PROJECT_PREVIEW_LIMIT = 6;
const OPEN_PROJECT_STATUSES = new Set<Project["status"]>(["planned", "in_progress", "paused"]);
const WORKSPACE_VIEW_SCOPE = { scope_type: "workspace" } as const;

function DraftDot() {
  const hasDraft = useIssueDraftStore((s) => s.hasDraft());
  if (!hasDraft) return null;
  return <span className="absolute -top-0.5 -right-0.5 size-1.5 rounded-full bg-brand" />;
}

export function WorkPanel() {
  const { t } = useT("layout");
  const p = useWorkspacePaths();
  const { pathname } = useNavigation();
  const wsId = useCurrentWorkspace()?.id;
  const userId = useAuthStore((s) => s.user?.id);
  const createIssueShortcut = useShortcut("createIssue");
  const myIssuesScope = useStore(myIssuesViewStore, (s) => s.scope);
  const setMyIssuesScope = useStore(myIssuesViewStore, (s) => s.setScope);
  const pins = usePanelPins(wsId, userId, pathname);

  const { data: projects = EMPTY_PROJECTS } = useQuery({
    ...projectListOptions(wsId ?? ""),
    enabled: !!wsId,
  });
  const openProjects = useMemo(
    () => projects.filter((project) => OPEN_PROJECT_STATUSES.has(project.status)),
    [projects],
  );
  // Older backends without the views endpoint error here; the section hides.
  const { data: views = EMPTY_VIEWS } = useQuery({
    ...issueViewListOptions(wsId ?? "", WORKSPACE_VIEW_SCOPE),
    enabled: !!wsId,
  });
  const viewContainerKey = issueViewContainerKey(wsId ?? "", WORKSPACE_VIEW_SCOPE);
  const activeViewId = useActiveIssueViewStore((s) => s.active[viewContainerKey] ?? null);
  const setActiveView = useActiveIssueViewStore((s) => s.setActive);

  const onMyIssues = pathname === p.myIssues();
  const onIssues = pathname === p.issues();
  const plainRowsLit = !pins.isActivePinnedRoute;

  return (
    <>
      <PanelHeader title={t(($) => $.spaces.work)}>
        <IconButton
          label={t(($) => $.sidebar.new_issue)}
          shortcut={
            createIssueShortcut ? <ShortcutKeycaps shortcut={createIssueShortcut} decorative /> : undefined
          }
          tooltipSide="bottom"
          className="text-muted-foreground"
          onClick={() => openCreateIssueWithPreference()}
        >
          <span className="relative">
            <SquarePen />
            <DraftDot />
          </span>
        </IconButton>
      </PanelHeader>
      <PanelBody>
        <PanelList>
          <PanelLink
            href={p.myIssues()}
            icon={<CircleUser />}
            label={t(($) => $.nav.my_issues)}
            active={plainRowsLit && onMyIssues && myIssuesScope !== "agents"}
            onNavigate={() => {
              if (myIssuesScope === "agents") setMyIssuesScope("assigned");
            }}
          />
          <PanelLink
            href={p.myIssues()}
            icon={<Bot />}
            label={t(($) => $.panels.work.my_agents)}
            active={plainRowsLit && onMyIssues && myIssuesScope === "agents"}
            onNavigate={() => setMyIssuesScope("agents")}
          />
          <PanelLink
            href={p.issues()}
            icon={<ListTodo />}
            label={t(($) => $.panels.work.all_issues)}
            active={
              plainRowsLit &&
              (onIssues ? activeViewId === null : isPathWithin(pathname, p.issues()))
            }
            onNavigate={() => setActiveView(viewContainerKey, null)}
          />
        </PanelList>

        {wsId && <PinnedSection pins={pins} wsId={wsId} pathname={pathname} />}

        {views.length > 0 && (
          <PanelSection label={t(($) => $.panels.work.views)}>
            <PanelList>
              {views.map((view) => (
                <PanelLink
                  key={view.id}
                  href={`${p.issues()}?view=${view.id}`}
                  icon={<Layers className="size-3.5" />}
                  label={view.name}
                  active={onIssues && activeViewId === view.id}
                  onNavigate={() => setActiveView(viewContainerKey, view.id)}
                />
              ))}
            </PanelList>
          </PanelSection>
        )}

        <PanelSection
          label={t(($) => $.nav.projects)}
          action={
            <IconButton
              label={t(($) => $.panels.work.new_project)}
              size="icon-xs"
              className="text-muted-foreground"
              onClick={() => useModalStore.getState().open("create-project")}
            >
              <Plus />
            </IconButton>
          }
        >
          <PanelList>
            {openProjects.slice(0, PROJECT_PREVIEW_LIMIT).map((project) => {
              const href = p.projectDetail(project.id);
              return (
                <PanelLink
                  key={project.id}
                  href={href}
                  icon={<ProjectIcon project={project} size="sm" />}
                  label={project.title}
                  active={plainRowsLit && isPathWithin(pathname, href)}
                  trailing={<PanelCount value={project.issue_count - project.done_count} />}
                />
              );
            })}
            <PanelLink
              href={p.projects()}
              icon={<span className="size-4" aria-hidden />}
              label={t(($) => $.panels.work.all_projects, { count: projects.length })}
              active={pathname === p.projects()}
              className="text-caption"
            />
          </PanelList>
        </PanelSection>
      </PanelBody>
    </>
  );
}
