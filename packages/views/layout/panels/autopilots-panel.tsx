"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Code, Clock, Webhook, Zap } from "lucide-react";
import { autopilotListOptions } from "@multica/core/autopilots/queries";
import { useCurrentWorkspace, useWorkspacePaths } from "@multica/core/paths";
import type { Autopilot } from "@multica/core/types";
import { useNavigation } from "../../navigation";
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

const EMPTY_AUTOPILOTS: Autopilot[] = [];

// Server-driven enum: unknown kinds get the generic icon.
const TRIGGER_ICONS: Record<string, typeof Zap> = {
  schedule: Clock,
  webhook: Webhook,
  api: Code,
};

/** Last run failed on an autopilot that is still meant to run. */
export function isAutopilotFailing(autopilot: Autopilot): boolean {
  return autopilot.status === "active" && autopilot.last_run_status === "failed";
}

/** Workspace autopilots that need attention, for the rail badge and the panel. */
export function useFailingAutopilots(wsId: string | undefined) {
  const { data: autopilots = EMPTY_AUTOPILOTS } = useQuery({
    ...autopilotListOptions(wsId ?? ""),
    enabled: !!wsId,
  });
  return useMemo(() => autopilots.filter(isAutopilotFailing), [autopilots]);
}

function hasDescription(autopilot: Autopilot): boolean {
  return (
    isAutopilotFailing(autopilot) ||
    autopilot.status === "paused" ||
    (autopilot.trigger_kinds?.length ?? 0) > 0
  );
}

function AutopilotDescription({ autopilot }: { autopilot: Autopilot }) {
  const { t } = useT("autopilots");
  if (isAutopilotFailing(autopilot)) {
    return <span className="text-destructive">{t(($) => $.run_status.failed)}</span>;
  }
  if (autopilot.status === "paused") return <>{t(($) => $.status.paused)}</>;
  return (
    <>
      {(autopilot.trigger_kinds ?? [])
        .map((kind) =>
          kind === "schedule" || kind === "webhook" || kind === "api"
            ? t(($) => $.trigger_kind[kind])
            : kind,
        )
        .join(" · ")}
    </>
  );
}

export function AutopilotsPanel() {
  const { t } = useT("layout");
  const p = useWorkspacePaths();
  const { pathname } = useNavigation();
  const wsId = useCurrentWorkspace()?.id;
  const { data: autopilots = EMPTY_AUTOPILOTS } = useQuery({
    ...autopilotListOptions(wsId ?? ""),
    enabled: !!wsId,
  });
  const visible = useMemo(
    () => autopilots.filter((autopilot) => autopilot.status !== "archived"),
    [autopilots],
  );

  return (
    <>
      <PanelHeader title={t(($) => $.nav.autopilots)} />
      <PanelBody>
        <PanelList>
          <PanelLink
            href={p.autopilots()}
            icon={<Zap />}
            label={t(($) => $.panels.autopilots.all)}
            active={pathname === p.autopilots()}
            trailing={<PanelCount value={visible.length} />}
          />
        </PanelList>

        {visible.length > 0 && (
          <PanelSection label={t(($) => $.panels.autopilots.list)}>
            <PanelList>
              {visible.map((autopilot) => {
                const href = p.autopilotDetail(autopilot.id);
                const Icon = TRIGGER_ICONS[autopilot.trigger_kinds?.[0] ?? ""] ?? Zap;
                return (
                  <PanelLink
                    key={autopilot.id}
                    href={href}
                    icon={<Icon />}
                    label={autopilot.title}
                    description={
                      hasDescription(autopilot) ? <AutopilotDescription autopilot={autopilot} /> : undefined
                    }
                    active={isPathWithin(pathname, href)}
                  />
                );
              })}
            </PanelList>
          </PanelSection>
        )}
      </PanelBody>
    </>
  );
}
