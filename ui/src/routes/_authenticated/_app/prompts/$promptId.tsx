import NumberFlow, { useCanAnimate } from "@number-flow/react";
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
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Competitor } from "@/api/competitors.api";
import type { VisibilityEngine, VisibilityResult } from "@/api/generated";
import { BrandPill } from "@/components/blocks/brand-pill";
import { type InspectionInput, VisibilityChart } from "@/components/blocks/visibility-chart";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { engines, mockRate, mockSeries, type VisibilityScope } from "@/lib/mock-visibility";
import { competitorKey, promptKey, runKey } from "@/lib/query-keys";
import { safeSourceURL } from "@/lib/source-url";
import { PromptDialog } from "@/routes/_authenticated/_app/prompts/-components/prompt-dialog";
import { usePromptMutations } from "@/routes/_authenticated/_app/prompts/-hooks/use-prompt-mutations";

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
  const { t, i18n } = useTranslation(["home", "management", "visibility"]);
  const { brand } = Route.useRouteContext();
  const { promptId } = Route.useParams();
  const api = useApi();
  const mutations = usePromptMutations(brand.id);
  const [search, setSearch] = useQueryStates(detailSearchParams);
  const [inspected, setInspected] = useState<number | null>(null);
  const [inspectionInput, setInspectionInput] = useState<InspectionInput>("keyboard");
  const canAnimateNumbers = useCanAnimate();
  const [editing, setEditing] = useState(false);
  const runTrigger = useRef<HTMLButtonElement | null>(null);
  const editTrigger = useRef<HTMLButtonElement | null>(null);

  const prompt = useQuery({
    queryKey: [...promptKey(brand.id), promptId],
    queryFn: () => api.prompts.getById(brand.id, promptId),
  });
  const history = useQuery({
    queryKey: [...runKey(brand.id), "prompt", promptId, search.page],
    queryFn: ({ signal }) =>
      api.runs.promptHistory(brand.id, promptId, search.page, 10, { signal }),
  });
  const competitors = useQuery({
    queryKey: [...competitorKey(brand.id), "detail-comparison"],
    queryFn: async ({ signal }) => {
      const items: Competitor[] = [];
      for (let page = 1; ; page += 1) {
        const response = await api.competitors.list(brand.id, { page, pageSize: 50 }, { signal });
        items.push(...response.items);
        if (!response.hasNext) return items.filter((item) => item.confirmed);
      }
    },
  });

  const runs = history.data?.items ?? [];
  const run = runs.find((item) => item.run.id === search.runId);
  const result = run?.results.find((item) => item.engine === search.responseEngine);
  const title = prompt.data?.text ?? runs[0]?.results[0]?.promptText;
  const scope: VisibilityScope = { days: search.days, engines: [] };
  const measured = prompt.data?.confirmed ? [prompt.data] : [];
  const comparison = competitors.data?.find((item) => item.id === search.compareId);
  const series = mockSeries(brand.id, measured, scope);
  const comparedSeries = comparison ? mockSeries(brand.id, measured, scope, comparison) : [];
  const rate = mockRate(brand.id, measured, scope);
  const rivals = (competitors.data ?? [])
    .map((competitor) => ({
      competitor,
      rate: mockRate(brand.id, measured, scope, competitor),
    }))
    .sort((left, right) => (right.rate ?? -1) - (left.rate ?? -1));
  const metricRate = series[inspected ?? -1]?.rate ?? rate;

  function inspect(index: number | null, input: InspectionInput) {
    setInspectionInput(input);
    setInspected(index);
  }

  function openRun(id: string, trigger: HTMLButtonElement, engine?: VisibilityEngine) {
    runTrigger.current = trigger;
    void setSearch({ runId: id, responseEngine: engine ?? "chatgpt" }, { history: "push" });
  }

  return (
    <article className="grid gap-6 pb-24">
      <Link
        to="/"
        hash="prompts"
        className="text-xs font-normal text-muted-foreground hover:text-foreground"
      >
        {t("home:detail.allPrompts")}
      </Link>
      <div className="mb-6 flex items-start justify-between gap-6">
        <h1 className="max-w-[75ch] text-xl leading-[1.4] font-bold tracking-[-0.015em]">
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

      <div className="grid gap-4 min-[801px]:grid-cols-[minmax(0,1fr)_332px]">
        <section
          className="min-w-0 rounded-lg bg-surface p-5 min-[801px]:min-h-[412px]"
          aria-labelledby="detail-visibility-heading"
          onPointerDownCapture={(event) =>
            setInspectionInput(event.pointerType === "touch" ? "touch" : "pointer")
          }
          onKeyDownCapture={() => setInspectionInput("keyboard")}
        >
          <div className="flex items-start justify-between gap-3">
            <div>
              <h2 id="detail-visibility-heading" className="text-base font-bold">
                {t("home:dashboard.aiVisibility")}
              </h2>
              <strong className="mt-4 block text-[36px] leading-[1.1] font-medium tracking-[-0.025em] tabular-nums">
                {metricRate === null ? (
                  "—"
                ) : (
                  <NumberFlow
                    value={Number(metricRate.toFixed(1))}
                    locales={i18n.language}
                    format={{ minimumFractionDigits: 1, maximumFractionDigits: 1 }}
                    animated={canAnimateNumbers && inspectionInput === "pointer"}
                    className="[--number-flow-mask-height:0.3em]"
                  />
                )}
                {metricRate !== null && (
                  <span className="ml-0.5 text-[0.667em] text-muted-foreground">
                    {t("home:dashboard.percentSign")}
                  </span>
                )}
              </strong>
              <p className="text-xs font-normal text-muted-foreground">
                {t("home:detail.runsCount", { count: history.data?.totalItems ?? 0 })}
              </p>
            </div>
            <fieldset className="flex gap-0.5" aria-label={t("home:dashboard.range")}>
              {([7, 30, 90] as const).map((days) => (
                <button
                  key={days}
                  type="button"
                  aria-pressed={search.days === days}
                  className="min-h-10 min-w-10 rounded-full px-2 text-xs text-muted-foreground hover:bg-muted hover:text-foreground aria-pressed:bg-muted aria-pressed:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  onClick={() => {
                    void setSearch({ days }, { history: "push" });
                    setInspected(null);
                  }}
                >
                  {`${days}D`}
                </button>
              ))}
            </fieldset>
          </div>
          {series.length ? (
            <VisibilityChart
              series={series}
              comparison={comparedSeries}
              brand={brand}
              competitor={comparison}
              inspected={inspected}
              inspectionInput={inspectionInput}
              onInspect={inspect}
              locale={i18n.language}
              variant="detail"
            />
          ) : (
            <p className="grid min-h-[244px] content-center justify-items-start gap-3 text-[13px] text-muted-foreground">
              {t("home:detail.noAnswers")}
            </p>
          )}
          {series.length > 0 && (
            <p className="text-xs font-normal text-muted-foreground flex justify-between">
              <time dateTime={series[0].date.toISOString()}>
                {series[0].date.toLocaleDateString(i18n.language)}
              </time>
              <time dateTime={series.at(-1)?.date.toISOString()}>
                {series.at(-1)?.date.toLocaleDateString(i18n.language)}
              </time>
            </p>
          )}
        </section>
        <aside
          className="min-w-0 rounded-lg bg-surface p-5 min-[801px]:min-h-[412px]"
          aria-labelledby="detail-compare-heading"
        >
          <h2 id="detail-compare-heading" className="mb-4 text-base font-bold">
            {t("home:detail.compareWith")}
          </h2>
          {rivals.length ? (
            rivals.map(({ competitor, rate: rivalRate }) => (
              <button
                key={competitor.id}
                type="button"
                aria-pressed={comparison?.id === competitor.id}
                className="grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-x-2 gap-y-1 rounded-md p-2 text-start hover:bg-muted aria-pressed:bg-muted"
                onClick={() => {
                  void setSearch(
                    { compareId: comparison?.id === competitor.id ? "" : competitor.id },
                    { history: "push" },
                  );
                  setInspected(null);
                }}
              >
                <span className="justify-self-start">
                  <BrandPill name={competitor.name} domain={competitor.domain} />
                </span>
                <span className="tabular-nums">
                  {rivalRate === null ? "—" : `${rivalRate.toFixed(1)}%`}
                </span>
                <small className="col-span-full text-xs text-muted-foreground">
                  {rivalRate !== null && rate !== null
                    ? t("home:detail.comparisonGap", {
                        points: Math.abs(rivalRate - rate).toFixed(1),
                        direction:
                          rivalRate > rate ? t("home:detail.higher") : t("home:detail.lower"),
                      })
                    : t("home:detail.noComparisonData")}
                </small>
              </button>
            ))
          ) : (
            <p className="text-xs font-normal text-muted-foreground">
              {t("home:dashboard.noCompetitors")}
            </p>
          )}
        </aside>
      </div>

      <section className="mt-8" aria-labelledby="run-history-heading">
        <div className="flex items-center justify-between">
          <h2 id="run-history-heading" className="text-base font-bold">
            {t("home:detail.runHistory")}
          </h2>
          <span className="text-xs font-normal text-muted-foreground">
            {t("home:detail.runsCount", { count: history.data?.totalItems ?? 0 })}
          </span>
        </div>
        {history.isError ? (
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
          <div className="overflow-x-auto">
            <table className="mt-5 w-full min-w-[680px] table-fixed border-collapse">
              <thead>
                <tr>
                  <th className="w-40 p-2 text-start text-xs font-normal text-muted-foreground">
                    {t("home:detail.runDate")}
                  </th>
                  <th className="p-2 text-start text-xs font-normal text-muted-foreground">
                    {t("home:detail.engineResults")}
                  </th>
                  <th className="w-[220px] p-2 text-start text-xs font-normal text-muted-foreground">
                    {t("home:detail.competitorsFound")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {runs.map((item) => {
                  const found = [
                    ...new Map(
                      item.results
                        .flatMap((cell) => cell.appearances)
                        .map((appearance) => [appearance.name, appearance]),
                    ).values(),
                  ];
                  return (
                    <tr key={item.run.id}>
                      <td className="border-t border-border px-2 py-3 align-top text-[13px]">
                        <button
                          type="button"
                          className="grid gap-0.5 text-start hover:underline"
                          onClick={(event) => openRun(item.run.id, event.currentTarget)}
                        >
                          <time dateTime={item.run.createdAt.toISOString()}>
                            {item.run.createdAt.toLocaleDateString(i18n.language)}
                          </time>
                          <small className="text-muted-foreground">
                            {item.run.createdAt.toLocaleTimeString(i18n.language, {
                              hour: "numeric",
                              minute: "2-digit",
                            })}
                          </small>
                        </button>
                      </td>
                      <td className="border-t border-border px-2 py-3 align-top text-[13px]">
                        <div className="flex flex-wrap gap-1.5">
                          {engines.map((engine) => {
                            const cell = item.results.find((entry) => entry.engine === engine.id);
                            const status = cell?.outcome ?? (cell?.errorCode ? "error" : "pending");
                            return (
                              <button
                                key={engine.id}
                                type="button"
                                className="inline-flex items-center gap-1 whitespace-nowrap rounded-full border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-muted hover:text-foreground"
                                onClick={(event) =>
                                  openRun(item.run.id, event.currentTarget, engine.id)
                                }
                                aria-label={`${engine.name}: ${t(`visibility:outcomes.${status}`, { defaultValue: status })}`}
                              >
                                <BrandPill
                                  name={engine.name}
                                  domain={
                                    engine.id === "chatgpt"
                                      ? "chatgpt.com"
                                      : engine.id === "perplexity"
                                        ? "perplexity.ai"
                                        : engine.id === "gemini"
                                          ? "gemini.google.com"
                                          : "google.com"
                                  }
                                  iconOnly
                                />
                                <span>
                                  {t(`visibility:outcomes.${status}`, { defaultValue: status })}
                                </span>
                              </button>
                            );
                          })}
                        </div>
                      </td>
                      <td className="border-t border-border px-2 py-3 align-top text-[13px]">
                        <div className="flex flex-wrap gap-1.5">
                          {found.length ? (
                            found.map((appearance) => (
                              <BrandPill
                                key={appearance.name}
                                name={appearance.name}
                                domain={
                                  competitors.data?.find(
                                    (candidate) => candidate.name === appearance.name,
                                  )?.domain
                                }
                              />
                            ))
                          ) : (
                            <span className="text-xs font-normal text-muted-foreground">{"—"}</span>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
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

      {measured.length > 0 && (
        <p className="text-xs font-normal text-muted-foreground mt-12">
          {t("home:dashboard.sampleAnalytics")}
        </p>
      )}

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
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
            <DialogDescription>
              {run?.run.createdAt.toLocaleString(i18n.language)}
            </DialogDescription>
          </DialogHeader>
          {run ? (
            <>
              <fieldset
                className="flex flex-wrap gap-2"
                aria-label={t("home:detail.answersByEngine")}
              >
                {engines.map((engine) => {
                  const cell = run.results.find((entry) => entry.engine === engine.id);
                  return (
                    <button
                      key={engine.id}
                      type="button"
                      aria-pressed={search.responseEngine === engine.id}
                      className="flex items-center gap-3 rounded-md border border-border px-3 py-2 text-xs hover:bg-muted aria-pressed:bg-muted"
                      onClick={() =>
                        void setSearch({ responseEngine: engine.id }, { history: "replace" })
                      }
                    >
                      <BrandPill
                        name={engine.name}
                        domain={
                          engine.id === "chatgpt"
                            ? "chatgpt.com"
                            : engine.id === "perplexity"
                              ? "perplexity.ai"
                              : engine.id === "gemini"
                                ? "gemini.google.com"
                                : "google.com"
                        }
                      />
                      <span>{cell?.outcome ?? "Pending"}</span>
                    </button>
                  );
                })}
              </fieldset>
              {result ? (
                <Answer result={result} />
              ) : (
                <p className="py-8 text-sm text-muted-foreground">
                  {t("home:detail.pendingAnswer")}
                </p>
              )}
            </>
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

function Answer({ result }: { result: VisibilityResult }) {
  const { t } = useTranslation(["home", "visibility"]);
  return (
    <div className="grid min-h-[300px] min-w-0 gap-4 lg:grid-cols-[minmax(0,1fr)_280px]">
      <article className="min-w-0 rounded-md bg-surface p-5">
        <div className="flex items-center justify-between gap-3">
          <strong>{t(`visibility:engines.${result.engine}`)}</strong>
          <span className="text-xs font-normal text-muted-foreground">
            {result.outcome ? t(`visibility:outcomes.${result.outcome}`) : t("visibility:pending")}
          </span>
        </div>
        {result.errorCode && (
          <p role="alert" className="text-sm text-destructive">
            {t("visibility:error", { code: result.errorCode })}
          </p>
        )}
        <p className="mt-6 whitespace-pre-wrap break-words text-sm leading-7">
          {result.rawAnswer ||
            (result.outcome === "miss"
              ? t("home:detail.noMention")
              : t("home:detail.pendingAnswer"))}
        </p>
        {result.appearances.length > 0 && (
          <div className="mt-8">
            <h3 className="text-[13px] font-medium">{t("home:detail.appearances")}</h3>
            <ul className="mt-3 space-y-2 text-sm">
              {result.appearances.map((appearance) => (
                <li key={appearance.competitorId ?? appearance.name}>
                  <span>{appearance.name}</span>
                  {appearance.evidence.length > 0 && (
                    <ul className="mt-1 space-y-1 text-xs text-muted-foreground">
                      {appearance.evidence.map((item) => {
                        const href =
                          item.kind === "citation" || item.kind === "link"
                            ? safeSourceURL(item.value)
                            : null;

                        return (
                          <li key={`${item.kind}:${item.value}`} className="break-all">
                            {`${t(`visibility:evidenceKinds.${item.kind}`)}: `}
                            {href ? (
                              <a
                                href={href}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="underline underline-offset-4"
                              >
                                {item.value}
                              </a>
                            ) : (
                              item.value
                            )}
                          </li>
                        );
                      })}
                    </ul>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}
      </article>
      <aside className="min-w-0 rounded-md bg-surface p-5">
        <h3 className="text-[13px] font-medium">{t("home:detail.sources")}</h3>
        {result.sources.length ? (
          <ul className="mt-4 space-y-3">
            {[...new Set(result.sources)].map((source) => {
              const href = safeSourceURL(source);
              return (
                <li key={source} className="break-all text-sm">
                  {href ? (
                    <a
                      href={href}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="underline underline-offset-4"
                    >
                      {source}
                    </a>
                  ) : (
                    source
                  )}
                </li>
              );
            })}
          </ul>
        ) : (
          <p className="mt-4 text-sm text-muted-foreground">{t("home:detail.noSources")}</p>
        )}
      </aside>
    </div>
  );
}
