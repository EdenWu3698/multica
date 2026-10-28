"use client";

import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import type { Issue, UpdateIssueRequest } from "@multica/core/types";
import { useBatchUpdateIssues } from "@multica/core/issues/mutations";
import { useModalStore } from "@multica/core/modals";
import { useIssueActions } from "../issues/actions/use-issue-actions";
import { useT } from "../i18n";
import type { IssueTarget } from "./types";

export function targetIssues(target: IssueTarget): Issue[] {
  return target.kind === "single" ? [target.issue] : target.issues;
}

/**
 * Field writes for an issue command, routed the way the matching on-page
 * control routes them: a single issue goes through `useIssueActions`, whose
 * run-confirm gate asks before handing work to an agent; a selection goes
 * through the batch toolbar's rules.
 */
export function useIssueTargetWriter(target: IssueTarget) {
  const { t } = useT("issues");
  const single = target.kind === "single" ? target.issue : null;
  const actions = useIssueActions(single);
  const batchUpdate = useBatchUpdateIssues();
  const openModal = useModalStore((s) => s.open);
  const issues = useMemo(() => targetIssues(target), [target]);

  const applyBatch = useCallback(
    async (updates: Partial<UpdateIssueRequest>) => {
      if (target.kind !== "batch") return;
      const ids = issues.map((issue) => issue.id);
      const count = ids.length;
      try {
        const { updated } = await batchUpdate.mutateAsync({ ids, updates });
        if (updated < count) {
          toast.warning(t(($) => $.batch.update_partial, { updated, count }));
        } else {
          toast.success(t(($) => $.batch.update_success, { count }));
        }
      } catch (err) {
        toast.error(
          err instanceof Error && err.message ? err.message : t(($) => $.batch.update_failed),
        );
      }
    },
    [batchUpdate, issues, t, target.kind],
  );

  const apply = useCallback(
    (updates: Partial<UpdateIssueRequest>) => {
      if (target.kind === "single") {
        actions.updateField(updates);
        return;
      }
      void applyBatch(updates);
    },
    [actions, applyBatch, target.kind],
  );

  const assign = useCallback(
    (assigneeType: "member" | "agent" | "squad" | null, assigneeId: string | null) => {
      const updates: Partial<UpdateIssueRequest> = {
        assignee_type: assigneeType,
        assignee_id: assigneeId,
      };
      if (target.kind === "batch" && (assigneeType === "agent" || assigneeType === "squad") && assigneeId) {
        // Same rule as the batch toolbar: assigning agent work across a
        // selection previews the run fan-out, unless everything is parked in
        // Backlog and nothing would start.
        const allBacklog = issues.every((issue) => issue.status === "backlog");
        if (!allBacklog) {
          openModal("issue-run-confirm", {
            issueIds: issues.map((issue) => issue.id),
            mode: "assign",
            assigneeType,
            assigneeId,
          });
          return;
        }
      }
      apply(updates);
    },
    [apply, issues, openModal, target.kind],
  );

  return { issues, apply, assign, actions };
}
