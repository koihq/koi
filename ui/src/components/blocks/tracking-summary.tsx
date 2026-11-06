import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { useTracking } from "@/hooks/use-tracking";
import { formatDateTime } from "@/lib/format";

export function TrackingSummary({
  brandID,
  showSchedule = true,
}: {
  brandID: string;
  showSchedule?: boolean;
}) {
  const tracking = useTracking(brandID);
  const { t, i18n } = useTranslation("visibility");

  if (tracking.isError) {
    return (
      <p role="alert" className="text-sm text-destructive">
        {t("trackingError")}{" "}
        <Button variant="link" onClick={() => void tracking.refetch()}>
          {t("retry")}
        </Button>
      </p>
    );
  }
  if (!tracking.data) {
    if (!showSchedule) return null;
    return <p className="text-sm text-muted-foreground">{t("readinessLoading")}</p>;
  }

  const state = tracking.data;
  if (!showSchedule && !state.blockedReason) return null;

  return (
    <div className="space-y-2 text-sm">
      {showSchedule && <p className="text-muted-foreground">{t("cadence")}</p>}
      {showSchedule && state.nextRunAt ? (
        <p>{t("nextRunAt", { date: formatDateTime(state.nextRunAt, i18n.language) })}</p>
      ) : null}
      {state.blockedReason ? <p>{t(`blocked.${state.blockedReason}`)}</p> : null}
      <div className="flex flex-wrap gap-4">
        {state.blockedReason === "billing_inactive_or_over_limit" ||
        state.blockedReason === "first_look_used" ||
        state.blockedReason === "first_look_site_used" ||
        state.blockedReason === "first_look_paused" ? (
          <Link to="/settings/billing" className="underline underline-offset-4">
            {t("reviewBilling")}
          </Link>
        ) : null}
      </div>
    </div>
  );
}
