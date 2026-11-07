import { useTranslation } from "react-i18next";
import { ApiError, messageFromUnknown } from "@/api/errors";
import { Button } from "@/components/ui/button";
import { useGeneration } from "@/hooks/use-generation";
import { useTracking } from "@/hooks/use-tracking";

/** Its message already says what to do instead, and trying again would not help. */
function isUnavailable(error: unknown): boolean {
  return error instanceof ApiError && error.code === "first_look_unavailable";
}

/** The state of the brand's one draft. It has no role once tracking has started. */
export function GenerationControls({
  brandID,
  setupStartFailed = false,
}: {
  brandID: string;
  setupStartFailed?: boolean;
}) {
  const { t } = useTranslation("management");
  const tracking = useTracking(brandID);
  const { capabilities, runs, run, create } = useGeneration(brandID);

  if (tracking.data?.confirmed !== false) return null;

  const enabled = capabilities.data?.aiDrafting === true;
  const failed = run?.state === "failed";
  const canRequest = enabled && runs.isSuccess && (failed || !run);

  function statusCopy() {
    if (create.isPending) return t("draftStarting");
    if (run?.state === "queued") return t("draftQueued");
    if (run?.state === "running") return t("draftRunning");
    if (failed) return t("draftFailed");
    if (run?.state === "succeeded" && run.promptCount + run.competitorCount === 0) {
      return t("draftEmpty");
    }

    return "";
  }

  const status = statusCopy();
  const setupFailed = setupStartFailed && enabled && !run && runs.isSuccess && !create.isError;

  if (!status && !runs.isError && !capabilities.isError && !create.isError && !canRequest) {
    return null;
  }

  return (
    <div className="space-y-3 text-sm">
      {status ? <output className="block">{status}</output> : null}
      {runs.isError ? (
        <p role="alert" className="text-destructive">
          {t("draftHistoryError")}{" "}
          <Button type="button" variant="link" size="sm" onClick={() => void runs.refetch()}>
            {t("retry")}
          </Button>
        </p>
      ) : null}
      {capabilities.isError ? (
        <p role="alert" className="text-destructive">
          {t("draftCapabilityError")}{" "}
          <Button
            type="button"
            variant="link"
            size="sm"
            onClick={() => void capabilities.refetch()}
          >
            {t("retry")}
          </Button>
        </p>
      ) : null}
      {create.isError ? (
        <p role="alert" className="text-destructive">
          {messageFromUnknown(create.error)}{" "}
          {isUnavailable(create.error) ? null : t("draftRequestRecovery")}
        </p>
      ) : null}
      {setupFailed ? (
        <p role="alert" className="text-destructive">
          {t("draftStartError")}
        </p>
      ) : null}
      {canRequest ? (
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={create.isPending}
          onClick={() => create.mutate()}
        >
          {failed || setupStartFailed ? t("retryDraft") : t("draftInitial")}
        </Button>
      ) : null}
    </div>
  );
}
