import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import { messageFromUnknown } from "@/api/errors";
import { Button } from "@/components/ui/button";
import { useTracking } from "@/hooks/use-tracking";
import { competitorKey, promptKey, runKey, trackingKey } from "@/lib/query-keys";

export function TrackingReview({
  brandID,
  kind,
  disabled,
}: {
  brandID: string;
  kind: "prompts" | "competitors";
  disabled: boolean;
}) {
  const api = useApi();
  const queryClient = useQueryClient();
  const tracking = useTracking(brandID);
  const { t } = useTranslation("visibility");

  const finish = useMutation({
    mutationFn: () => api[kind].confirmAll(brandID),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: trackingKey(brandID) }),
        queryClient.invalidateQueries({ queryKey: runKey(brandID) }),
        queryClient.invalidateQueries({
          queryKey: kind === "prompts" ? promptKey(brandID) : competitorKey(brandID),
        }),
      ]),
  });

  const reviewed =
    kind === "prompts" ? tracking.data?.promptsReviewed : tracking.data?.competitorsReviewed;

  if (reviewed && !finish.isError && !tracking.isError) return null;

  return (
    <div className="max-w-sm space-y-2 text-sm">
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={disabled || finish.isPending || tracking.isPending || reviewed}
        onClick={() => finish.mutate()}
      >
        {t(reviewed ? "reviewFinished" : finish.isPending ? "finishingReview" : "finishReview")}
      </Button>
      <p className="text-xs text-muted-foreground">{t("reviewNotice")}</p>
      {kind === "competitors" ? (
        <p className="text-xs text-muted-foreground">{t("observedReviewNotice")}</p>
      ) : null}
      {finish.isError ? (
        <p role="alert" className="text-destructive">
          {messageFromUnknown(finish.error)}
        </p>
      ) : null}
      {tracking.isError ? (
        <p role="alert" className="text-destructive">
          {t("trackingError")}
        </p>
      ) : null}
    </div>
  );
}

export function TrackingSummary({ brandID }: { brandID: string }) {
  const tracking = useTracking(brandID);
  const { t, i18n } = useTranslation("visibility");

  if (tracking.isError) {
    return (
      <p role="alert" className="text-sm text-destructive">
        {t("trackingError")}
      </p>
    );
  }
  if (!tracking.data) {
    return <p className="text-sm text-muted-foreground">{t("readinessLoading")}</p>;
  }

  const state = tracking.data;

  return (
    <div className="space-y-2 text-sm">
      <p className="text-muted-foreground">{t("cadence")}</p>
      {state.nextRunAt ? (
        <p>{t("nextRunAt", { date: state.nextRunAt.toLocaleString(i18n.language) })}</p>
      ) : null}
      {state.blockedReason ? <p>{t(`blocked.${state.blockedReason}`)}</p> : null}
      <div className="flex flex-wrap gap-4">
        {!state.promptsReviewed ? (
          <Link to="/" hash="prompts" className="underline underline-offset-4">
            {t("reviewPrompts")}
          </Link>
        ) : null}
        {!state.competitorsReviewed ? (
          <Link to="/competitors" className="underline underline-offset-4">
            {t("reviewCompetitors")}
          </Link>
        ) : null}
      </div>
    </div>
  );
}
