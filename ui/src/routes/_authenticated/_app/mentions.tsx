import { createFileRoute, Link } from "@tanstack/react-router";
import { createStandardSchemaV1, parseAsInteger, parseAsString } from "nuqs";
import { useTranslation } from "react-i18next";

import { messageFromUnknown } from "@/api/errors";
import { Button } from "@/components/ui/button";
import { FieldLabel } from "@/components/ui/field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { VisibilityMatrix } from "@/routes/_authenticated/_app/mentions/-components/visibility-matrix";
import { useVisibility } from "@/routes/_authenticated/_app/mentions/-hooks/use-visibility";

export const Route = createFileRoute("/_authenticated/_app/mentions")({
  validateSearch: createStandardSchemaV1(
    { run: parseAsString, page: parseAsInteger.withDefault(1) },
    { partialOutput: true },
  ),
  component: Mentions,
});

function Mentions() {
  const { brand } = Route.useRouteContext();
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const { t, i18n } = useTranslation("visibility");
  const page = Math.max(1, search.page ?? 1);
  const selectRun = (runID: string) => {
    void navigate({ search: { run: runID, page: 1 } });
  };
  const state = useVisibility(brand.id, search.run ?? undefined, page, selectRun);
  const run = state.run;
  const history = [...(state.runs.data?.items ?? [])];
  if (run && !history.some((item) => item.id === run.id)) history.unshift(run);
  const options = history.map((item) => ({
    value: item.id,
    label: item.createdAt.toLocaleString(i18n.language),
  }));
  const pageLoading = state.results.isPending || state.results.isPlaceholderData;

  return (
    <section aria-labelledby="visibility-heading" className="mx-auto min-w-0 max-w-6xl space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0 space-y-2">
          <h2 id="visibility-heading" className="text-2xl font-medium tracking-tight">
            {t("title")}
          </h2>
          <p className="max-w-prose text-sm text-muted-foreground">{t("sampleNotice")}</p>
        </div>
        <Button type="button" disabled={!state.canStart} onClick={() => state.create.mutate()}>
          {t(state.create.isPending ? "starting" : "start")}
        </Button>
      </div>

      <div className="space-y-3 text-sm">
        {state.readinessError ? (
          <p role="alert" className="text-destructive">
            {t("readinessError")}{" "}
            <Button
              type="button"
              variant="link"
              size="sm"
              onClick={() => void state.retryReadiness()}
            >
              {t("retry")}
            </Button>
          </p>
        ) : !state.readinessLoaded ? (
          <p className="text-muted-foreground">{t("readinessLoading")}</p>
        ) : !state.readiness.ready ? (
          <div className="space-y-2">
            <p>{t(state.readiness.needsPrompts ? "needPrompts" : "needsConfirmation")}</p>
            <div className="flex flex-wrap gap-4">
              {state.readiness.needsPrompts || state.readiness.needsPromptReview ? (
                <Link to="/prompts" className="underline underline-offset-4">
                  {t("reviewPrompts")}
                </Link>
              ) : null}
              {state.readiness.needsCompetitorReview ? (
                <Link to="/competitors" className="underline underline-offset-4">
                  {t("reviewCompetitors")}
                </Link>
              ) : null}
            </div>
          </div>
        ) : null}
        {state.create.isError ? (
          <p role="alert" className="text-destructive">
            {messageFromUnknown(state.create.error)} {t("requestErrorRecovery")}
          </p>
        ) : null}
        {state.runs.isError ? (
          <p role="alert" className="text-destructive">
            {t("historyError")}{" "}
            <Button
              type="button"
              variant="link"
              size="sm"
              onClick={() => void state.runs.refetch()}
            >
              {t("retry")}
            </Button>
          </p>
        ) : null}
      </div>

      {options.length ? (
        <div className="flex flex-wrap items-center gap-3">
          <FieldLabel htmlFor="visibility-run">{t("history")}</FieldLabel>
          <Select
            value={state.runID ?? null}
            items={options}
            onValueChange={(value) => {
              if (value) selectRun(value);
            }}
          >
            <SelectTrigger id="visibility-run" aria-label={t("history")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {options.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      ) : null}

      <output className="block text-sm tabular-nums" aria-live="polite">
        {run ? t(run.state, { completed: run.completedResults, total: run.totalResults }) : null}
      </output>
      {state.status.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {t("statusError")}{" "}
          <Button
            type="button"
            variant="link"
            size="sm"
            onClick={() => void state.status.refetch()}
          >
            {t("retry")}
          </Button>
        </p>
      ) : (state.runID && !run) || state.runs.isPending ? (
        <p className="text-sm text-muted-foreground">{t("loading")}</p>
      ) : !state.runID && state.runs.isSuccess ? (
        <p className="py-10 text-sm text-muted-foreground">{t("empty")}</p>
      ) : null}

      {run ? (
        <div className="min-w-0 space-y-4" aria-busy={state.results.isFetching}>
          {state.results.isError ? (
            <p role="alert" className="text-sm text-destructive">
              {t("resultsError")}{" "}
              <Button
                type="button"
                variant="link"
                size="sm"
                onClick={() => void state.results.refetch()}
              >
                {t("retry")}
              </Button>
            </p>
          ) : state.results.isPending ? (
            <p className="text-sm text-muted-foreground">{t("loading")}</p>
          ) : state.results.data?.items.length ? (
            <VisibilityMatrix key={run.id} run={run} results={state.results.data.items} />
          ) : (
            <p className="text-sm text-muted-foreground">{t("noResults")}</p>
          )}
          <div className="flex flex-wrap items-center justify-between gap-3 text-xs text-muted-foreground">
            <p className="tabular-nums">
              {t("page", {
                current: page,
                total: Math.max(1, state.results.data?.totalPages ?? 0),
              })}
            </p>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={pageLoading || page <= 1}
                onClick={() => void navigate({ search: { run: run.id, page: page - 1 } })}
              >
                {t("previous")}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={pageLoading || state.results.isError || !state.results.data?.hasNext}
                onClick={() => void navigate({ search: { run: run.id, page: page + 1 } })}
              >
                {t("next")}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}
