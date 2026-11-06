import NumberFlow from "@number-flow/react";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { memo, type ReactNode, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import type { VisibilityEngine, VisibilityResult, VisibilityRunStatus } from "@/api/generated";
import { BrandIcon, useBrandColor } from "@/components/blocks/brand-icon";
import { OutcomeMark } from "@/components/blocks/outcome-mark";
import { SetupProgress } from "@/components/blocks/setup-progress";
import { Button } from "@/components/ui/button";
import { useCollection } from "@/hooks/use-collection";
import { useCopy } from "@/hooks/use-copy";
import { useOnboardingStep, useOnboardingTracker } from "@/hooks/use-onboarding-step";
import { useTracking } from "@/hooks/use-tracking";
import { markAnswer } from "@/lib/answer-marks";
import { getAnswerState } from "@/lib/answer-state";
import { brandGlow } from "@/lib/brand-tint";
import {
  agentHandoffPrompt,
  answerPassage,
  type FirstLookStep,
  type FirstLookVerdict,
  firstLookPlan,
  firstLookRows,
  firstLookScore,
  firstLookVerdict,
  keepVerdict,
} from "@/lib/first-look";
import { generationOptions, invalidateProduct } from "@/lib/queries";
import { billingCatalogKey, competitorKey, credentialsKey, runResultsKey } from "@/lib/query-keys";
import { setupProgress } from "@/lib/setup-steps";
import { safeSourceURL } from "@/lib/source-url";
import { engines, type VisibilityScope } from "@/lib/visibility";
import { EngineSwitch } from "../prompts/-components/run-evidence";
import { PromptManager } from "./prompt-manager";
import { TrackingStart } from "./tracking-start";

const DRAFT_POLL_MS = 3_000;
const OUTAGE = "cloro_engine_unavailable";
const VISIBLE_SOURCES = 6;
const VISIBLE_COMPETITORS = 8;
const MemoPromptManager = memo(PromptManager);

type FirstLookBrand = { id: string; name: string; domain: string; githubRepo: string };
type Selection = { promptId: string; engine: VisibilityEngine };

/** The trial's real price, so the offer never hides what comes after the free days. */
function useEntryPrice() {
  const { i18n } = useTranslation();
  const api = useApi();
  const catalog = useQuery({
    queryKey: billingCatalogKey,
    queryFn: ({ signal }) => api.billing.catalog({ signal }),
  });
  const cheapest = catalog.data?.options[0];
  if (!cheapest || !catalog.data) return null;

  const format = (cents: number) =>
    new Intl.NumberFormat(i18n.language, {
      style: "currency",
      currency: catalog.data.currency,
      minimumFractionDigits: 0,
      maximumFractionDigits: 2,
    }).format(cents / 100);

  return { today: format(0), monthly: format(cheapest.amountCents) };
}

function useCompetitorDomains(brandID: string, enabled: boolean) {
  const api = useApi();
  const competitors = useQuery({
    queryKey: [...competitorKey(brandID), "first-look"],
    queryFn: ({ signal }) =>
      api.competitors.list({ brandId: brandID, page: 1, pageSize: 10 }, { signal }),
    enabled,
  });

  return competitors.data?.items ?? [];
}

/** The lists a free check runs on. The full page edits them. */
function CompetitorSummary({ brandID }: { brandID: string }) {
  const { t } = useTranslation("home");
  const competitors = useCompetitorDomains(brandID, true);
  const visible = competitors.slice(0, VISIBLE_COMPETITORS);
  const hidden = competitors.length - visible.length;

  if (competitors.length === 0) return null;

  return (
    <div className="grid justify-items-start gap-3">
      <h2 className="text-section font-semibold">{t("firstLook.competitorsTitle")}</h2>
      <ul className="flex flex-wrap items-center gap-1.5">
        {visible.map((competitor) => (
          <li
            key={competitor.id}
            className="flex h-8 items-center gap-1.5 rounded-full bg-muted/50 ps-1 pe-3 text-sm"
          >
            <BrandIcon name={competitor.name} domain={competitor.domain} className="size-6" />
            {competitor.name}
          </li>
        ))}
        {hidden > 0 && (
          <li className="px-2 text-sm text-muted-foreground">
            {t("firstLook.competitorsMore", { count: hidden })}
          </li>
        )}
      </ul>
      <Link to="/competitors" className="text-sm underline underline-offset-4">
        {t("firstLook.competitorsReview")}
      </Link>
    </div>
  );
}

function PriceBlock({ brand, canHandOff }: { brand: string; canHandOff: boolean }) {
  const { t } = useTranslation("home");
  const track = useOnboardingTracker();
  const price = useEntryPrice();
  const { copy, copied, failed } = useCopy();

  return (
    <section aria-labelledby="first-look-price" className="grid gap-4">
      <div className="grid gap-1">
        <h2 id="first-look-price" className="text-title font-semibold tabular-nums">
          {price ? t("firstLook.priceToday", { price: price.today }) : t("firstLook.cta")}
        </h2>
        <p className="text-sm text-muted-foreground">
          {price ? t("firstLook.priceThen", { price: price.monthly }) : t("firstLook.priceFree")}
        </p>
      </div>
      <div className="grid gap-2">
        <Button
          size="lg"
          role="link"
          render={<Link to="/settings/billing" />}
          nativeButton={false}
          onClick={() => track("trial_clicked")}
        >
          {t("firstLook.cta")}
        </Button>
        {canHandOff && (
          <Button
            size="lg"
            variant="outline"
            onClick={() => {
              track("agent_handoff");
              void copy(agentHandoffPrompt(brand));
            }}
          >
            {copied ? t("firstLook.handoffCopied") : t("firstLook.handoff")}
          </Button>
        )}
      </div>
      {copied && (
        <output className="block text-sm text-muted-foreground">
          {t("firstLook.handoffHint")}{" "}
          <Link to="/settings/connections" className="underline underline-offset-4">
            {t("firstLook.handoffConnect")}
          </Link>
        </output>
      )}
      {failed && (
        <p role="alert" className="text-sm text-destructive">
          {t("firstLook.handoffError")}
        </p>
      )}
    </section>
  );
}

/** Every question and engine at a glance. Choosing one shows its answer beside the list. */
function Questions({
  run,
  results,
  selected,
  collecting,
  onSelect,
}: {
  run: VisibilityRunStatus;
  results: VisibilityResult[];
  selected?: string;
  collecting: boolean;
  onSelect: (promptId: string, answers: VisibilityResult[]) => void;
}) {
  const { t } = useTranslation("home");

  return (
    <section aria-labelledby="first-look-questions" className="grid gap-3">
      <div className="flex items-baseline justify-between gap-3">
        <h2 id="first-look-questions" className="text-section font-semibold">
          {t("firstLook.questions")}
        </h2>
        {collecting && (
          <output className="text-xs text-muted-foreground tabular-nums">
            {t("firstLook.progress", { completed: run.completedResults, count: run.totalResults })}
          </output>
        )}
      </div>
      <ul className="list-rows -mx-2 grid gap-0.5">
        {firstLookRows(results).map((row) => (
          <li key={row.id}>
            <button
              type="button"
              aria-pressed={selected === row.id}
              onClick={() => onSelect(row.id, row.answers)}
              className="grid w-full gap-2 rounded-control px-2 py-2 text-start hover:bg-signal-pill aria-pressed:bg-signal-pill focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            >
              <span className="line-clamp-2 text-sm">{row.text}</span>
              <span className="flex gap-1">
                {engines.map((engine) => (
                  <OutcomeMark
                    key={engine.id}
                    engine={engine.id}
                    emphasize
                    className="size-5 transition-colors duration-200"
                    state={getAnswerState(
                      run,
                      engine.id,
                      row.answers.find(
                        (answer) => answer.engine === engine.id && answer.errorCode !== OUTAGE,
                      ),
                    )}
                  />
                ))}
              </span>
            </button>
          </li>
        ))}
      </ul>
      {collecting && <p className="text-xs text-muted-foreground">{t("firstLook.leave")}</p>}
    </section>
  );
}

/** Reflects what the draft understood, in the time the customer would otherwise only wait. */
function Mirror({ brandID, draftDone }: { brandID: string; draftDone: boolean }) {
  const { t } = useTranslation("home");
  const api = useApi();
  const draft = useQuery(generationOptions(api, brandID));
  const summary = draft.data?.items[0]?.summary;
  const names = useCompetitorDomains(brandID, draftDone)
    .slice(0, 4)
    .map((competitor) => competitor.name);

  if (!summary?.category) return null;

  return (
    <div className="grid max-w-[60ch] animate-in gap-3 fade-in duration-200">
      <h3 className="text-section font-semibold">{t("firstLook.mirrorTitle")}</h3>
      <dl className="grid gap-2 text-body sm:grid-cols-[8rem_minmax(0,1fr)]">
        <dt className="text-muted-foreground">{t("firstLook.mirrorCategory")}</dt>
        <dd>{summary.category}</dd>
        {summary.audience && (
          <>
            <dt className="text-muted-foreground">{t("firstLook.mirrorAudience")}</dt>
            <dd>{summary.audience}</dd>
          </>
        )}
        {names.length > 0 && (
          <>
            <dt className="animate-in text-muted-foreground fade-in duration-200">
              {t("firstLook.mirrorCompetitors")}
            </dt>
            <dd className="animate-in fade-in duration-200">{names.join(", ")}</dd>
          </>
        )}
      </dl>
    </div>
  );
}

function MarkedText({ text, brand, rivals }: { text: string; brand: string; rivals: string[] }) {
  return (
    <>
      {markAnswer(text, brand, rivals).map((segment, index) =>
        segment.mark ? (
          <mark
            // biome-ignore lint/suspicious/noArrayIndexKey: segments are positional and never reorder.
            key={index}
            data-mark={segment.mark}
            className={`rounded-sm px-0.5 font-semibold text-foreground ${segment.mark === "brand" ? "bg-positive/20" : "bg-negative/20"}`}
          >
            {segment.text}
          </mark>
        ) : (
          segment.text
        ),
      )}
    </>
  );
}

function Sources({
  sources,
  brandDomain,
  rivalDomains,
}: {
  sources: string[];
  brandDomain: string;
  rivalDomains: Set<string>;
}) {
  const { t } = useTranslation("home");
  const links = [...new Set(sources)]
    .map((source) => safeSourceURL(source))
    .filter((href): href is string => Boolean(href))
    .map((href) => ({ href, host: new URL(href).hostname.replace(/^www\./, "") }));

  if (links.length === 0) return null;

  const hidden = links.length - VISIBLE_SOURCES;

  return (
    <div className="grid gap-2">
      <h3 className="text-xs text-muted-foreground">{t("firstLook.sources")}</h3>
      <ul className="flex flex-wrap items-center gap-1.5">
        {links.slice(0, VISIBLE_SOURCES).map(({ href, host }) => (
          <li key={href}>
            <a
              href={href}
              target="_blank"
              rel="noopener noreferrer"
              title={href}
              className="flex h-8 max-w-56 items-center gap-1.5 rounded-full bg-muted/50 ps-1 pe-3 text-sm hover:bg-signal-pill focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            >
              <BrandIcon name={host} domain={host} className="size-6" />
              <span className="min-w-0 truncate">{host}</span>
              {host === brandDomain && (
                <span className="text-xs text-positive">{t("firstLook.sourceYou")}</span>
              )}
              {rivalDomains.has(host) && (
                <span className="text-xs text-negative">{t("firstLook.sourceCompetitor")}</span>
              )}
            </a>
          </li>
        ))}
        {hidden > 0 && (
          <li className="px-2 text-sm text-muted-foreground">
            {t("firstLook.sourcesMore", { count: hidden })}
          </li>
        )}
      </ul>
    </div>
  );
}

function SelectedAnswer({
  run,
  brand,
  answers,
  engine,
  rivalDomains,
  onEngine,
}: {
  run: VisibilityRunStatus;
  brand: FirstLookBrand;
  answers: VisibilityResult[];
  engine: VisibilityEngine;
  rivalDomains: Set<string>;
  onEngine: (engine: VisibilityEngine) => void;
}) {
  const { t } = useTranslation("home");
  const [expanded, setExpanded] = useState(false);
  const answer = answers.find((candidate) => candidate.engine === engine);
  const rivals = answer?.appearances.map((appearance) => appearance.name) ?? [];
  const { passage, trimmed } = answerPassage(answer?.rawAnswer ?? "", brand.name, rivals);
  const answered = Boolean(answer?.rawAnswer);

  return (
    <div className="grid min-w-0 gap-4">
      <EngineSwitch
        emphasize
        run={run}
        results={answers.filter((candidate) => candidate.errorCode !== OUTAGE)}
        selected={engine}
        onSelect={(next) => {
          setExpanded(false);
          onEngine(next);
        }}
      />
      <p className="text-section font-semibold">{answers[0]?.promptText}</p>
      {answered ? (
        <div
          key={answer?.id}
          className="grid animate-in justify-items-start gap-2 fade-in duration-200"
        >
          <p className="max-w-[68ch] whitespace-pre-line break-words text-body leading-7">
            <MarkedText
              text={expanded ? (answer?.rawAnswer ?? "") : passage}
              brand={brand.name}
              rivals={rivals}
            />
          </p>
          {trimmed && (
            <Button
              variant="link"
              className="-ms-2.5"
              aria-expanded={expanded}
              onClick={() => setExpanded(!expanded)}
            >
              {t(expanded ? "firstLook.showLess" : "firstLook.showFull")}
            </Button>
          )}
        </div>
      ) : (
        <p className="text-body text-muted-foreground">
          {t(answer?.outcome ? "firstLook.answerMissing" : "firstLook.answerWaiting")}
        </p>
      )}
      <Sources
        sources={answer?.sources ?? []}
        brandDomain={brand.domain}
        rivalDomains={rivalDomains}
      />
    </div>
  );
}

function Plan({ steps }: { steps: FirstLookStep[] }) {
  const { t } = useTranslation("home");

  return (
    <div className="grid gap-3">
      <h3 className="text-section font-semibold">{t("firstLook.planTitle")}</h3>
      <ol className="grid max-w-[64ch] gap-2 text-body">
        {steps.map((step, index) => (
          <li key={step.kind} className="grid grid-cols-[1.25rem_minmax(0,1fr)] gap-2">
            <span className="text-muted-foreground tabular-nums">{index + 1}</span>
            <span>
              {step.kind === "rival"
                ? t("firstLook.planRival", {
                    competitor: step.competitor,
                    count: step.count,
                    total: step.total,
                  })
                : step.kind === "source"
                  ? t("firstLook.planSource", { source: step.source, count: step.count })
                  : t("firstLook.planPrompt", { prompt: step.prompt, count: step.count })}
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
}

function Headline({
  verdict,
  brand,
  rivalDomain,
  engineName,
}: {
  verdict: FirstLookVerdict;
  brand: string;
  rivalDomain?: string;
  engineName: string;
}) {
  const { t } = useTranslation("home");

  return (
    <div key={verdict.result.id} className="flex animate-in items-start gap-3 fade-in duration-200">
      {verdict.competitor && (
        <BrandIcon name={verdict.competitor} domain={rivalDomain} className="mt-0.5 size-9" />
      )}
      <h1 className="max-w-[26ch] text-title font-semibold">
        {t(`firstLook.${verdict.kind}`, {
          engine: engineName,
          competitor: verdict.competitor,
          brand,
        })}
      </h1>
    </div>
  );
}

/** The free check, from the first draft to the first real answer. The answer is the page. */
export function FirstLook({
  brand,
  scope,
  draftStartFailed,
}: {
  brand: FirstLookBrand;
  scope: VisibilityScope;
  draftStartFailed: boolean;
}) {
  const { t } = useTranslation("home");
  const visibility = useTranslation("visibility");
  const api = useApi();
  const queryClient = useQueryClient();
  const tracking = useTracking(brand.id);
  const { collecting, latest } = useCollection(brand.id);

  const draftQuery = useQuery({
    ...generationOptions(api, brand.id),
    refetchInterval: (query) => {
      const state = query.state.data?.items[0]?.state;

      return !state || state === "queued" || state === "running" ? DRAFT_POLL_MS : false;
    },
  });
  const draft = draftQuery.data?.items[0];
  const draftState = draft?.state;
  // Only setup asks for this. An agent may still be correcting prompts, and it starts by itself.
  const autoStart = Boolean(draft?.startTracking);

  const start = useMutation({
    mutationFn: () => api.runs.startTracking(brand.id),
    onSuccess: () => invalidateProduct(queryClient, brand.id, "competitors"),
  });
  const startMutate = start.mutate;
  const started = useRef(false);

  // The worker starts the check, so a closed page cannot stop it. This covers a failed start,
  // and repeating it changes nothing.
  useEffect(() => {
    if (!autoStart || draftState !== "succeeded" || tracking.data?.confirmed !== false) return;
    if (started.current) return;

    started.current = true;
    startMutate();
  }, [autoStart, draftState, startMutate, tracking.data?.confirmed]);

  const run = latest.data?.items[0];
  const completed = run?.completedResults ?? 0;

  const results = useQuery({
    queryKey: [...runResultsKey(brand.id, run?.id), completed],
    queryFn: ({ signal }) => api.runs.listResults(run?.id ?? "", 1, 40, { signal }),
    enabled: Boolean(run),
    placeholderData: keepPreviousData,
  });

  const items = results.data?.items ?? [];
  const held = useRef<FirstLookVerdict | null>(null);
  held.current = keepVerdict(held.current, firstLookVerdict(items));
  const verdict = held.current;

  const [selection, setSelection] = useState<Selection | null>(null);
  const competitors = useCompetitorDomains(brand.id, Boolean(run));
  const rivalDomains = new Set(
    competitors.map((competitor) => competitor.domain).filter((domain) => Boolean(domain)),
  );
  const rivalDomain = competitors.find(
    (competitor) => competitor.name.toLowerCase() === verdict?.competitor?.toLowerCase(),
  )?.domain;
  const tint = useBrandColor(rivalDomain || brand.domain || undefined);

  const engineName = (engine: VisibilityEngine) => visibility.t(`engines.${engine}`);
  const site = brand.domain || brand.githubRepo;
  const score = firstLookScore(items);
  const unanswered = items.filter((result) => result.outcome === "error").length;
  const offline = engines.filter((engine) =>
    items.some((result) => result.engine === engine.id && result.errorCode === OUTAGE),
  );

  useOnboardingStep("first_look");
  useOnboardingStep(verdict ? "first_answer" : null);

  const promptId = selection?.promptId ?? verdict?.result.promptId;
  const engine = selection?.engine ?? verdict?.engine;
  const answers = items.filter((result) => result.promptId === promptId);

  const credentials = useQuery({
    queryKey: [...credentialsKey, "first-look"],
    queryFn: () => api.credentials.list(),
  });
  const blockedReason = tracking.data?.blockedReason;
  const blocked = blockedReason === "first_look_site_used" || blockedReason === "first_look_paused";
  const drafting = draftState === "queued" || draftState === "running";
  const startFailed = draftStartFailed || draftState === "failed" || start.isError;
  const checking = autoStart && !startFailed && (drafting || draftState === "succeeded");
  const reviewing = !tracking.isPending && tracking.data?.confirmed === false && !checking;

  // Nothing has been asked yet while someone reviews, so the result step hasn't begun.
  const progress = setupProgress(reviewing ? "agent_prompt" : "first_look", {
    canChoose: true,
    cloud: true,
  });

  let main: ReactNode;

  if (tracking.isPending) {
    main = <output className="block text-section font-semibold">{t("dashboard.loading")}</output>;
  } else if (tracking.data?.confirmed === false) {
    main = checking ? (
      <div className="grid gap-8">
        <div className="grid gap-1">
          <output className="block text-title font-semibold">
            {draftState === "succeeded"
              ? t("firstLook.starting")
              : t("firstLook.reading", { site })}
          </output>
          <p className="max-w-[48ch] text-body text-muted-foreground">
            {t("firstLook.readingDescription")}
          </p>
        </div>
        <Mirror brandID={brand.id} draftDone={draftState === "succeeded"} />
      </div>
    ) : (
      <div className="grid gap-8">
        <TrackingStart
          brandID={brand.id}
          scope={scope}
          draftStartFailed={draftStartFailed}
          description={t(
            credentials.data?.items.length ? "firstLook.reviewAgent" : "firstLook.reviewManual",
          )}
          actionLabel={t("firstLook.reviewAction")}
          showCompetitorLink={false}
        />
        <CompetitorSummary brandID={brand.id} />
      </div>
    );
  } else if (blocked && !run) {
    const notice = blockedReason === "first_look_paused" ? "paused" : "siteUsed";

    main = (
      <div className="grid gap-8">
        <div className="grid gap-1">
          <h1 className="text-title font-semibold">{t(`firstLook.${notice}Title`)}</h1>
          <p className="max-w-[48ch] text-body text-muted-foreground">
            {t(`firstLook.${notice}Description`)}
          </p>
        </div>
        <CompetitorSummary brandID={brand.id} />
      </div>
    );
  } else if (!run) {
    main = <output className="block text-title font-semibold">{t("firstLook.starting")}</output>;
  } else if (!verdict && !collecting && !results.isFetching) {
    main = (
      <div className="grid gap-1">
        <h1 className="text-title font-semibold">{t("firstLook.failed")}</h1>
        <p className="max-w-[48ch] text-body text-muted-foreground">
          {t("firstLook.failedDescription")}
        </p>
      </div>
    );
  } else if (!verdict) {
    main = (
      <div className="grid gap-8">
        <div className="grid gap-1">
          <output className="block text-title font-semibold">{t("firstLook.asking")}</output>
          <p className="text-body text-muted-foreground">{t("firstLook.scoreWaiting")}</p>
        </div>
        <Mirror brandID={brand.id} draftDone />
      </div>
    );
  } else {
    const steps = collecting ? [] : firstLookPlan(items, brand.domain);

    main = (
      <div className="grid gap-8">
        <div className="grid gap-3">
          <output className="block text-sm text-muted-foreground tabular-nums">
            <span className="sr-only">
              {t("firstLook.score", { seen: score.seen, count: score.total, brand: brand.name })}
            </span>
            <span aria-hidden="true">
              <NumberFlow
                value={score.seen}
                className={`font-semibold ${score.seen > 0 ? "text-positive" : "text-negative"}`}
              />{" "}
              {t("firstLook.scoreOf", { count: score.total, brand: brand.name })}
            </span>
          </output>
          <Headline
            verdict={verdict}
            brand={brand.name}
            rivalDomain={rivalDomain}
            engineName={engineName(verdict.engine)}
          />
        </div>
        {engine && (
          <SelectedAnswer
            key={promptId}
            run={run}
            brand={brand}
            answers={answers}
            engine={engine}
            rivalDomains={rivalDomains}
            onEngine={(next) => promptId && setSelection({ promptId, engine: next })}
          />
        )}
        {steps.length > 0 && <Plan steps={steps} />}
        {offline.length > 0 && (
          <p className="text-sm text-muted-foreground">
            {t("firstLook.engineOffline", {
              engines: offline.map((entry) => entry.name).join(", "),
            })}
          </p>
        )}
        {!collecting && offline.length === 0 && unanswered > 0 && (
          <p className="text-sm text-muted-foreground">
            {t("firstLook.unanswered", { count: unanswered })}
          </p>
        )}
      </div>
    );
  }

  return (
    <div className="grid gap-4 pb-24">
      {!(blocked && !run) && (
        <div className="justify-self-end">
          <SetupProgress current={progress.current} total={progress.total} />
        </div>
      )}
      <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
        <section
          className="min-w-0 rounded-panel bg-surface p-4 transition-colors sm:px-11 sm:py-9"
          style={brandGlow(tint)}
        >
          {main}
        </section>
        <aside className="grid min-w-0 gap-8 rounded-panel bg-surface p-4 sm:p-6 lg:sticky lg:top-6">
          <PriceBlock brand={brand.name} canHandOff={Boolean(verdict)} />
          {run && items.length > 0 && (
            <Questions
              run={run}
              results={items}
              selected={promptId}
              collecting={collecting}
              onSelect={(id, rowAnswers) =>
                setSelection({
                  promptId: id,
                  engine: firstLookVerdict(rowAnswers)?.engine ?? engine ?? "chatgpt",
                })
              }
            />
          )}
          {run && !collecting && (
            <Link to="/competitors" className="text-sm underline underline-offset-4">
              {t("firstLook.seeCompetitors")}
            </Link>
          )}
        </aside>
      </div>
      {(reviewing || (blocked && !run)) && (
        <MemoPromptManager brandID={brand.id} scope={scope} collecting={false} />
      )}
    </div>
  );
}
