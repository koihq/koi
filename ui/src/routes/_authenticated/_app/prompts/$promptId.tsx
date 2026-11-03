import { useCanAnimate } from "@number-flow/react";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
  createStandardSchemaV1,
  parseAsInteger,
  parseAsNumberLiteral,
  parseAsString,
  parseAsStringLiteral,
  useQueryStates,
} from "nuqs";
import { useCallback, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { VisibilityEngine } from "@/api/generated";
import { BrandLabel } from "@/components/blocks/brand-label";
import { OutcomeLegend } from "@/components/blocks/outcome-mark";
import { PromptDialog } from "@/components/blocks/prompt-dialog";
import { QueryRefreshIndicator } from "@/components/blocks/query-refresh-indicator";
import { type InspectionInput, VisibilityChart } from "@/components/blocks/visibility-chart";
import { VisibilityHeadline } from "@/components/blocks/visibility-headline";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCollection, useRefreshAfterCollection } from "@/hooks/use-collection";
import { useFlip } from "@/hooks/use-flip";
import { usePromptMutations } from "@/hooks/use-prompt-mutations";
import { isCollecting } from "@/lib/answer-state";
import { formatDateTime, formatDayRange, formatPoints, formatRate } from "@/lib/format";
import { keepPreviousDataForScope } from "@/lib/queries";
import { promptKey, runKey } from "@/lib/query-keys";
import {
  type VisibilityScope,
  rate as visibilityRate,
  series as visibilitySeries,
} from "@/lib/visibility";
import { CopyForAgent, EngineSwitch, RunAnswer } from "./-components/run-evidence";
import { RunHistory } from "./-components/run-history";

const detailSearchParams = {
  page: parseAsInteger.withDefault(1).withOptions({ clearOnDefault: true }),
  days: parseAsNumberLiteral([7, 30, 90] as const)
    .withDefault(30)
    .withOptions({ clearOnDefault: true }),
  compareId: parseAsString.withDefault("").withOptions({ clearOnDefault: true }),
  runId: parseAsString.withDefault("").withOptions({ clearOnDefault: true }),
  responseEngine: parseAsStringLiteral([
    "chatgpt",
    "perplexity",
    "gemini",
    "google_ai_overviews",
  ] as const).withDefault("chatgpt"),
};

export const Route = createFileRoute("/_authenticated/_app/prompts/$promptId")({
  validateSearch: createStandardSchemaV1(detailSearchParams, { partialOutput: true }),
  component: PromptDetail,
});

function PromptDetail() {
  const { t, i18n } = useTranslation(["home", "management", "visibility", "common"]);
  const { brand } = Route.useRouteContext();
  const { promptId } = Route.useParams();
  const api = useApi();
  const mutations = usePromptMutations(brand.id);
  const { collecting } = useCollection(brand.id);
  const [search, setSearch] = useQueryStates(detailSearchParams);
  const [inspected, setInspected] = useState<number | null>(null);
  const [inspectionInput, setInspectionInput] = useState<InspectionInput>("keyboard");
  const canAnimateNumbers = useCanAnimate();
  const [editing, setEditing] = useState(false);
  const runTrigger = useRef<HTMLButtonElement | null>(null);
  const editTrigger = useRef<HTMLButtonElement | null>(null);

  const prompt = useQuery({
    queryKey: [...promptKey(brand.id), promptId],
    queryFn: ({ signal }) => api.prompts.getById(promptId, { signal }),
  });
  const history = useQuery({
    retry: 1,
    queryKey: [...runKey(brand.id), "prompt", promptId, search.page],
    queryFn: ({ signal }) =>
      api.runs.promptHistory(brand.id, promptId, search.page, 10, { signal }),
    placeholderData: keepPreviousDataForScope([...runKey(brand.id), "prompt", promptId]),
    refetchInterval: (query) =>
      collecting || query.state.data?.items.some((item) => isCollecting(item.run)) ? 5_000 : false,
  });
  const historyCollecting = history.data?.items.some((item) => isCollecting(item.run)) ?? false;
  useRefreshAfterCollection(brand.id, history.data?.items.map((item) => item.run) ?? []);
  const scope: VisibilityScope = { days: search.days, engines: [] };
  const visibility = useQuery({
    retry: 1,
    queryKey: [...runKey(brand.id), "prompt-visibility", promptId, search.days],
    queryFn: ({ signal }) => api.runs.promptVisibility(brand.id, promptId, scope, { signal }),
    placeholderData: keepPreviousDataForScope([...runKey(brand.id), "prompt-visibility", promptId]),
    refetchInterval: collecting || historyCollecting ? 5_000 : false,
  });

  const runs = history.data?.items ?? [];
  const run = runs.find((item) => item.run.id === search.runId);
  const result = run?.results.find((item) => item.engine === search.responseEngine);
  const title = prompt.data?.text ?? runs[0]?.results[0]?.promptText;
  const comparison = visibility.data?.competitors.find((item) => item.id === search.compareId);
  const series = useMemo(() => visibilitySeries(visibility.data), [visibility.data]);
  const comparedSeries = useMemo(
    () => (comparison ? visibilitySeries(visibility.data, comparison.id) : []),
    [visibility.data, comparison],
  );
  const rate = visibilityRate(visibility.data);
  const comparisonRate = comparison
    ? visibilityRate(visibility.data, undefined, comparison.id)
    : null;
  const rivals = (visibility.data?.competitors ?? [])
    .map((competitor) => ({
      competitor,
      rate: competitor.metric.rate,
    }))
    .sort((left, right) => (right.rate ?? -1) - (left.rate ?? -1));
  const rivalList = useRef<HTMLDivElement>(null);
  useFlip(rivalList, rivals.map(({ competitor }) => competitor.id).join(","));
  const metricRate = inspected === null ? rate : (series[inspected]?.rate ?? null);
  const metricComparisonRate =
    inspected === null ? comparisonRate : (comparedSeries[inspected]?.rate ?? null);

  const inspect = useCallback((index: number | null, input: InspectionInput) => {
    setInspectionInput(input);
    setInspected(index);
  }, []);

  function openRun(id: string, trigger: HTMLButtonElement) {
    runTrigger.current = trigger;
    void setSearch({ runId: id, responseEngine: "chatgpt" }, { history: "push" });
  }

  function rivalDomain(name: string) {
    return visibility.data?.competitors.find((candidate) => candidate.name === name)?.domain;
  }

  return (
    <article className="grid min-w-0 grid-cols-1 gap-6 pb-24">
      <Link
        to="/"
        hash="prompts"
        className="text-xs font-normal text-muted-foreground hover:text-foreground"
      >
        {t("home:detail.allPrompts")}
      </Link>
      <div className="mb-6 flex items-start justify-between gap-6">
        <h1 className="max-w-[60ch] text-title font-semibold text-balance">
          {title ??
            t(
              history.isPending || prompt.isPending
                ? "home:detail.loadingPrompt"
                : "home:detail.notFound",
            )}
        </h1>
        {prompt.data && (
          <Button ref={editTrigger} variant="ghost" size="sm" onClick={() => setEditing(true)}>
            {t("management:editPrompt")}
          </Button>
        )}
      </div>
      {prompt.isError && title && (
        <p className="text-xs font-normal text-muted-foreground">{t("home:detail.historical")}</p>
      )}
      {prompt.data && !prompt.data.confirmed && (
        <p className="text-xs font-normal text-muted-foreground">{t("home:dashboard.draft")}</p>
      )}

      {visibility.isError && visibility.data && (
        <p role="alert" className="text-sm text-destructive">
          {t("home:dashboard.loadError")}{" "}
          <Button variant="link" onClick={() => void visibility.refetch()}>
            {t("home:dashboard.retry")}
          </Button>
        </p>
      )}
      <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_20rem]">
        <section
          className="group/refresh relative min-w-0 rounded-lg bg-surface p-4 sm:p-7 "
          aria-busy={visibility.isFetching}
          data-refreshing={visibility.isFetching && Boolean(visibility.data)}
          aria-labelledby="detail-visibility-heading"
          onPointerDownCapture={(event) =>
            setInspectionInput(event.pointerType === "touch" ? "touch" : "pointer")
          }
          onKeyDownCapture={() => setInspectionInput("keyboard")}
        >
          <QueryRefreshIndicator
            active={visibility.isFetching}
            label={t("home:dashboard.loading")}
          />
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 id="detail-visibility-heading" className="text-section font-semibold">
                {t("home:dashboard.aiVisibility")}
              </h2>
              {visibility.data && (
                <VisibilityHeadline
                  brand={brand}
                  competitor={comparison}
                  rate={metricRate}
                  comparisonRate={metricComparisonRate}
                  paired={Boolean(comparison && metricComparisonRate !== null)}
                  locale={i18n.language}
                  animated={canAnimateNumbers && inspectionInput === "pointer"}
                  variant="detail"
                />
              )}
            </div>
            <fieldset className="flex gap-0.5" aria-label={t("home:dashboard.range")}>
              {([7, 30, 90] as const).map((days) => (
                <Button
                  key={days}
                  type="button"
                  aria-pressed={search.days === days}
                  variant="ghost"
                  size="pill"
                  className="text-muted-foreground"
                  onClick={() => {
                    void setSearch({ days }, { history: "push" });
                    setInspected(null);
                  }}
                >
                  {`${days}D`}
                </Button>
              ))}
            </fieldset>
          </div>
          {visibility.isPending ? (
            <p className="py-8 text-sm text-muted-foreground">{t("home:dashboard.loading")}</p>
          ) : visibility.isError && !visibility.data ? (
            <p role="alert" className="py-8 text-sm">
              {t("home:dashboard.loadError")}{" "}
              <Button variant="link" onClick={() => void visibility.refetch()}>
                {t("home:dashboard.retry")}
              </Button>
            </p>
          ) : visibility.data?.metric.completed ? (
            <VisibilityChart
              series={series}
              comparison={comparedSeries}
              brand={brand}
              competitor={comparison}
              inspected={inspected}
              inspectionInput={inspectionInput}
              onInspect={inspect}
              locale={i18n.language}
              scopeKey={`${brand.id}:${promptId}:${search.days}:${search.compareId}`}
            />
          ) : (
            <p className="group-data-[refreshing=true]/refresh:opacity-50 grid min-h-[244px] content-center justify-items-start gap-3 text-body text-muted-foreground">
              {t("home:detail.noAnswers")}
            </p>
          )}
          {series.length > 0 && (
            <p className="mt-4 text-xs font-normal text-muted-foreground group-data-[refreshing=true]/refresh:opacity-50">
              {formatDayRange(series[0].date, series[series.length - 1].date, i18n.language)}
            </p>
          )}
        </section>
        <aside
          className="group/refresh min-w-0 rounded-lg bg-surface p-5 "
          aria-labelledby="detail-compare-heading"
          aria-busy={visibility.isFetching}
          data-refreshing={visibility.isFetching && Boolean(visibility.data)}
        >
          <div className="mb-4 flex items-baseline justify-between gap-3">
            <h2 id="detail-compare-heading" className="text-section font-semibold">
              {t("home:detail.compareWith")}
            </h2>
            <span className="text-xs text-muted-foreground">
              {t("home:detail.compareAverage", { days: search.days })}
            </span>
          </div>
          {rivals.length ? (
            <div ref={rivalList} className="list-rows grid gap-0.5">
              {rivals.map(({ competitor, rate: rivalRate }) => (
                <button
                  key={competitor.id}
                  data-flip-key={competitor.id}
                  type="button"
                  aria-pressed={comparison?.id === competitor.id}
                  className="grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 rounded-md p-2 text-start hover:bg-signal-pill aria-pressed:bg-signal-pill focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  onClick={() => {
                    void setSearch(
                      { compareId: comparison?.id === competitor.id ? "" : competitor.id },
                      { history: "push" },
                    );
                    setInspected(null);
                  }}
                >
                  <span className="justify-self-start">
                    <BrandLabel name={competitor.name} domain={competitor.domain} />
                  </span>
                  <span className="text-body tabular-nums group-data-[refreshing=true]/refresh:opacity-50">
                    {rivalRate === null ? "—" : formatRate(rivalRate, i18n.language)}
                  </span>
                  <small className="col-span-full text-xs text-muted-foreground tabular-nums group-data-[refreshing=true]/refresh:opacity-50">
                    {rivalRate !== null && rate !== null
                      ? t("home:detail.comparisonGap", {
                          points: formatPoints(rivalRate - rate, i18n.language),
                        })
                      : t("home:detail.noComparisonData")}
                  </small>
                </button>
              ))}
            </div>
          ) : visibility.isSuccess ? (
            <p className="text-xs font-normal text-muted-foreground">
              {t("home:dashboard.noCompetitors")}
            </p>
          ) : null}
        </aside>
      </div>

      <section
        className="group/refresh relative mt-8 min-w-0"
        aria-labelledby="run-history-heading"
        aria-busy={history.isFetching}
        data-refreshing={history.isFetching && Boolean(history.data)}
      >
        <QueryRefreshIndicator
          active={history.isFetching}
          label={t("home:detail.loadingAnswers")}
        />
        <div className="flex flex-wrap items-baseline justify-between gap-x-6 gap-y-2">
          <h2 id="run-history-heading" className="text-section font-semibold">
            {t("home:detail.runHistory")}
            <span className="ms-2 text-xs font-normal text-muted-foreground tabular-nums">
              {t("home:detail.runsCount", { count: history.data?.totalItems ?? 0 })}
            </span>
          </h2>
          <OutcomeLegend />
        </div>
        {history.isError && history.data && (
          <p role="alert" className="text-sm text-destructive">
            {t("home:detail.loadError")}{" "}
            <Button variant="link" onClick={() => void history.refetch()}>
              {t("home:detail.retry")}
            </Button>
          </p>
        )}
        {history.isError && !history.data ? (
          <p role="alert" className="py-8 text-sm text-destructive">
            {t("home:detail.loadError")}{" "}
            <Button variant="link" onClick={() => void history.refetch()}>
              {t("home:detail.retry")}
            </Button>
          </p>
        ) : history.isPending ? (
          <p className="py-8 text-sm text-muted-foreground">{t("home:detail.loadingAnswers")}</p>
        ) : runs.length === 0 ? (
          <p className="py-8 text-sm text-muted-foreground">{t("home:detail.noAnswers")}</p>
        ) : (
          <div className="group-data-[refreshing=true]/refresh:opacity-50">
            <RunHistory runs={runs} rivalDomain={rivalDomain} onOpen={openRun} />
          </div>
        )}
        {history.data && history.data.totalPages > 1 && (
          <div className="mt-6 flex items-center gap-3 text-xs text-muted-foreground">
            <span>
              {t("home:dashboard.page", { current: search.page, total: history.data.totalPages })}
            </span>
            <div className="ms-auto flex gap-2">
              <Button
                variant="ghost"
                size="sm"
                disabled={!history.data.hasPrev}
                onClick={() =>
                  void setSearch({ page: search.page - 1, runId: "" }, { history: "push" })
                }
              >
                {t("home:dashboard.previous")}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                disabled={!history.data.hasNext}
                onClick={() =>
                  void setSearch({ page: search.page + 1, runId: "" }, { history: "push" })
                }
              >
                {t("home:dashboard.next")}
              </Button>
            </div>
          </div>
        )}
      </section>

      <Dialog
        open={Boolean(search.runId)}
        onOpenChange={(open) => {
          if (!open) void setSearch({ runId: "" }, { history: "replace" });
        }}
      >
        <DialogContent
          className="max-h-[calc(100dvh-32px)] w-[calc(100vw-32px)] max-w-[1100px] overflow-y-auto sm:max-w-[1100px]"
          finalFocus={() => runTrigger.current}
        >
          <DialogHeader className="gap-4 pe-8 sm:flex-row sm:items-start sm:justify-between">
            <div className="grid gap-1">
              <DialogTitle className="text-section font-semibold text-balance">{title}</DialogTitle>
              <DialogDescription className="tabular-nums">
                {run ? formatDateTime(run.run.createdAt, i18n.language) : null}
              </DialogDescription>
            </div>
            {run && title && (
              <CopyForAgent brand={brand} prompt={title} run={run.run} result={result} />
            )}
          </DialogHeader>
          {run ? (
            <div className="grid gap-8">
              <EngineSwitch
                run={run.run}
                results={run.results}
                selected={search.responseEngine}
                onSelect={(engine: VisibilityEngine) =>
                  void setSearch({ responseEngine: engine }, { history: "replace" })
                }
              />
              <RunAnswer
                brand={brand}
                run={run.run}
                result={result}
                engine={search.responseEngine}
                rivalDomain={rivalDomain}
              />
            </div>
          ) : (
            <p className="py-8 text-sm text-muted-foreground">{t("home:detail.noAnswers")}</p>
          )}
        </DialogContent>
      </Dialog>
      {prompt.data && (
        <PromptDialog
          prompt={prompt.data}
          label={t("management:editPrompt")}
          pending={mutations.update.isPending}
          hideTrigger
          open={editing}
          onOpenChange={setEditing}
          onSave={(request) =>
            mutations.update.mutateAsync({ id: promptId, request }).then(() => undefined)
          }
        />
      )}
    </article>
  );
}
