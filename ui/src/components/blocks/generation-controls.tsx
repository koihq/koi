import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { messageFromUnknown } from "@/api/errors";
import type { GenerationMode } from "@/api/generations.api";
import { Button } from "@/components/ui/button";
import { useGeneration } from "@/hooks/use-generation";

export function GenerationControls({
  brandID,
  mode,
  setupStartFailed = false,
}: {
  brandID: string;
  mode: "prompts" | "competitors";
  setupStartFailed?: boolean;
}) {
  const { t } = useTranslation("management");
  const { capabilities, runs, run, create } = useGeneration(brandID);
  const enabled = capabilities.data?.aiDrafting === true;
  const busy = run?.state === "queued" || run?.state === "running" || create.isPending;

  const showStatus =
    !create.isPending && run && (busy || run.mode === "initial" || run.mode === mode);

  const retryMode: GenerationMode | undefined =
    run?.state === "failed" && (run.mode === "initial" || run.mode === mode) ? run.mode : undefined;

  const showMore = run && !retryMode && (run.mode !== "initial" || run.state === "succeeded");
  function statusCopy() {
    if (!run) return "";

    const subject = t(`draftSubject.${run.mode}`);
    if (run.state === "queued") return t("draftQueued", { subject });

    if (run.state === "running") return t("draftRunning", { subject });

    if (run.state === "failed") return t("draftFailed", { subject });

    if (run.mode === "initial") {
      return run.promptCount + run.competitorCount === 0
        ? t("draftEmpty")
        : t("draftInitialDone", {
            promptCount: run.promptCount,
            competitorCount: run.competitorCount,
          });
    }

    const count = run.mode === "prompts" ? run.promptCount : run.competitorCount;

    return count === 0 ? t("draftEmpty") : t("draftListDone", { count, subject });
  }

  return (
    <div className="mb-6 space-y-3 text-sm">
      <output className="block space-y-2">
        {create.isPending ? (
          <span className="block">
            {t("draftStarting", { subject: t(`draftSubject.${create.variables ?? mode}`) })}
          </span>
        ) : null}
        {showStatus ? (
          <>
            <span className="block">{statusCopy()}</span>
            {run.mode === "initial" && run.state === "succeeded" && mode === "prompts" ? (
              <Link to="/competitors" className="text-sm underline underline-offset-4">
                {t("reviewCompetitors")}
              </Link>
            ) : null}
          </>
        ) : null}
      </output>
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
          {messageFromUnknown(create.error)} {t("draftRequestRecovery")}
        </p>
      ) : null}
      {setupStartFailed && enabled && !run && runs.isSuccess && !create.isError ? (
        <p role="alert" className="text-destructive">
          {t("draftStartError")}
        </p>
      ) : null}
      {enabled && runs.isSuccess ? (
        <div className="flex flex-wrap items-center gap-3">
          {retryMode ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={busy}
              onClick={() => create.mutate(retryMode)}
            >
              {t("retryDraft")}
            </Button>
          ) : null}
          {!run && mode === "prompts" ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={busy}
              onClick={() => create.mutate("initial")}
            >
              {setupStartFailed ? t("retryDraft") : t("draftInitial")}
            </Button>
          ) : null}
          {!run && mode === "competitors" ? (
            <Link to="/prompts" className="underline underline-offset-4">
              {t("startDraftOnPrompts")}
            </Link>
          ) : null}
          {showMore ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={busy}
              onClick={() => create.mutate(mode)}
            >
              {mode === "prompts" ? t("draftMorePrompts") : t("draftMoreCompetitors")}
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
