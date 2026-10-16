import {
  ChatGptIcon,
  GoogleGeminiIcon,
  GoogleIcon,
  InformationCircleIcon,
  PerplexityAiIcon,
  Settings01Icon,
  Tick02Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import NumberFlow, { useCanAnimate } from "@number-flow/react";
import { useCallback, useEffect, useId, useState } from "react";
import type { VisibilityEngine } from "@/api/generated";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { ENGINES } from "../-lib/data";
import type { PreviewBrand } from "../-lib/exploration-fixtures";
import type { HomeBrand } from "../-lib/home-evidence";
import {
  type ComparedBrand,
  comparedBrands,
  completed,
  dailySeries,
  type EvidenceRecord,
  gapChange,
  inScope,
  percent,
  points,
  summarize,
  type VisibilityScope,
} from "../-lib/prompt-visibility";
import C from "../-locales/prompt-visibility.en.json";
import { copy } from "./list-controls";
import { formatSignalDate, SignalChart, type SignalInspectionInput } from "./signal-chart";

const ENGINE_ICONS = {
  chatgpt: ChatGptIcon,
  perplexity: PerplexityAiIcon,
  gemini: GoogleGeminiIcon,
  google_ai_overviews: GoogleIcon,
};

function signedPoints(value: number | null) {
  if (value === null) return "—";

  const rounded = Number(value.toFixed(1));
  return copy(C.shortPoints, {
    points: `${rounded > 0 ? "+" : rounded < 0 ? "−" : ""}${points(rounded)}`,
  });
}

export function HomeSignal({
  brand,
  data,
  records,
  scope,
  compareId,
  onCompare,
  onDaysChange,
  onEnginesChange,
  onManage,
}: {
  brand: HomeBrand;
  data: PreviewBrand;
  records: EvidenceRecord[];
  scope: VisibilityScope;
  compareId?: string;
  onCompare: (id?: string) => void;
  onDaysChange: (days: 7 | 30 | 90) => void;
  onEnginesChange: (engines: VisibilityEngine[]) => void;
  onManage: () => void;
}) {
  const titleId = useId();
  const [input, setInput] = useState<SignalInspectionInput>("keyboard");
  const [inspection, setInspection] = useState<{
    scopeKey: string;
    index: number;
    input: SignalInspectionInput;
  } | null>(null);
  const canAnimateNumbers = useCanAnimate();
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [detailsPinned, setDetailsPinned] = useState(false);
  const brands = comparedBrands(brand, data);
  const you = brands[0];
  const stats = summarize(records, brand, scope, you);
  const rivals = brands
    .slice(1)
    .map((item) => ({ ...item, stats: summarize(records, brand, scope, item) }))
    .sort((a, b) => (b.stats.rate ?? -1) - (a.stats.rate ?? -1) || a.name.localeCompare(b.name));
  const selected = rivals.find((item) => item.id === compareId && item.stats.rate !== null);
  const series = dailySeries(records, brand, scope, you);
  const comparison = selected ? dailySeries(records, brand, scope, selected) : undefined;
  const excluded = inScope(records, scope).filter((record) => !completed(record)).length;
  const scopeKey = `${scope.days}:${scope.engines?.join(",") ?? ""}:${selected?.id ?? ""}`;
  const activeInspection = inspection?.scopeKey === scopeKey ? inspection : null;
  const selectedPoint = activeInspection ? series[activeInspection.index] : null;
  const selectedComparisonPoint =
    activeInspection && comparison ? comparison[activeInspection.index] : null;
  const showComparisonReadout = !!selectedPoint && !!selected && !!selectedComparisonPoint;
  const headlineRate = selectedPoint ? selectedPoint.rate : stats.rate;
  const rawChange = selectedPoint ? selectedPoint.changePoints : stats.changePoints;
  const change = rawChange === null ? null : Number(rawChange.toFixed(1));
  const animateNumbers = canAnimateNumbers && input === "pointer";

  useEffect(() => {
    setInspection((current) => (current?.scopeKey === scopeKey ? current : null));
  }, [scopeKey]);

  const onSelectedIndexChange = useCallback(
    (index: number | null, source: SignalInspectionInput) => {
      setInput(source);
      if (index !== null) {
        setDetailsOpen(false);
        setDetailsPinned(false);
      }
      setInspection(index === null ? null : { scopeKey, index, input: source });
    },
    [scopeKey],
  );

  return (
    <section
      className="hc-signal"
      aria-labelledby={titleId}
      data-input={input}
      onPointerDownCapture={() => setInput("pointer")}
      onKeyDownCapture={() => setInput("keyboard")}
    >
      <h2 id={titleId} className="sr-only">
        {C.heading}
      </h2>
      <TooltipProvider delay={350}>
        <div className="hc-signal-header">
          <div className="hc-signal-metric">
            <div className="hc-signal-reading">
              {showComparisonReadout && selected && selectedPoint && selectedComparisonPoint ? (
                <div className="hc-signal-comparison-reading">
                  <SignalRate brand={you} rate={selectedPoint.rate} animated={animateNumbers} />
                  <SignalRate
                    brand={selected}
                    rate={selectedComparisonPoint.rate}
                    animated={animateNumbers}
                  />
                </div>
              ) : (
                <>
                  <div className="hc-rate">
                    {headlineRate === null ? (
                      "—"
                    ) : (
                      <NumberFlow
                        className="hc-signal-number"
                        value={Number(headlineRate.toFixed(1))}
                        format={{ maximumFractionDigits: 1 }}
                        animated={animateNumbers}
                      />
                    )}
                    {headlineRate !== null && <span>{C.percentSign}</span>}
                  </div>
                  <span
                    className="hc-signal-delta"
                    data-direction={change ? (change > 0 ? "positive" : "negative") : undefined}
                  >
                    <span aria-hidden="true">
                      {change === null ? (
                        "—"
                      ) : (
                        <NumberFlow
                          className="hc-signal-number"
                          value={Math.abs(change)}
                          prefix={change > 0 ? "+" : change < 0 ? "−" : undefined}
                          format={{ maximumFractionDigits: 1 }}
                          animated={animateNumbers}
                        />
                      )}
                    </span>
                    <span className="sr-only">
                      {change === null
                        ? activeInspection
                          ? C.noDailyComparison
                          : C.noComparison
                        : copy(change > 0 ? C.up : change < 0 ? C.down : C.unchanged, {
                            points: points(change),
                          })}
                    </span>
                  </span>
                </>
              )}
              {selectedPoint && (
                <time className="hc-selected-date" dateTime={selectedPoint.date}>
                  {formatSignalDate(selectedPoint.date)}
                </time>
              )}
            </div>
            <Tooltip
              open={detailsOpen}
              onOpenChange={(open, details) => {
                if (
                  detailsPinned &&
                  (details.reason === "trigger-hover" || details.reason === "trigger-focus")
                )
                  return;
                setDetailsOpen(open);
                if (!open) setDetailsPinned(false);
              }}
            >
              <TooltipTrigger
                render={<Button variant="ghost" size="icon-lg" className="hc-signal-info" />}
                aria-label={C.metricDetails}
                aria-expanded={detailsOpen}
                closeOnClick={false}
                onClick={() => {
                  setDetailsPinned(!detailsPinned);
                  setDetailsOpen(!detailsPinned);
                }}
              >
                <HugeiconsIcon icon={InformationCircleIcon} aria-hidden="true" />
              </TooltipTrigger>
              <TooltipContent
                className="hc-signal-tooltip"
                data-input={input}
                side="bottom"
                align="start"
              >
                <div className="hc-metric-details">
                  <strong>
                    {selectedPoint ? formatSignalDate(selectedPoint.date) : C.heading}
                  </strong>
                  <p>
                    {C.mentionedIn} <strong>{selectedPoint?.mentioned ?? stats.mentioned}</strong>{" "}
                    {C.of} <strong>{selectedPoint?.completed ?? stats.completed}</strong>{" "}
                    {C.completedSampleAnswers}
                  </p>
                  <p>
                    {selectedPoint ? (
                      selectedPoint.changePoints === null ? (
                        C.noDailyComparison
                      ) : (
                        <>
                          <strong>{signedPoints(selectedPoint.changePoints)}</strong>{" "}
                          {C.vsPreviousCompletedDataDay}
                        </>
                      )
                    ) : stats.previousRate === null ? (
                      C.noComparison
                    ) : (
                      <>
                        <strong>{percent(stats.previousRate)}</strong>{" "}
                        {copy(C.previousPeriodSuffix, { days: scope.days })}
                      </>
                    )}
                  </p>
                  {excluded > 0 && <p>{copy(C.collectionNote, { count: excluded })}</p>}
                  <p>{C.multiple}</p>
                </div>
              </TooltipContent>
            </Tooltip>
          </div>
          <fieldset className="hc-engine-filter" aria-label={C.byEngine}>
            {ENGINES.map((engine) => {
              const checked = scope.engines?.includes(engine.id) ?? false;

              return (
                <Tooltip key={engine.id}>
                  <TooltipTrigger
                    render={<button type="button" className="hc-engine-toggle" />}
                    aria-label={engine.name}
                    aria-pressed={checked}
                    onClick={() =>
                      onEnginesChange(
                        ENGINES.filter((item) =>
                          item.id === engine.id ? !checked : scope.engines?.includes(item.id),
                        ).map((item) => item.id),
                      )
                    }
                  >
                    <span className="hc-engine-disc">
                      <HugeiconsIcon icon={ENGINE_ICONS[engine.id]} size={24} aria-hidden="true" />
                      {checked && (
                        <HugeiconsIcon
                          icon={Tick02Icon}
                          className="hc-engine-check"
                          size={10}
                          aria-hidden="true"
                        />
                      )}
                    </span>
                  </TooltipTrigger>
                  <TooltipContent className="hc-signal-tooltip" data-input={input} side="bottom">
                    {engine.name}
                  </TooltipContent>
                </Tooltip>
              );
            })}
          </fieldset>
        </div>
        {stats.rate === null ? (
          <div className="hc-signal-empty">
            <p>{scope.promptIds.length ? C.noPeriodAnswers : C.noTrackedAnswers}</p>
            {!!scope.engines?.length && (
              <Button variant="ghost" onClick={() => onEnginesChange([])}>
                {C.clearEngines}
              </Button>
            )}
          </div>
        ) : (
          <SignalChart
            series={series}
            brand={you}
            comparison={comparison}
            competitor={selected}
            scopeKey={scopeKey}
            animate={input === "pointer"}
            selectedIndex={activeInspection?.index ?? null}
            inspectionInput={activeInspection?.input ?? null}
            onSelectedIndexChange={onSelectedIndexChange}
          />
        )}
        <div className="hc-signal-range">
          <span className="hc-signal-dates">
            <time dateTime={series[0].date}>{formatSignalDate(series[0].date)}</time>
            {" – "}
            <time dateTime={series[series.length - 1].date}>
              {formatSignalDate(series[series.length - 1].date)}
            </time>
          </span>
          <fieldset className="hc-periods hc-signal-periods" aria-label={C.chartRange}>
            {([7, 30, 90] as const).map((days) => (
              <button
                type="button"
                key={days}
                aria-pressed={scope.days === days}
                onClick={() => onDaysChange(days)}
              >
                {copy(C.periodLabel, { days })}
              </button>
            ))}
          </fieldset>
        </div>
        <div className="hc-signal-context">
          <div className="hc-compare-heading">
            <h3>{C.competitors}</h3>
            <Tooltip>
              <TooltipTrigger
                render={<Button variant="ghost" size="icon-lg" />}
                aria-label={C.manageCompetitors}
                onClick={onManage}
              >
                <HugeiconsIcon icon={Settings01Icon} aria-hidden="true" />
              </TooltipTrigger>
              <TooltipContent className="hc-signal-tooltip" data-input={input}>
                {C.manageCompetitors}
              </TooltipContent>
            </Tooltip>
          </div>
          {rivals.map((item) => {
            const difference =
              item.stats.rate !== null && stats.rate !== null ? item.stats.rate - stats.rate : null;
            const movement = gapChange(stats, item.stats);
            const gapText =
              difference === null
                ? C.noObservations
                : copy(
                    Math.abs(difference) < 0.05 ? C.level : difference > 0 ? C.ahead : C.behind,
                    { points: points(difference) },
                  );
            const movementText =
              movement === null
                ? C.noComparison
                : copy(
                    Math.abs(movement) < 0.05
                      ? C.gapUnchanged
                      : movement > 0
                        ? C.gaining
                        : C.losing,
                    { points: points(movement), name: item.name },
                  );
            const description = `${percent(item.stats.rate)}. ${gapText}. ${movementText}`;
            const available = item.stats.rate !== null;

            return (
              <Tooltip key={item.id}>
                <TooltipTrigger
                  render={<button type="button" className="hc-compare-row" />}
                  aria-label={copy(C.compare, { name: item.name })}
                  aria-description={description}
                  aria-disabled={!available}
                  aria-pressed={selected?.id === item.id}
                  onClick={() => {
                    if (available) onCompare(selected?.id === item.id ? undefined : item.id);
                  }}
                >
                  <BrandPill name={item.name} domain={item.domain} />
                  <span>{percent(item.stats.rate)}</span>
                  <span
                    className="hc-compare-gap"
                    data-direction={
                      difference !== null && Math.abs(difference) >= 0.05
                        ? difference > 0
                          ? "negative"
                          : "positive"
                        : undefined
                    }
                  >
                    {signedPoints(difference)}
                  </span>
                </TooltipTrigger>
                <TooltipContent className="hc-signal-tooltip" data-input={input}>
                  <div className="hc-metric-details">
                    <p>{gapText}</p>
                    <p>{movementText}</p>
                  </div>
                </TooltipContent>
              </Tooltip>
            );
          })}
          {!rivals.length && <p className="hc-caption">{C.noCompetitors}</p>}
        </div>
      </TooltipProvider>
    </section>
  );
}

function SignalRate({
  brand,
  rate,
  animated,
}: {
  brand: ComparedBrand;
  rate: number | null;
  animated: boolean;
}) {
  return (
    <span className="hc-signal-comparison-rate">
      <span className="sr-only">
        {brand.name}
        {": "}
      </span>
      <BrandIcon name={brand.name} domain={brand.domain} src={brand.src} className="size-6" />
      {rate === null ? (
        <span className="hc-signal-number" aria-hidden="true">
          {"—"}
        </span>
      ) : (
        <span className="hc-rate" aria-hidden="true">
          <NumberFlow
            className="hc-signal-number"
            value={Number(rate.toFixed(1))}
            format={{ maximumFractionDigits: 1 }}
            animated={animated}
          />
          <span>{C.percentSign}</span>
        </span>
      )}
    </span>
  );
}
