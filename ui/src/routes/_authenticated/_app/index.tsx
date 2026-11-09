import { InformationCircleIcon, RankingIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import NumberFlow, { useCanAnimate } from "@number-flow/react";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useLocation } from "@tanstack/react-router";
import {
  createStandardSchemaV1,
  parseAsNumberLiteral,
  parseAsString,
  parseAsStringLiteral,
  useQueryStates,
} from "nuqs";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { VisibilityEngine } from "@/api/generated";
import { useBrandColor } from "@/components/blocks/brand-icon";
import { BrandLabel } from "@/components/blocks/brand-label";
import { chartRanges, EngineFilter, RangeControl } from "@/components/blocks/chart-controls";
import { QueryRefreshIndicator } from "@/components/blocks/query-refresh-indicator";
import { TrackingSummary } from "@/components/blocks/tracking-summary";
import { type InspectionInput, VisibilityChart } from "@/components/blocks/visibility-chart";
import { VisibilityHeadline } from "@/components/blocks/visibility-headline";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useCollection } from "@/hooks/use-collection";
import { useFlip } from "@/hooks/use-flip";
import { useTracking } from "@/hooks/use-tracking";
import { brandGlow, brandRowTint } from "@/lib/brand-tint";
import { formatDay, formatDayRange, formatPoints, formatRate } from "@/lib/format";
import { billingOptions, keepPreviousDataForScope, visibilitySummaryOptions } from "@/lib/queries";
import { promptKey } from "@/lib/query-keys";
import {
  engines,
  type VisibilityScope,
  change as visibilityChange,
  rate as visibilityRate,
  series as visibilitySeries,
} from "@/lib/visibility";
import {
  PromptManager,
  promptSearchParams,
} from "@/routes/_authenticated/_app/-components/prompt-manager";
import { FirstLook } from "./-components/first-look";
import { TrackingStart } from "./-components/tracking-start";

const graphSearchParams = {
  days: parseAsNumberLiteral(chartRanges).withDefault(30).withOptions({ clearOnDefault: true }),
  engines: parseAsString.withDefault("").withOptions({ clearOnDefault: true }),
  compareId: parseAsString.withDefault("").withOptions({ clearOnDefault: true }),
};

const MemoPromptManager = memo(PromptManager);

export const Route = createFileRoute("/_authenticated/_app/")({
  validateSearch: createStandardSchemaV1(
    {
      ...promptSearchParams,
      ...graphSearchParams,
      draftStart: parseAsStringLiteral(["failed", "unavailable"] as const),
    },
    { partialOutput: true },
  ),
  component: Home,
});

const FIRST_LOOK_SCOPE: VisibilityScope = { days: 30, engines: [] };

function Home() {
  const api = useApi();
  const { brand, capabilities, draftStartFailed, me } = Route.useRouteContext();
  const { draftStart } = Route.useSearch();
  const billing = useQuery({
    ...billingOptions(api, me.organization.id),
    enabled: capabilities.cloudBilling,
  });

  if (capabilities.cloudBilling && billing.data?.firstLook) {
    return (
      <FirstLook
        brand={brand}
        scope={FIRST_LOOK_SCOPE}
        draftStartFailed={draftStartFailed || draftStart === "failed"}
        draftUnavailable={draftStart === "unavailable"}
      />
    );
  }

  return <Dashboard />;
}

function Dashboard() {
  const { t, i18n } = useTranslation("home");
  const { brand, draftStartFailed } = Route.useRouteContext();
  const { draftStart } = Route.useSearch();
  const navigate = Route.useNavigate();
  const hash = useLocation({ select: (location) => location.hash });
  const api = useApi();
  const { collecting } = useCollection(brand.id);
  const tracking = useTracking(brand.id);
  const [graph, setGraph] = useQueryStates(graphSearchParams);
  const [inspected, setInspected] = useState<number | null>(null);

  const [inspectionInput, setInspectionInput] = useState<InspectionInput>("keyboard");
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [detailsPinned, setDetailsPinned] = useState(false);
  const canAnimateNumbers = useCanAnimate();

  // The router scrolls to the hash again on every navigation, and each filter is one. Once the
  // link has landed, drop the hash so a later filter change keeps the page where it is.
  useEffect(() => {
    if (!hash) return;

    const frame = requestAnimationFrame(() => {
      void navigate({ to: ".", search: true, replace: true, resetScroll: false });
    });

    return () => cancelAnimationFrame(frame);
  }, [hash, navigate]);

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

  const summary = useQuery({
    ...visibilitySummaryOptions(api, brand.id, scope),
    placeholderData: keepPreviousDataForScope(promptKey(brand.id)),
    refetchInterval: collecting ? 5_000 : false,
  });
  const hasConfirmedPrompts = summary.data?.hasConfirmedPrompts;
  const comparison = summary.data?.competitors.find(
    (competitor) => competitor.id === graph.compareId,
  );
  const compareColor = useBrandColor(comparison?.domain);
  const { series, comparedSeries, rate, change, rivals, comparisonRate } = useMemo(() => {
    const report = summary.data;
    const series = visibilitySeries(report);
    const comparedSeries = comparison ? visibilitySeries(report, comparison.id) : [];
    const rate = visibilityRate(report);
    const change = visibilityChange(report);
    const rivals = (report?.competitors ?? [])
      .map((competitor) => ({
        competitor,
        rate: competitor.metric.rate,
      }))
      // A rival with no mentions in this period was not recommended instead of anyone.
      .filter((rival) => rival.rate !== 0)
      .sort((left, right) => (right.rate ?? 0) - (left.rate ?? 0));
    const comparisonRate = comparison ? visibilityRate(report, undefined, comparison.id) : null;

    return { series, comparedSeries, rate, change, rivals, comparisonRate };
  }, [summary.data, comparison]);
  const rivalList = useRef<HTMLDivElement>(null);
  useFlip(rivalList, rivals.map(({ competitor }) => competitor.id).join(","));
  const selectedPoint = inspected === null ? undefined : series[inspected];
  const selectedComparisonPoint = inspected === null ? undefined : comparedSeries[inspected];
  const headlineRate = selectedPoint ? selectedPoint.rate : rate;
  const rawChange = selectedPoint
    ? inspected !== null && inspected > 0
      ? selectedPoint.rate !== null && series[inspected - 1].rate !== null
        ? selectedPoint.rate - (series[inspected - 1].rate ?? 0)
        : null
      : null
    : change;
  const headlineChange = rawChange === null ? null : Number(rawChange.toFixed(1));
  const trend = !headlineChange ? "Same" : headlineChange > 0 ? "Up" : "Down";
  const previousRate = selectedPoint
    ? inspected !== null && inspected > 0
      ? series[inspected - 1].rate
      : null
    : rate !== null && change !== null
      ? rate - change
      : null;
  const animateNumbers = canAnimateNumbers && inspectionInput === "pointer";

  const inspect = useCallback((index: number | null, input: InspectionInput) => {
    setInspectionInput(input);
    setInspected(index);
    if (index !== null) {
      setDetailsOpen(false);
      setDetailsPinned(false);
    }
  }, []);

  const empty = summary.isSuccess && rate === null;

  return (
    <div className="grid gap-6 pb-24">
      <h1 className="sr-only">{t("dashboard.title")}</h1>
      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
        <section
          className="group/refresh relative min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9"
          style={brandGlow(compareColor)}
          aria-busy={summary.isFetching}
          data-refreshing={summary.isFetching && Boolean(summary.data)}
          aria-labelledby="signal-heading"
          onPointerDownCapture={(event) =>
            setInspectionInput(event.pointerType === "touch" ? "touch" : "pointer")
          }
          onKeyDownCapture={() => setInspectionInput("keyboard")}
        >
          <QueryRefreshIndicator active={summary.isFetching} label={t("dashboard.loading")} />
          <div className="flex min-h-8 flex-wrap items-center justify-between gap-x-4 gap-y-2">
            <h2 id="signal-heading" className="text-section font-semibold">
              {t("dashboard.aiVisibility")}
            </h2>
            {!empty && (
              <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
                <EngineFilter
                  selected={selectedEngines}
                  onToggle={(engine) => {
                    const next = selectedEngines.includes(engine)
                      ? selectedEngines.filter((id) => id !== engine)
                      : [...selectedEngines, engine];

                    void setGraph({ engines: next.join(",") }, { history: "push" });
                    setInspected(null);
                  }}
                />
                <RangeControl
                  days={graph.days}
                  onChange={(days) => {
                    void setGraph({ days }, { history: "push" });
                    setInspected(null);
                  }}
                />
              </div>
            )}
          </div>

          {summary.isPending ? (
            <output className="mt-4 block">
              <span className="sr-only">{t("dashboard.loading")}</span>
              <Skeleton className="h-[42px] w-40" />
              <Skeleton className="mt-1 h-[18px] w-44" />
              <div className="h-[236px]" />
            </output>
          ) : summary.isError ? (
            <p role="alert" className="mt-6 text-sm">
              {t("dashboard.loadError")}{" "}
              <Button
                variant="link"
                onClick={() => {
                  void summary.refetch();
                }}
              >
                {t("dashboard.retry")}
              </Button>
            </p>
          ) : empty && tracking.isPending ? (
            <p className="mt-6 text-sm text-muted-foreground">{t("dashboard.loading")}</p>
          ) : empty && tracking.data?.confirmed === false ? (
            <TrackingStart
              brandID={brand.id}
              scope={scope}
              draftStartFailed={draftStartFailed || draftStart === "failed"}
            />
          ) : empty ? (
            <div className="grid min-h-60 content-center justify-items-start gap-4 group-data-[refreshing=true]/refresh:opacity-50">
              <output className="block max-w-[48ch] text-body text-muted-foreground">
                {t(
                  collecting
                    ? "dashboard.firstCheckRunning"
                    : hasConfirmedPrompts
                      ? "dashboard.noCompletedAnswers"
                      : "dashboard.emptyDescription",
                )}
              </output>
              {!hasConfirmedPrompts && (
                <Button render={<Link to="/" hash="prompts" />} nativeButton={false} size="lg">
                  {t("dashboard.managePrompts")}
                </Button>
              )}
            </div>
          ) : (
            <>
              <div className="mt-4 flex flex-wrap items-center gap-x-3 gap-y-2 group-data-[refreshing=true]/refresh:opacity-50">
                <VisibilityHeadline
                  brand={brand}
                  competitor={comparison}
                  rate={headlineRate}
                  comparisonRate={
                    selectedComparisonPoint ? selectedComparisonPoint.rate : comparisonRate
                  }
                  paired={Boolean(comparison && comparisonRate !== null)}
                  locale={i18n.language}
                  animated={animateNumbers}
                  variant="home"
                />
                {(!comparison || comparisonRate === null) && headlineChange !== null && (
                  <span
                    className={`rounded-chip px-1.5 py-0.5 text-xs tabular-nums ${headlineChange > 0 ? "bg-positive/12 text-positive" : headlineChange < 0 ? "bg-negative/12 text-negative" : "bg-muted text-muted-foreground"}`}
                  >
                    <NumberFlow
                      value={Math.abs(headlineChange)}
                      prefix={headlineChange > 0 ? "+" : headlineChange < 0 ? "−" : undefined}
                      suffix=" pp"
                      locales={i18n.language}
                      format={{ minimumFractionDigits: 1, maximumFractionDigits: 1 }}
                      animated={animateNumbers}
                      className="[--number-flow-mask-height:0.3em]"
                    />
                  </span>
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
                        <Button variant="ghost" size="icon-lg" className="text-muted-foreground" />
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
                      className="metric-details-tooltip surface-float block w-fit max-w-[32ch] p-3 text-xs leading-normal text-foreground data-[state=delayed-open]:animate-none data-open:animate-none data-closed:animate-none"
                      data-input={inspectionInput}
                      side="bottom"
                      align="start"
                    >
                      <div className="metric-details grid gap-2">
                        <strong className="font-semibold">
                          {selectedPoint
                            ? formatDay(selectedPoint.date, i18n.language)
                            : t("dashboard.metricHeading")}
                        </strong>
                        <p className="text-muted-foreground">
                          {previousRate === null
                            ? t(
                                selectedPoint
                                  ? "dashboard.metricNoPreviousDay"
                                  : "dashboard.metricNoPreviousPeriod",
                              )
                            : t(
                                `dashboard.metricPrevious${selectedPoint ? "Day" : "Period"}${trend}`,
                                {
                                  rate: previousRate.toFixed(1),
                                  // "Up" or "Down" already carries the sign.
                                  points: formatPoints(headlineChange ?? 0, i18n.language).replace(
                                    /^[+−]/,
                                    "",
                                  ),
                                  days: graph.days,
                                },
                              )}
                        </p>
                        <p className="text-muted-foreground">{t("dashboard.metricSavedAnswers")}</p>
                        <p className="text-muted-foreground">
                          {t("cadence", { ns: "visibility" })}
                        </p>
                      </div>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
              <p className="mt-1 min-h-[18px] text-xs text-muted-foreground tabular-nums group-data-[refreshing=true]/refresh:opacity-50">
                {selectedPoint ? (
                  <time dateTime={selectedPoint.date.toISOString()}>
                    {formatDay(selectedPoint.date, i18n.language)}
                  </time>
                ) : (
                  series.length > 0 &&
                  t("dashboard.periodAverage", {
                    range: formatDayRange(
                      series[0].date,
                      series[series.length - 1].date,
                      i18n.language,
                    ),
                  })
                )}
              </p>
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
                competitorColor={compareColor}
              />
            </>
          )}
        </section>

        <aside
          className="group/refresh min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9 xl:px-5"
          aria-labelledby="rivals-heading"
          data-refreshing={summary.isFetching && Boolean(summary.data)}
        >
          <div className="mb-3 flex min-h-8 items-center justify-between gap-3">
            <h2 id="rivals-heading" className="text-section font-semibold">
              {t("dashboard.recommendedInstead")}
            </h2>
            <Button
              variant="ghost"
              size="circle-sm"
              render={<Link to="/competitors" />}
              nativeButton={false}
              aria-label={t("dashboard.manageCompetitors")}
              title={t("dashboard.manageCompetitors")}
              className="-me-2 text-muted-foreground"
            >
              <HugeiconsIcon icon={RankingIcon} aria-hidden="true" />
            </Button>
          </div>
          {rivals.length ? (
            <div ref={rivalList} className="list-rows -mx-2 grid gap-0.5">
              {rivals.map(({ competitor, rate: rivalRate }) => {
                const gap = rivalRate !== null && rate !== null ? rivalRate - rate : null;
                return (
                  <button
                    key={competitor.id}
                    data-flip-key={competitor.id}
                    type="button"
                    aria-pressed={graph.compareId === competitor.id}
                    style={
                      graph.compareId === competitor.id ? brandRowTint(compareColor) : undefined
                    }
                    onClick={() => {
                      void setGraph(
                        { compareId: graph.compareId === competitor.id ? "" : competitor.id },
                        { history: "push" },
                      );
                      setInspected(null);
                    }}
                    className="grid min-h-11 w-full grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-3 rounded-control px-2 py-2 text-start text-body hover:bg-signal-pill aria-pressed:bg-signal-pill focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  >
                    <span className="flex min-w-0">
                      <BrandLabel name={competitor.name} domain={competitor.domain} />
                    </span>
                    <span className="text-end font-semibold tabular-nums group-data-[refreshing=true]/refresh:opacity-50">
                      {rivalRate === null ? "—" : formatRate(rivalRate, i18n.language)}
                    </span>
                    <span
                      className={`w-16 text-end text-xs tabular-nums group-data-[refreshing=true]/refresh:opacity-50 ${gap === null ? "text-muted-foreground" : gap > 0 ? "text-negative" : "text-positive"}`}
                    >
                      {gap === null ? "—" : formatPoints(gap, i18n.language)}
                    </span>
                  </button>
                );
              })}
            </div>
          ) : summary.isSuccess ? (
            <p className="py-2 text-xs text-muted-foreground">
              {t(
                tracking.data?.confirmed === false
                  ? "dashboard.competitorsAfterStart"
                  : summary.data?.competitors.length
                    ? "dashboard.noneRecommended"
                    : "dashboard.noCompetitors",
              )}
            </p>
          ) : null}
        </aside>
      </div>

      {hasConfirmedPrompts && tracking.data?.confirmed !== false && (
        <TrackingSummary brandID={brand.id} showSchedule={false} />
      )}
      <MemoPromptManager brandID={brand.id} scope={scope} collecting={collecting} />
    </div>
  );
}
