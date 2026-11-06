import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import { messageFromUnknown } from "@/api/errors";
import { GenerationControls } from "@/components/blocks/generation-controls";
import { Button } from "@/components/ui/button";
import { useCompetitorCounts } from "@/hooks/use-competitor-counts";
import { useGeneration } from "@/hooks/use-generation";
import { useTracking } from "@/hooks/use-tracking";
import { invalidateProduct } from "@/lib/queries";
import type { VisibilityScope } from "@/lib/visibility";
import { trackingStartView } from "./tracking-start-view";
import { usePromptCounts } from "./use-prompt-counts";

/** The brand's one confirmation: it starts daily checks and does not come back. */
export function TrackingStart({
  brandID,
  scope,
  draftStartFailed,
  description,
  actionLabel,
  showCompetitorLink = true,
}: {
  brandID: string;
  scope: VisibilityScope;
  draftStartFailed: boolean;
  description?: string;
  actionLabel?: string;
  showCompetitorLink?: boolean;
}) {
  const { t } = useTranslation("visibility");
  const api = useApi();
  const queryClient = useQueryClient();
  const tracking = useTracking(brandID);
  const { run, create } = useGeneration(brandID);
  const promptCounts = usePromptCounts(brandID, scope);
  const competitorCounts = useCompetitorCounts(brandID);

  const start = useMutation({
    mutationFn: () => api.runs.startTracking(brandID),
    onSuccess: () => invalidateProduct(queryClient, brandID, "competitors"),
  });

  const prompts = (promptCounts.tracking ?? 0) + (promptCounts.drafts ?? 0);
  const competitors = competitorCounts.all ?? 0;
  const view = trackingStartView({
    collectionAvailable: tracking.data?.blockedReason !== "collection_unavailable",
    drafting: create.isPending || run?.state === "queued" || run?.state === "running",
    prompts,
  });

  const promptLabel = t("start.promptCount", { count: prompts });
  const competitorLabel = t("start.competitorCount", { count: competitors });

  return (
    <div className="grid min-h-60 content-center justify-items-start gap-4">
      {view === "ready" ? (
        <div className="grid gap-1">
          <p className="text-body">
            {competitors > 0
              ? t("start.ready", { prompts: promptLabel, competitors: competitorLabel })
              : t("start.readyPromptsOnly", { prompts: promptLabel })}
          </p>
          <p className="max-w-[48ch] text-body text-muted-foreground">
            {description ?? t("start.description")}
          </p>
        </div>
      ) : null}
      {view === "empty" ? <p className="text-body">{t("start.noPrompts")}</p> : null}
      {view === "unavailable" ? (
        <p className="max-w-[48ch] text-body">{t("blocked.collection_unavailable")}</p>
      ) : null}
      <GenerationControls brandID={brandID} setupStartFailed={draftStartFailed} />
      {view === "unavailable" ? null : (
        <div className="flex flex-wrap items-center gap-3">
          <Button
            type="button"
            size="lg"
            disabled={view !== "ready" || start.isPending}
            onClick={() => start.mutate()}
          >
            {start.isPending ? t("start.starting") : (actionLabel ?? t("start.action"))}
          </Button>
          {showCompetitorLink && (
            <Button variant="link" render={<Link to="/competitors" />} nativeButton={false}>
              {t("reviewCompetitors")}
            </Button>
          )}
        </div>
      )}
      {start.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {messageFromUnknown(start.error)}
        </p>
      ) : null}
    </div>
  );
}
