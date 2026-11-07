import { Tick02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import NumberFlow from "@number-flow/react";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { memo, type ReactNode, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import type { VisibilityEngine, VisibilityResult, VisibilityRunStatus } from "@/api/generated";
import type { Generation } from "@/api/generations.api";
import { BrandIcon, useBrandColor } from "@/components/blocks/brand-icon";
import { MentionNote } from "@/components/blocks/mention-note";
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
  displaySite,
  type FirstLookStep,
  type FirstLookVerdict,
  firstLookPlan,
  firstLookRows,
  firstLookScore,
  firstLookVerdict,
  firstLookWait,
  keepVerdict,
  recognitionGap,
  type WaitState,
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
// The draft alone took about 30 seconds when measured, so twice that is worth saying out loud.
const SLOW_WAIT_MS = 60_000;
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

function CompetitorChips({ brandID, total }: { brandID: string; total?: number }) {
  const { t } = useTranslation("home");
  const competitors = useCompetitorDomains(brandID, true);
  const visible = competitors.slice(0, VISIBLE_COMPETITORS);
  const hidden = Math.max(total ?? 0, competitors.length) - visible.length;

  if (competitors.length === 0) return null;

  return (
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
  );
}

/** The lists a free check runs on. The full page edits them. */
function CompetitorSummary({ brandID }: { brandID: string }) {
  const { t } = useTranslation("home");
  const competitors = useCompetitorDomains(brandID, true);

  if (competitors.length === 0) return null;

  return (
    <div className="grid justify-items-start gap-3">
      <h2 className="text-section font-semibold">{t("firstLook.competitorsTitle")}</h2>
      <CompetitorChips brandID={brandID} />
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

function WaitMark({ state }: { state: WaitState }) {
  if (state === "done") {
    return <HugeiconsIcon icon={Tick02Icon} className="size-4 text-muted-foreground" />;
  }

  if (state === "active") {
    return (
      <span className="size-3.5 animate-spin rounded-full border-2 border-muted border-t-foreground motion-reduce:animate-pulse" />
    );
  }

  return <span className="size-1.5 rounded-full bg-muted-foreground/50" />;
}

/** A finished step keeps what it found, and the steps below make room for it. */
function WaitRow({
  state,
  label,
  children,
}: {
  state: WaitState;
  label: string;
  children?: ReactNode;
}) {
  return (
    <li className="grid grid-cols-[1.25rem_minmax(0,1fr)] gap-x-2.5">
      <span aria-hidden="true" className="grid h-5 place-items-center">
        <WaitMark state={state} />
      </span>
      <p
        className={`text-body leading-5 break-words ${state === "active" ? "font-medium" : "text-muted-foreground"}`}
      >
        {label}
      </p>
      {children && (
        <div className="col-start-2 grid grid-rows-[1fr] transition-[grid-template-rows] duration-200 ease-out starting:grid-rows-[0fr] motion-reduce:transition-none">
          <div className="min-h-0 overflow-hidden">
            <div className="animate-in pt-2.5 fade-in duration-200">{children}</div>
          </div>
        </div>
      )}
    </li>
  );
}

/** Says so only when the current step has really run long. A new step starts the clock again. */
function useSlowWait(since: Date | undefined) {
  const [slowSince, setSlowSince] = useState<number | null>(null);
  const start = since?.getTime();

  useEffect(() => {
    if (start === undefined) return;

    const timer = setTimeout(() => setSlowSince(start), start + SLOW_WAIT_MS - Date.now());

    return () => clearTimeout(timer);
  }, [start]);

  return start !== undefined && slowSince === start;
}

/** From the first draft to the first answer: every finished step shows what it found. */
function FirstLookWaiting({
  brand,
  draft,
  run,
}: {
  brand: FirstLookBrand;
  draft?: Generation;
  run?: VisibilityRunStatus;
}) {
  const { t, i18n } = useTranslation("home");
  const steps = firstLookWait(draft);
  const summary = draft?.summary;
  const slow = useSlowWait(run?.createdAt ?? draft?.createdAt);
  const site = displaySite(brand.domain || brand.githubRepo);
  const engineNames = new Intl.ListFormat(i18n.language, { type: "conjunction" }).format(
    engines.map((engine) => engine.name),
  );
  const asking = t("firstLook.waitAsking", { engines: engineNames });
  const drafted = draft?.promptCount
    ? t("firstLook.waitDrafted", { count: draft.promptCount })
    : t("firstLook.waitDraftReady");
  const current =
    steps.read === "active"
      ? t("firstLook.waitReading", { site })
      : steps.draft === "active"
        ? t("firstLook.waitDrafting")
        : asking;

  return (
    <div className="grid gap-8">
      <div className="grid gap-1">
        <h1 className="text-title font-semibold">
          {t("firstLook.waitTitle", { brand: brand.name })}
        </h1>
        <p className="max-w-[56ch] text-body text-muted-foreground">
          {t(slow ? "firstLook.waitSlow" : "firstLook.waitDescription")}
        </p>
      </div>
      <output className="sr-only">{current}</output>
      <ol className="grid gap-4">
        {steps.read && (
          <WaitRow
            state={steps.read}
            label={t(steps.read === "done" ? "firstLook.waitRead" : "firstLook.waitReading", {
              site,
            })}
          >
            {summary?.category && (
              <dl className="grid gap-2 text-body sm:grid-cols-[8rem_minmax(0,1fr)]">
                <dt className="text-muted-foreground">{t("firstLook.mirrorCategory")}</dt>
                <dd>{summary.category}</dd>
                {summary.audience && (
                  <>
                    <dt className="text-muted-foreground">{t("firstLook.mirrorAudience")}</dt>
                    <dd>{summary.audience}</dd>
                  </>
                )}
              </dl>
            )}
          </WaitRow>
        )}
        {steps.draft && (
          <WaitRow
            state={steps.draft}
            label={steps.draft === "done" ? drafted : t("firstLook.waitDrafting")}
          >
            {steps.draft === "done" && draft && draft.competitorCount > 0 && (
              <dl className="grid gap-2 text-body sm:grid-cols-[8rem_minmax(0,1fr)]">
                <dt className="text-muted-foreground sm:leading-8">
                  {t("firstLook.competitorsTitle")}
                </dt>
                <dd>
                  <CompetitorChips brandID={brand.id} total={draft.competitorCount} />
                </dd>
              </dl>
            )}
          </WaitRow>
        )}
        <WaitRow state={steps.ask} label={asking}>
          {run && (
            <p className="text-sm text-muted-foreground tabular-nums">
              {t("firstLook.progress", {
                completed: run.completedResults,
                count: run.totalResults,
              })}
            </p>
          )}
        </WaitRow>
      </ol>
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
  // When the engine doesn't recognize the brand, the proof is the passage about the brand.
  const proofRivals = answer && recognitionGap(answer, brand.name) ? [] : rivals;
  const { passage, trimmed } = answerPassage(answer?.rawAnswer ?? "", brand.name, proofRivals);
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
          <MentionNote verdict={answer?.mentionVerdict} brand={brand.name} />
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
        {t(
          verdict.kind === "unknown" && verdict.competitor
            ? "firstLook.unknownRival"
            : `firstLook.${verdict.kind}`,
          {
            engine: engineName,
            competitor: verdict.competitor,
            brand,
          },
        )}
      </h1>
    </div>
  );
}

/** The free check, from the first draft to the first real answer. The answer is the page. */
export function FirstLook({
  brand,
  scope,
  draftStartFailed,
  draftUnavailable,
}: {
  brand: FirstLookBrand;
  scope: VisibilityScope;
  draftStartFailed: boolean;
  draftUnavailable: boolean;
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
  held.current = keepVerdict(held.current, firstLookVerdict(items, brand.name));
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
  let waiting = false;

  if (tracking.isPending) {
    waiting = true;
    main = <output className="block text-section font-semibold">{t("dashboard.loading")}</output>;
  } else if (tracking.data?.confirmed === false) {
    waiting = checking;
    main = checking ? (
      <FirstLookWaiting brand={brand} draft={draft} />
    ) : (
      <div className="grid gap-8">
        {draftUnavailable && (
          <div className="grid gap-1">
            <h1 className="text-title font-semibold">{t("firstLook.draftUnavailableTitle")}</h1>
            <p className="max-w-[48ch] text-body text-muted-foreground">
              {t("firstLook.draftUnavailableDescription")}
            </p>
          </div>
        )}
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
    waiting = true;
    main = <FirstLookWaiting brand={brand} draft={draft} />;
  } else if (!verdict && !collecting && !results.isFetching) {
    main = (
      <div className="grid gap-1">
        <h1 className="text-title font-semibold">{t("firstLook.failed")}</h1>
        <p className="max-w-[48ch] text-body text-muted-foreground">
          {t(
            blockedReason === "first_look_used"
              ? "firstLook.failedDescription"
              : "firstLook.failedRetryDescription",
          )}
        </p>
      </div>
    );
  } else if (!verdict) {
    waiting = true;
    main = <FirstLookWaiting brand={brand} draft={draft} run={run} />;
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
    <div className={`grid gap-4 pb-24 ${waiting ? "mx-auto w-full max-w-3xl" : ""}`}>
      {!(blocked && !run) && (
        <div className="justify-self-end">
          <SetupProgress current={progress.current} total={progress.total} />
        </div>
      )}
      <div
        className={`grid items-start gap-6 ${waiting ? "" : "lg:grid-cols-[minmax(0,1fr)_22rem]"}`}
      >
        <section
          className="min-w-0 rounded-panel bg-surface p-4 transition-colors sm:px-11 sm:py-9"
          style={brandGlow(tint)}
        >
          {main}
        </section>
        {/* The offer waits for something worth keeping an eye on. */}
        {!waiting && (
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
                    engine: firstLookVerdict(rowAnswers, brand.name)?.engine ?? engine ?? "chatgpt",
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
        )}
      </div>
      {(reviewing || (blocked && !run)) && (
        <MemoPromptManager brandID={brand.id} scope={scope} collecting={false} />
      )}
    </div>
  );
}
