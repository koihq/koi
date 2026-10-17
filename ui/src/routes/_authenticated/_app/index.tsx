import {
  ChatGptIcon,
  GoogleGeminiIcon,
  GoogleIcon,
  InformationCircleIcon,
  PerplexityAiIcon,
  Settings01Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import NumberFlow, { useCanAnimate } from "@number-flow/react";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
  createStandardSchemaV1,
  parseAsNumberLiteral,
  parseAsString,
  parseAsStringLiteral,
  useQueryStates,
} from "nuqs";
import { memo, useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Competitor } from "@/api/competitors.api";
import type { VisibilityEngine } from "@/api/generated";
import type { Prompt } from "@/api/prompts.api";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { BrandPill } from "@/components/blocks/brand-pill";
import { PromptManager, promptSearchParams } from "@/components/blocks/prompt-manager";
import { TrackingSummary } from "@/components/blocks/tracking-review";
import { type InspectionInput, VisibilityChart } from "@/components/blocks/visibility-chart";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useTracking } from "@/hooks/use-tracking";
import {
  engines,
  mockChangePoints,
  mockRate,
  mockSeries,
  type VisibilityScope,
} from "@/lib/mock-visibility";
import { competitorKey, promptKey } from "@/lib/query-keys";

const graphSearchParams = {
  days: parseAsNumberLiteral([7, 30, 90] as const)
    .withDefault(30)
    .withOptions({ clearOnDefault: true }),
  engines: parseAsString.withDefault("").withOptions({ clearOnDefault: true }),
  compareId: parseAsString.withDefault("").withOptions({ clearOnDefault: true }),
};

const engineIcons = {
  chatgpt: ChatGptIcon,
  perplexity: PerplexityAiIcon,
  gemini: GoogleGeminiIcon,
  google_ai_overviews: GoogleIcon,
};

const MemoPromptManager = memo(PromptManager);

export const Route = createFileRoute("/_authenticated/_app/")({
  validateSearch: createStandardSchemaV1(
    {
      ...promptSearchParams,
      ...graphSearchParams,
      draftStart: parseAsStringLiteral(["failed"] as const),
    },
    { partialOutput: true },
  ),
  component: Home,
});

async function allPages<T>(fetchPage: (page: number) => Promise<{ items: T[]; hasNext: boolean }>) {
  const items: T[] = [];
  for (let page = 1; ; page += 1) {
    const response = await fetchPage(page);
    items.push(...response.items);
    if (!response.hasNext) return items;
  }
}

function Home() {
  const { t, i18n } = useTranslation("home");
  const { brand } = Route.useRouteContext();
  const { draftStart } = Route.useSearch();
  const api = useApi();
  const tracking = useTracking(brand.id);
  const [graph, setGraph] = useQueryStates(graphSearchParams);
  const [inspected, setInspected] = useState<number | null>(null);
  const [inspectionInput, setInspectionInput] = useState<InspectionInput>("keyboard");
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [detailsPinned, setDetailsPinned] = useState(false);
  const canAnimateNumbers = useCanAnimate();
  const selectedEngines = useMemo(
    () =>
      graph.engines
        .split(",")
        .filter((id): id is VisibilityEngine => engines.some((engine) => engine.id === id)),
    [graph.engines],
  );
  const scope: VisibilityScope = useMemo(
    () => ({ days: graph.days, engines: selectedEngines }),
    [graph.days, selectedEngines],
  );

  const prompts = useQuery({
    queryKey: [...promptKey(brand.id), "analytics"],
    queryFn: ({ signal }) =>
      allPages<Prompt>((page) => api.prompts.list(brand.id, { page, pageSize: 50 }, { signal })),
  });
  const competitors = useQuery({
    queryKey: [...competitorKey(brand.id), "analytics"],
    queryFn: ({ signal }) =>
      allPages<Competitor>((page) =>
        api.competitors.list(brand.id, { page, pageSize: 50 }, { signal }),
      ),
  });

  const confirmedPrompts = useMemo(
    () => (prompts.data ?? []).filter((prompt) => prompt.confirmed),
    [prompts.data],
  );
  const confirmedCompetitors = useMemo(
    () => (competitors.data ?? []).filter((competitor) => competitor.confirmed),
    [competitors.data],
  );
  const comparison = confirmedCompetitors.find((competitor) => competitor.id === graph.compareId);
  const { series, comparedSeries, rate, change, rivals, comparisonRate } = useMemo(() => {
    const series = mockSeries(brand.id, confirmedPrompts, scope);
    const comparedSeries = comparison
      ? mockSeries(brand.id, confirmedPrompts, scope, comparison)
      : [];
    const rate = mockRate(brand.id, confirmedPrompts, scope);
    const change = mockChangePoints(brand.id, confirmedPrompts, scope);
    const rivals = confirmedCompetitors
      .map((competitor) => ({
        competitor,
        rate: mockRate(brand.id, confirmedPrompts, scope, competitor),
      }))
      .sort((left, right) => (right.rate ?? 0) - (left.rate ?? 0));
    const comparisonRate = comparison
      ? mockRate(brand.id, confirmedPrompts, scope, comparison)
      : null;

    return { series, comparedSeries, rate, change, rivals, comparisonRate };
  }, [brand.id, confirmedPrompts, scope, comparison, confirmedCompetitors]);
  const selectedPoint = inspected === null ? undefined : series[inspected];
  const selectedComparisonPoint = inspected === null ? undefined : comparedSeries[inspected];
  const headlineRate = selectedPoint?.rate ?? rate;
  const rawChange = selectedPoint
    ? inspected !== null && inspected > 0
      ? selectedPoint.rate - series[inspected - 1].rate
      : null
    : change;
  const headlineChange = rawChange === null ? null : Number(rawChange.toFixed(1));
  const previousRate = selectedPoint
    ? inspected !== null && inspected > 0
      ? series[inspected - 1].rate
      : null
    : rate !== null && change !== null
      ? rate - change
      : null;
  const animateNumbers = canAnimateNumbers && inspectionInput === "pointer";
  const retryPrompts = useCallback(() => void prompts.refetch(), [prompts.refetch]);

  const inspect = useCallback((index: number | null, input: InspectionInput) => {
    setInspectionInput(input);
    setInspected(index);
    if (index !== null) {
      setDetailsOpen(false);
      setDetailsPinned(false);
    }
  }, []);

  return (
    <div className="grid gap-6 pb-24">
      <h1 className="sr-only">{t("dashboard.title")}</h1>
      <section
        className="min-w-0 rounded-lg bg-surface p-4 sm:p-7"
        aria-labelledby="signal-heading"
        onPointerDownCapture={(event) =>
          setInspectionInput(event.pointerType === "touch" ? "touch" : "pointer")
        }
        onKeyDownCapture={() => setInspectionInput("keyboard")}
      >
        <div className="flex flex-col items-start justify-between gap-4 sm:flex-row">
          <div>
            <time
              className="text-xs font-normal text-muted-foreground"
              dateTime={new Date().toISOString()}
            >
              {new Date().toLocaleDateString(i18n.language, {
                weekday: "long",
                month: "long",
                day: "numeric",
              })}
            </time>
            <h2 id="signal-heading" className="mt-0.5 text-xl leading-tight font-bold">
              {t("dashboard.aiVisibility")}
            </h2>
            {prompts.isPending || competitors.isPending ? (
              <p className="mt-6 text-sm text-muted-foreground">{t("dashboard.loading")}</p>
            ) : prompts.isError || competitors.isError ? (
              <p role="alert" className="mt-6 text-sm">
                {t("dashboard.loadError")}{" "}
                <Button
                  variant="link"
                  onClick={() => {
                    void prompts.refetch();
                    void competitors.refetch();
                  }}
                >
                  {t("dashboard.retry")}
                </Button>
              </p>
            ) : (
              <div className="mt-4 flex flex-wrap items-baseline gap-2 text-[13px]">
                {comparison && comparisonRate !== null ? (
                  <div className="flex flex-wrap items-center gap-5">
                    {(
                      [
                        { subject: brand, value: selectedPoint?.rate ?? rate },
                        {
                          subject: comparison,
                          value: selectedComparisonPoint?.rate ?? comparisonRate,
                        },
                      ] as const
                    ).map(({ subject, value }) => (
                      <span
                        key={subject.id}
                        className="inline-flex items-center gap-2 whitespace-nowrap"
                      >
                        <BrandIcon name={subject.name} domain={subject.domain} className="size-6" />
                        <strong className="text-[36px] leading-[1.1] font-medium tracking-[-0.025em] tabular-nums">
                          {value === null ? (
                            "—"
                          ) : (
                            <NumberFlow
                              value={Number(value.toFixed(1))}
                              locales={i18n.language}
                              format={{ minimumFractionDigits: 1, maximumFractionDigits: 1 }}
                              animated={animateNumbers}
                              className="[--number-flow-mask-height:0.3em]"
                            />
                          )}
                          {value !== null && (
                            <span className="ml-0.5 text-[0.667em] text-muted-foreground">
                              {t("dashboard.percentSign")}
                            </span>
                          )}
                        </strong>
                      </span>
                    ))}
                  </div>
                ) : (
                  <>
                    <strong className="whitespace-nowrap text-[36px] leading-[1.1] font-medium tracking-[-0.025em] tabular-nums">
                      {headlineRate === null ? (
                        "—"
                      ) : (
                        <NumberFlow
                          value={Number(headlineRate.toFixed(1))}
                          locales={i18n.language}
                          format={{ minimumFractionDigits: 1, maximumFractionDigits: 1 }}
                          animated={animateNumbers}
                          className="[--number-flow-mask-height:0.3em]"
                        />
                      )}
                      {headlineRate !== null && (
                        <span className="ml-0.5 text-[0.667em] text-muted-foreground">
                          {t("dashboard.percentSign")}
                        </span>
                      )}
                    </strong>
                    <span
                      className={
                        headlineChange === null
                          ? "text-muted-foreground"
                          : headlineChange >= 0
                            ? "text-positive"
                            : "text-negative"
                      }
                    >
                      {headlineChange === null ? (
                        "—"
                      ) : (
                        <NumberFlow
                          value={Math.abs(headlineChange)}
                          prefix={headlineChange > 0 ? "+" : headlineChange < 0 ? "−" : undefined}
                          locales={i18n.language}
                          format={{ minimumFractionDigits: 1, maximumFractionDigits: 1 }}
                          animated={animateNumbers}
                          className="[--number-flow-mask-height:0.3em]"
                        />
                      )}
                    </span>
                  </>
                )}
                {selectedPoint && (
                  <time
                    className="text-xs font-normal text-muted-foreground"
                    dateTime={selectedPoint.date.toISOString()}
                  >
                    {selectedPoint.date.toLocaleDateString(i18n.language, {
                      month: "short",
                      day: "numeric",
                    })}
                  </time>
                )}
                <TooltipProvider delay={350}>
                  <Tooltip
                    open={detailsOpen}
                    onOpenChange={(open, details) => {
                      if (
                        detailsPinned &&
                        (details.reason === "trigger-hover" || details.reason === "trigger-focus")
                      ) {
                        return;
                      }
                      setDetailsOpen(open);
                      if (!open) setDetailsPinned(false);
                    }}
                  >
                    <TooltipTrigger
                      render={
                        <Button
                          variant="ghost"
                          size="icon-lg"
                          className="relative -top-[7px] self-end"
                        />
                      }
                      aria-label={t("dashboard.metricDetails")}
                      aria-expanded={detailsOpen}
                      closeOnClick={false}
                      onClick={() => {
                        setDetailsPinned(!detailsPinned);
                        setDetailsOpen(!detailsPinned);
                      }}
                    >
                      <HugeiconsIcon icon={InformationCircleIcon} size={16} aria-hidden="true" />
                    </TooltipTrigger>
                    <TooltipContent
                      className="metric-details-tooltip block w-fit max-w-[32ch] rounded-[10px] border border-(--metric-tooltip-border) bg-(--metric-tooltip) p-3 text-xs leading-normal text-foreground data-[state=delayed-open]:animate-none data-open:animate-none data-closed:animate-none"
                      data-input={inspectionInput}
                      side="bottom"
                      align="start"
                    >
                      <div className="metric-details grid gap-2">
                        <strong className="font-semibold">
                          {selectedPoint
                            ? selectedPoint.date.toLocaleDateString(i18n.language, {
                                month: "short",
                                day: "numeric",
                              })
                            : t("dashboard.metricHeading")}
                        </strong>
                        <p className="text-muted-foreground">{t("dashboard.metricPreview")}</p>
                        <p className="text-muted-foreground">
                          {previousRate === null
                            ? t(
                                selectedPoint
                                  ? "dashboard.metricNoPreviousDay"
                                  : "dashboard.metricNoPreviousPeriod",
                              )
                            : t(
                                selectedPoint
                                  ? "dashboard.metricPreviousDay"
                                  : "dashboard.metricPreviousPeriod",
                                { rate: previousRate.toFixed(1), days: graph.days },
                              )}
                        </p>
                        <p className="text-muted-foreground">{t("dashboard.metricSavedAnswers")}</p>
                      </div>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
            )}
          </div>
          <fieldset className="flex shrink-0 gap-1" aria-label={t("dashboard.filterEngines")}>
            {engines.map((engine) => {
              const selected = selectedEngines.includes(engine.id);
              return (
                <button
                  key={engine.id}
                  type="button"
                  title={engine.name}
                  aria-label={engine.name}
                  aria-pressed={selected}
                  className="grid size-10 place-items-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground aria-pressed:bg-muted aria-pressed:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  onClick={() => {
                    const next = selected
                      ? selectedEngines.filter((id) => id !== engine.id)
                      : [...selectedEngines, engine.id];
                    void setGraph({ engines: next.join(",") }, { history: "push" });
                    setInspected(null);
                  }}
                >
                  <HugeiconsIcon icon={engineIcons[engine.id]} size={23} aria-hidden="true" />
                </button>
              );
            })}
          </fieldset>
        </div>

        {rate === null ? (
          <div className="grid min-h-[244px] content-center justify-items-start gap-3 text-[13px] text-muted-foreground">
            <p>{t("dashboard.emptyDescription")}</p>
            <a href="#prompts" className="underline underline-offset-4">
              {t("dashboard.managePrompts")}
            </a>
          </div>
        ) : (
          <VisibilityChart
            series={series}
            comparison={comparedSeries}
            brand={brand}
            competitor={comparison}
            inspected={inspected}
            inspectionInput={inspectionInput}
            onInspect={inspect}
            locale={i18n.language}
            scopeKey={`${brand.id}:${graph.days}:${graph.engines}:${graph.compareId}`}
          />
        )}

        <div className="mt-4 flex items-center justify-between gap-4">
          <span className="text-xs font-normal text-muted-foreground">
            {series.length > 0 &&
              `${series[0].date.toLocaleDateString(i18n.language, { month: "short", day: "numeric" })} – ${series.at(-1)?.date.toLocaleDateString(i18n.language, { month: "short", day: "numeric" })}`}
          </span>
          <fieldset className="flex gap-0.5" aria-label={t("dashboard.range")}>
            {([7, 30, 90] as const).map((days) => (
              <button
                key={days}
                type="button"
                aria-pressed={graph.days === days}
                className="min-h-10 min-w-10 rounded-full px-2 text-xs text-muted-foreground hover:bg-muted hover:text-foreground aria-pressed:bg-muted aria-pressed:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                onClick={() => {
                  void setGraph({ days }, { history: "push" });
                  setInspected(null);
                }}
              >
                {`${days}D`}
              </button>
            ))}
          </fieldset>
        </div>

        <div className="mt-6 text-[13px]">
          <div className="flex items-center justify-between">
            <h3>{t("dashboard.competitors")}</h3>
            <Link
              to="/competitors"
              className="grid size-8 place-items-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
              aria-label={t("dashboard.manageCompetitors")}
              title={t("dashboard.manageCompetitors")}
            >
              <HugeiconsIcon icon={Settings01Icon} size={16} aria-hidden="true" />
            </Link>
          </div>
          {rivals.length ? (
            rivals.map(({ competitor, rate: rivalRate }) => {
              const gap = rivalRate !== null && rate !== null ? rivalRate - rate : null;
              return (
                <button
                  key={competitor.id}
                  type="button"
                  aria-pressed={graph.compareId === competitor.id}
                  onClick={() => {
                    void setGraph(
                      { compareId: graph.compareId === competitor.id ? "" : competitor.id },
                      { history: "push" },
                    );
                    setInspected(null);
                  }}
                  className="grid min-h-11 w-full grid-cols-[minmax(0,1fr)_60px_65px] items-center gap-1.5 rounded-md p-2 text-start hover:bg-muted aria-pressed:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring sm:grid-cols-[minmax(0,1fr)_76px_88px] sm:gap-4 max-[359px]:grid-cols-[minmax(0,1fr)_auto] max-[359px]:gap-y-2"
                >
                  <span className="min-w-0 justify-self-start max-[359px]:col-span-2">
                    <BrandPill name={competitor.name} domain={competitor.domain} />
                  </span>
                  <span className="text-end tabular-nums max-[359px]:text-start">
                    {rivalRate === null ? "—" : `${rivalRate.toFixed(1)}%`}
                  </span>
                  <span
                    className={`text-end tabular-nums ${gap === null ? "" : gap > 0 ? "text-negative" : "text-positive"}`}
                  >
                    {gap === null ? "—" : `${gap > 0 ? "+" : ""}${gap.toFixed(1)} pp`}
                  </span>
                </button>
              );
            })
          ) : (
            <p className="text-xs font-normal text-muted-foreground py-2">
              {t("dashboard.noCompetitors")}
            </p>
          )}
        </div>
      </section>

      {tracking.data && confirmedPrompts.length === 0 && (
        <div className="mt-6 border-l-2 border-border pl-4">
          <TrackingSummary brandID={brand.id} />
        </div>
      )}
      <MemoPromptManager
        brandID={brand.id}
        scope={scope}
        competitors={confirmedCompetitors}
        allPrompts={prompts.data ?? []}
        loading={prompts.isPending}
        failed={prompts.isError}
        onRetry={retryPrompts}
        draftStartFailed={draftStart === "failed"}
      />
      {confirmedPrompts.length > 0 && (
        <p className="text-xs font-normal text-muted-foreground mt-8">
          {t("dashboard.sampleAnalytics")}
        </p>
      )}
    </div>
  );
}
