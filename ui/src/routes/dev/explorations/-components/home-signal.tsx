import { useId, useLayoutEffect, useRef, useState } from "react";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Button } from "@/components/ui/button";
import { ENGINES } from "../-lib/data";
import type { PreviewBrand } from "../-lib/exploration-fixtures";
import type { HomeBrand } from "../-lib/home-evidence";
import {
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
import { copy, LocalChoice } from "./list-controls";

export function HomeSignal({
  brand,
  data,
  records,
  scope,
  compareId,
  onCompare,
  onDaysChange,
  onEngineChange,
  onManage,
}: {
  brand: HomeBrand;
  data: PreviewBrand;
  records: EvidenceRecord[];
  scope: VisibilityScope;
  compareId?: string;
  onCompare: (id?: string) => void;
  onDaysChange: (days: 7 | 30 | 90) => void;
  onEngineChange: (engine?: string) => void;
  onManage: () => void;
}) {
  const titleId = useId();
  const [active, setActive] = useState<number | null>(null);
  const plot = useRef<SVGSVGElement | null>(null);
  const [plotSize, setPlotSize] = useState({ width: 1000, height: 240 });

  useLayoutEffect(() => {
    const element = plot.current;
    if (!element) return;

    const resize = () =>
      setPlotSize({
        width: Math.max(240, element.getBoundingClientRect().width),
        height: document.documentElement.clientWidth <= 640 ? 180 : 240,
      });
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(element);
    return () => observer.disconnect();
  }, [records, scope.promptIds.length]);
  const brands = comparedBrands(brand, data);
  const you = brands[0];
  const stats = summarize(records, brand, scope, you);
  const rivals = brands
    .slice(1)
    .map((item) => ({ ...item, stats: summarize(records, brand, scope, item) }))
    .sort((a, b) => (b.stats.rate ?? -1) - (a.stats.rate ?? -1) || a.name.localeCompare(b.name));
  const selected = rivals.find((item) => item.id === compareId && item.stats.rate !== null);
  const benchmark = selected ?? rivals.find((item) => item.stats.rate !== null);
  const series = dailySeries(records, brand, scope, you);
  const comparison = selected ? dailySeries(records, brand, scope, selected) : undefined;
  const index = Math.min(active ?? series.length - 1, series.length - 1);
  const current = series[index];
  const excluded = inScope(records, scope).filter((record) => !completed(record)).length;
  const gap =
    stats.rate !== null && benchmark?.stats.rate != null ? benchmark.stats.rate - stats.rate : null;
  const movement = benchmark ? gapChange(stats, benchmark.stats) : null;
  const previousGap =
    stats.previousRate !== null && benchmark?.stats.previousRate != null
      ? benchmark.stats.previousRate - stats.previousRate
      : null;
  const formatDate = (value: string) =>
    new Date(`${value}T12:00:00Z`).toLocaleDateString("en", {
      month: "short",
      day: "numeric",
      timeZone: "UTC",
    });
  const labelWidth = plotSize.width < 500 ? 64 : 144;
  const plotWidth = plotSize.width - 36 - labelWidth;
  const x = (i: number) => 36 + (i / Math.max(1, series.length - 1)) * plotWidth;
  const y = (value: number) => 16 + ((100 - value) / 100) * (plotSize.height - 32);
  const path = (values: typeof series) => {
    let connected = false;

    return values
      .map((point, i) => {
        if (point.rate === null) {
          connected = false;
          return "";
        }
        const segment = `${(connected ? "L" : "M") + x(i)},${y(point.rate)}`;
        connected = true;
        return segment;
      })
      .join(" ");
  };
  const last = series.reduce((found, point, i) => (point.rate !== null ? i : found), -1);
  const otherLast =
    comparison?.reduce((found, point, i) => (point.rate !== null ? i : found), -1) ?? -1;
  const readout = copy(C.point, {
    date: current.date,
    brand: you.name,
    rate: current.rate === null ? C.noData : percent(current.rate),
    count: current.completed,
  });
  const otherReadout =
    selected && comparison
      ? copy(C.point, {
          date: current.date,
          brand: selected.name,
          rate: comparison[index].rate === null ? C.noData : percent(comparison[index].rate),
          count: comparison[index].completed,
        })
      : "";
  const changeText =
    stats.changePoints === null
      ? C.noComparison
      : Math.abs(stats.changePoints) < 0.05
        ? C.unchanged
        : copy(stats.changePoints > 0 ? C.up : C.down, { points: points(stats.changePoints) });

  return (
    <section className="hc-signal" aria-labelledby={titleId}>
      <div className="hc-section-heading">
        <h2 id={titleId}>{C.heading}</h2>
        <LocalChoice
          label={C.byEngine}
          value={scope.engine ?? "all"}
          options={[
            { value: "all", label: C.allEngines },
            ...ENGINES.map((item) => ({ value: item.id, label: item.name })),
          ]}
          onChange={(value) => onEngineChange(value === "all" ? undefined : value)}
        />
      </div>
      <div className="hc-signal-metrics">
        <div>
          <div className="hc-rate">{percent(stats.rate)}</div>
          <p>{copy(C.period, { days: scope.days })}</p>
        </div>
        <div className="hc-signal-change">
          <strong>{changeText}</strong>
          <p>{copy(C.previousPeriod, { days: scope.days })}</p>
        </div>
      </div>
      <p className="hc-signal-count">
        {stats.completed
          ? copy(C.sampleCount, { mentioned: stats.mentioned, completed: stats.completed })
          : C.noAnswers}
      </p>
      {stats.rate === null ? (
        <p className="hc-empty">{C.noTrackedAnswers}</p>
      ) : (
        <figure className="hc-trend">
          <svg
            ref={plot}
            viewBox={`0 0 ${plotSize.width} ${plotSize.height}`}
            preserveAspectRatio="none"
            role="img"
            aria-label={C.chartDescription}
            onPointerMove={(event) => {
              const bounds = event.currentTarget.getBoundingClientRect();
              setActive(
                Math.max(
                  0,
                  Math.min(
                    series.length - 1,
                    Math.round(
                      ((event.clientX - bounds.left - 36) / plotWidth) * (series.length - 1),
                    ),
                  ),
                ),
              );
            }}
            onPointerLeave={() => setActive(null)}
          >
            {[0, 50, 100].map((value) => (
              <text key={value} x="0" y={y(value) + 4} className="hc-trend-axis">
                {percent(value)}
              </text>
            ))}
            <path d={path(series)} className="hc-trend-line" />
            {comparison && (
              <path d={path(comparison)} className="hc-trend-line hc-trend-comparison" />
            )}
            {series.map((point, i) =>
              point.rate !== null &&
              (i === 0 || series[i - 1].rate === null) &&
              (i === series.length - 1 || series[i + 1].rate === null) ? (
                <circle
                  key={point.day}
                  cx={x(i)}
                  cy={y(point.rate)}
                  r="3"
                  className="hc-trend-dot"
                />
              ) : null,
            )}
            {last >= 0 && (
              <circle cx={x(last)} cy={y(series[last].rate ?? 0)} r="4" className="hc-trend-dot" />
            )}
            {comparison && otherLast >= 0 && (
              <circle
                cx={x(otherLast)}
                cy={y(comparison[otherLast].rate ?? 0)}
                r="3"
                className="hc-trend-other-dot"
              />
            )}
            {active !== null && (
              <line
                x1={x(index)}
                x2={x(index)}
                y1="16"
                y2={plotSize.height - 16}
                className="hc-trend-crosshair"
              />
            )}
            {last >= 0 && (
              <text
                x={plotSize.width - labelWidth + 12}
                y={y(series[last].rate ?? 0) + 4}
                className="hc-trend-end"
              >
                {plotSize.width >= 500 && (
                  <tspan x={plotSize.width - labelWidth + 12} dy="-8">
                    {you.name}
                  </tspan>
                )}
                <tspan x={plotSize.width - labelWidth + 12} dy={plotSize.width >= 500 ? "16" : "0"}>
                  {percent(series[last].rate)}
                </tspan>
              </text>
            )}
            {comparison && otherLast >= 0 && (
              <text
                x={plotSize.width - labelWidth + 12}
                y={Math.max(
                  12,
                  Math.min(
                    232,
                    y(comparison[otherLast].rate ?? 0) +
                      (Math.abs(y(comparison[otherLast].rate ?? 0) - y(series[last]?.rate ?? 0)) <
                      18
                        ? -18
                        : 4),
                  ),
                )}
                className="hc-trend-end hc-trend-other-label"
              >
                {plotSize.width >= 500 && (
                  <tspan x={plotSize.width - labelWidth + 12} dy="-8">
                    {selected?.name}
                  </tspan>
                )}
                <tspan x={plotSize.width - labelWidth + 12} dy={plotSize.width >= 500 ? "16" : "0"}>
                  {percent(comparison[otherLast].rate)}
                </tspan>
              </text>
            )}
          </svg>
          <div className="hc-trend-labels">
            <span>
              <i />
              {you.name}
            </span>
            {selected && (
              <span>
                <i className="hc-trend-key-other" />
                {selected.name}
              </span>
            )}
          </div>
          <div className="hc-chart-dates">
            <span>{formatDate(series[0].date)}</span>
            <span>{formatDate(series.at(-1)?.date ?? "")}</span>
          </div>
          <label className="hc-inspect">
            <span className="sr-only">{C.inspectDate}</span>
            <input
              type="range"
              min="0"
              max={series.length - 1}
              value={index}
              aria-valuetext={readout + (otherReadout ? `. ${otherReadout}` : "")}
              onChange={(event) => setActive(Number(event.target.value))}
              onBlur={() => setActive(null)}
            />
          </label>
          <figcaption className="hc-trend-readout" aria-live="off">
            {active === null ? C.inspectHint : readout + (otherReadout ? ` · ${otherReadout}` : "")}
          </figcaption>
        </figure>
      )}
      <fieldset className="hc-periods hc-signal-periods" aria-label={C.chartRange}>
        {([7, 30, 90] as const).map((days) => (
          <button
            type="button"
            key={days}
            aria-pressed={scope.days === days}
            onClick={() => {
              setActive(null);
              onDaysChange(days);
            }}
          >
            {copy(C.periodLabel, { days })}
          </button>
        ))}
      </fieldset>
      {excluded > 0 && <p className="hc-caption">{copy(C.collectionNote, { count: excluded })}</p>}
      <div className="hc-signal-context">
        <div>
          {benchmark && gap !== null && (
            <div className="hc-gap-summary">
              <p>
                {copy(Math.abs(gap) < 0.05 ? C.youLevel : gap > 0 ? C.youBehind : C.youAhead, {
                  points: points(gap),
                  name: benchmark.name,
                })}
              </p>
              {movement !== null && (
                <p>
                  {Math.abs(movement) < 0.05
                    ? C.gapUnchanged
                    : copy(
                        previousGap !== null && previousGap >= 0 && gap >= 0
                          ? movement > 0
                            ? C.gapNarrowed
                            : C.gapWidened
                          : movement > 0
                            ? C.gaining
                            : C.losing,
                        { points: points(movement), name: benchmark.name },
                      )}
                </p>
              )}
            </div>
          )}
          <p className="hc-caption">{C.multiple}</p>
        </div>
        <div>
          <div className="hc-compare-heading">
            <h3>{C.comparedBrands}</h3>
            <span>{C.mentionRate}</span>
            <span>{C.gapToYou}</span>
          </div>
          {rivals.map((item) => {
            const difference =
              item.stats.rate !== null && stats.rate !== null ? item.stats.rate - stats.rate : null;

            return (
              <button
                type="button"
                key={item.id}
                className="hc-compare-row"
                disabled={item.stats.rate === null}
                aria-label={copy(C.compare, { name: item.name })}
                aria-pressed={selected?.id === item.id}
                onClick={() => onCompare(selected?.id === item.id ? undefined : item.id)}
              >
                <BrandPill name={item.name} domain={item.domain} />
                <span>
                  {item.stats.rate === null ? C.noObservations : percent(item.stats.rate)}
                </span>
                <span>
                  {difference === null
                    ? "—"
                    : Math.abs(difference) < 0.05
                      ? C.level
                      : copy(difference > 0 ? C.ahead : C.behind, { points: points(difference) })}
                </span>
              </button>
            );
          })}
          {!rivals.length && <p className="hc-caption">{C.noCompetitors}</p>}
          <div className="hc-compare-footer">
            {selected && (
              <Button variant="ghost" size="sm" onClick={() => onCompare(undefined)}>
                {C.removeComparison}
              </Button>
            )}
            <Button variant="ghost" size="sm" onClick={onManage}>
              {C.manageCompetitors}
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}
