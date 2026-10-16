import {
  ChatGptIcon,
  GoogleGeminiIcon,
  GoogleIcon,
  InformationCircleIcon,
  PerplexityAiIcon,
  Settings01Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
  createStandardSchemaV1,
  parseAsNumberLiteral,
  parseAsString,
  parseAsStringLiteral,
  useQueryStates,
} from "nuqs";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Competitor } from "@/api/competitors.api";
import type { VisibilityEngine } from "@/api/generated";
import type { Prompt } from "@/api/prompts.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { PromptManager, promptSearchParams } from "@/components/blocks/prompt-manager";
import { TrackingSummary } from "@/components/blocks/tracking-review";
import { VisibilityChart } from "@/components/blocks/visibility-chart";
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
  const selectedEngines = graph.engines
    .split(",")
    .filter((id): id is VisibilityEngine => engines.some((engine) => engine.id === id));
  const scope: VisibilityScope = { days: graph.days, engines: selectedEngines };

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
  const selectedPoint = inspected === null ? undefined : series[inspected];

  return (
    <div className="grid gap-6 pb-24">
      <h1 className="sr-only">{t("dashboard.title")}</h1>
      <section
        className="min-w-0 rounded-lg bg-surface p-4 sm:p-7"
        aria-labelledby="signal-heading"
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
              <div className="mt-4 flex items-baseline gap-2 whitespace-nowrap text-[13px]">
                <strong className="text-4xl leading-[1.1] font-medium tracking-[-0.025em] tabular-nums">
                  {selectedPoint || rate !== null
                    ? `${(selectedPoint?.rate ?? rate)?.toFixed(1)}%`
                    : "—"}
                </strong>
                {!selectedPoint && change !== null && (
                  <span className={change >= 0 ? "text-positive" : "text-negative"}>
                    {`${change > 0 ? "+" : ""}${change.toFixed(1)}`}
                  </span>
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
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <button
                          type="button"
                          className="grid size-8 place-items-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                        />
                      }
                      aria-label={t("dashboard.metricDetails")}
                    >
                      <HugeiconsIcon icon={InformationCircleIcon} size={16} aria-hidden="true" />
                    </TooltipTrigger>
                    <TooltipContent className="max-w-72 text-xs">
                      {t("dashboard.metricExplanation")}
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
            inspected={inspected}
            onInspect={setInspected}
            locale={i18n.language}
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
                  onClick={() =>
                    void setGraph(
                      { compareId: graph.compareId === competitor.id ? "" : competitor.id },
                      { history: "push" },
                    )
                  }
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
      <PromptManager
        brandID={brand.id}
        scope={scope}
        competitors={confirmedCompetitors}
        allPrompts={prompts.data ?? []}
        loading={prompts.isPending}
        failed={prompts.isError}
        onRetry={() => void prompts.refetch()}
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
