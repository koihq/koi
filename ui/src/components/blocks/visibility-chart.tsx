import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type DotProps,
  LabelList,
  type LabelProps,
  Line,
  LineChart,
  ReferenceDot,
  XAxis,
  YAxis,
} from "recharts";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { type ChartConfig, ChartContainer } from "@/components/ui/chart";

export type VisibilityPoint = { date: Date; rate: number | null };
export type InspectionInput = "pointer" | "touch" | "keyboard";

type ChartBrand = { name: string; domain?: string };

const HEIGHT = 220;
const PILL_WIDTH = 72;
const PILL_HEIGHT = 28;
const chartConfig = {
  own: { color: "var(--foreground)" },
  other: { color: "var(--muted-foreground)" },
} satisfies ChartConfig;

const clamp = (value: number, min: number, max: number) => Math.max(min, Math.min(max, value));

function endpointLabel(
  brand: ChartBrand,
  lastIndex: number,
  otherRate: number | null,
  isComparison: boolean,
  locale: string | undefined,
) {
  return function EndLabel({ index, x, y, value }: LabelProps) {
    if (index !== lastIndex || x == null || y == null || value == null) return null;

    const rate = Number(value);
    let top = clamp(Number(y) - PILL_HEIGHT / 2, 8, HEIGHT - PILL_HEIGHT - 8);

    if (
      isComparison &&
      otherRate !== null &&
      (Math.abs(rate - otherRate) / 100) * (HEIGHT - 32) < PILL_HEIGHT + 8
    ) {
      top =
        top + PILL_HEIGHT + 8 <= HEIGHT - PILL_HEIGHT - 8
          ? top + PILL_HEIGHT + 8
          : top - PILL_HEIGHT - 8;
    }

    return (
      <foreignObject
        x={Number(x) + 8}
        y={top}
        width={PILL_WIDTH}
        height={PILL_HEIGHT}
        style={{ pointerEvents: "none" }}
      >
        <div className="flex h-7 w-[72px] items-center justify-center gap-1 rounded-full bg-(--signal-pill) text-xs font-medium text-foreground tabular-nums">
          {otherRate !== null && (
            <BrandIcon
              name={brand.name}
              domain={brand.domain}
              lookupLogo={isComparison}
              className="size-3"
            />
          )}
          <span>{`${rate.toLocaleString(locale, { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%`}</span>
        </div>
      </foreignObject>
    );
  };
}

function ActiveMarker({ cx, cy, fill }: DotProps) {
  if (cx == null || cy == null) return <g />;

  return (
    <g className="signal-active-marker" style={{ transform: `translate(${cx}px, ${cy}px)` }}>
      <circle r={8} fill="var(--signal-pill)" />
      <circle r={6} fill={fill} />
    </g>
  );
}

export function VisibilityChart({
  series,
  comparison,
  brand,
  competitor,
  inspected,
  inspectionInput,
  onInspect,
  locale,
  scopeKey,
  variant = "home",
}: {
  series: VisibilityPoint[];
  comparison: VisibilityPoint[];
  brand: ChartBrand;
  competitor?: ChartBrand;
  inspected: number | null;
  inspectionInput: InspectionInput | null;
  onInspect: (index: number | null, input: InspectionInput) => void;
  locale?: string;
  scopeKey?: string;
  variant?: "home" | "detail";
}) {
  const { t } = useTranslation("home");
  const figure = useRef<HTMLElement>(null);
  const previousScope = useRef(scopeKey);
  const [width, setWidth] = useState(320);
  const activeIndex = inspected === null ? null : clamp(inspected, 0, series.length - 1);
  const lastIndex = series.length - 1;
  const last = series[lastIndex];
  const otherLast = comparison[lastIndex];
  const rightPadding = Math.max(width * 0.15, PILL_WIDTH + 16);
  const data = useMemo(
    () =>
      series.map((point, index) => ({
        day: point.date.toISOString(),
        own: point.rate,
        other: comparison[index]?.rate ?? null,
        ownPast: activeIndex !== null && index <= activeIndex ? point.rate : null,
        otherPast:
          activeIndex !== null && index <= activeIndex ? (comparison[index]?.rate ?? null) : null,
      })),
    [series, comparison, activeIndex],
  );
  const selected = activeIndex === null ? null : data[activeIndex];
  const selectedDate = activeIndex === null ? null : series[activeIndex]?.date;
  const readout = selectedDate
    ? `${selectedDate.toLocaleDateString(locale, { month: "short", day: "numeric" })}: ${brand.name} ${selected?.own == null ? t("dashboard.noData") : `${selected.own.toFixed(1)}%`}${competitor && selected?.other != null ? `, ${competitor.name} ${selected.other.toFixed(1)}%` : ""}`
    : undefined;

  useEffect(() => {
    const element = figure.current;
    if (!element) return;

    const observer = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width));
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  useLayoutEffect(() => {
    const changed = previousScope.current !== scopeKey;
    previousScope.current = scopeKey;
    if (!changed || inspectionInput !== "pointer") return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

    const plot = figure.current?.querySelector<HTMLElement>("[data-slot=chart]");
    const animation = plot?.animate(
      [
        { opacity: 0.55, transform: "translateY(5px) scaleY(0.96)" },
        { opacity: 1, transform: "translateY(0) scaleY(1)" },
      ],
      { duration: 260, easing: "cubic-bezier(0.23, 1, 0.32, 1)" },
    );

    return () => animation?.cancel();
  }, [scopeKey, inspectionInput]);

  useEffect(() => {
    if (inspectionInput !== "touch" || inspected === null) return;

    const dismiss = (event: PointerEvent) => {
      if (event.target instanceof Node && !figure.current?.contains(event.target)) {
        onInspect(null, "touch");
      }
    };
    const dismissOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") onInspect(null, "touch");
    };

    document.addEventListener("pointerdown", dismiss);
    document.addEventListener("keydown", dismissOnEscape);
    return () => {
      document.removeEventListener("pointerdown", dismiss);
      document.removeEventListener("keydown", dismissOnEscape);
    };
  }, [inspected, inspectionInput, onInspect]);

  function inspect(clientX: number, input: InspectionInput) {
    const bounds = figure.current?.getBoundingClientRect();
    if (!bounds) return;

    const index = clamp(
      Math.round(((clientX - bounds.left) / Math.max(1, bounds.width - rightPadding)) * lastIndex),
      0,
      lastIndex,
    );
    if (index !== inspected || input !== inspectionInput) onInspect(index, input);
  }

  return (
    <figure
      ref={figure}
      data-motion={inspectionInput === "pointer"}
      className={variant === "detail" ? "relative -mx-5 mt-5 mb-2" : "relative -mx-4 mt-6 sm:-mx-7"}
    >
      <ChartContainer
        config={chartConfig}
        initialDimension={{ width: 320, height: HEIGHT }}
        className="h-[220px] w-full aspect-auto touch-pan-y [&_.recharts-surface]:overflow-visible"
        role="img"
        aria-label={t("dashboard.chartDescription")}
        onPointerDown={(event) => {
          if (!event.isPrimary) return;
          if (event.pointerType === "touch") event.currentTarget.setPointerCapture(event.pointerId);
          inspect(event.clientX, event.pointerType === "touch" ? "touch" : "pointer");
        }}
        onPointerMove={(event) => {
          if (!event.isPrimary || (event.pointerType === "touch" && !event.buttons)) return;
          inspect(event.clientX, event.pointerType === "touch" ? "touch" : "pointer");
        }}
        onPointerLeave={() => {
          if (inspectionInput === "pointer") onInspect(null, "pointer");
        }}
        onPointerCancel={(event) =>
          onInspect(null, event.pointerType === "touch" ? "touch" : "pointer")
        }
      >
        <LineChart data={data} margin={{ top: 16, right: 0, bottom: 16, left: 0 }}>
          <XAxis dataKey="day" hide height={0} padding={{ right: rightPadding }} />
          <YAxis hide width={0} domain={[0, 100]} allowDataOverflow />
          <Line
            dataKey="other"
            type="monotone"
            stroke="var(--color-other)"
            strokeWidth={1.75}
            strokeDasharray="5 5"
            strokeOpacity={activeIndex === null ? 1 : 0.2}
            dot={false}
            activeDot={false}
            isAnimationActive={false}
            connectNulls={false}
          >
            {activeIndex === null && competitor && otherLast && otherLast.rate !== null && (
              <LabelList
                content={endpointLabel(competitor, lastIndex, last?.rate ?? null, true, locale)}
              />
            )}
          </Line>
          <Line
            dataKey="own"
            type="monotone"
            stroke="var(--color-own)"
            strokeWidth={4}
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeOpacity={activeIndex === null ? 1 : 0.2}
            dot={false}
            activeDot={false}
            isAnimationActive={false}
            connectNulls={false}
          >
            {activeIndex === null && last && last.rate !== null && (
              <LabelList
                content={endpointLabel(brand, lastIndex, otherLast?.rate ?? null, false, locale)}
              />
            )}
          </Line>
          {activeIndex !== null && (
            <>
              <Line
                dataKey="otherPast"
                type="monotone"
                stroke="var(--color-other)"
                strokeWidth={1.75}
                strokeDasharray="5 5"
                dot={false}
                activeDot={false}
                isAnimationActive={false}
                connectNulls={false}
              />
              <Line
                dataKey="ownPast"
                type="monotone"
                stroke="var(--color-own)"
                strokeWidth={4}
                strokeLinecap="round"
                strokeLinejoin="round"
                dot={false}
                activeDot={false}
                isAnimationActive={false}
                connectNulls={false}
              />
            </>
          )}
          {activeIndex === null ? (
            <>
              {series.map((point, index) =>
                index < lastIndex &&
                point.rate !== null &&
                (index === 0 || series[index - 1].rate === null) &&
                series[index + 1].rate === null ? (
                  <ReferenceDot
                    key={`own-${point.date.toISOString()}`}
                    x={point.date.toISOString()}
                    y={point.rate}
                    r={3}
                    fill="var(--color-own)"
                    stroke="none"
                  />
                ) : null,
              )}
              {comparison.map((point, index) =>
                index < lastIndex &&
                point.rate !== null &&
                (index === 0 || comparison[index - 1].rate === null) &&
                comparison[index + 1].rate === null ? (
                  <ReferenceDot
                    key={`other-${point.date.toISOString()}`}
                    x={point.date.toISOString()}
                    y={point.rate}
                    r={3}
                    fill="var(--color-other)"
                    stroke="none"
                  />
                ) : null,
              )}
              {last && last.rate !== null && (
                <ReferenceDot
                  x={last.date.toISOString()}
                  y={last.rate}
                  r={6}
                  fill="var(--color-own)"
                  stroke="var(--signal-pill)"
                  strokeWidth={2}
                />
              )}
              {otherLast && otherLast.rate !== null && (
                <ReferenceDot
                  x={otherLast.date.toISOString()}
                  y={otherLast.rate}
                  r={6}
                  fill="var(--color-other)"
                  stroke="var(--signal-pill)"
                  strokeWidth={2}
                />
              )}
            </>
          ) : (
            <>
              {selected?.own !== null && selected?.own !== undefined && (
                <ReferenceDot
                  x={selected.day}
                  y={selected.own}
                  fill="var(--color-own)"
                  shape={ActiveMarker}
                />
              )}
              {selected?.other !== null && selected?.other !== undefined && (
                <ReferenceDot
                  x={selected.day}
                  y={selected.other}
                  fill="var(--color-other)"
                  shape={ActiveMarker}
                />
              )}
            </>
          )}
        </LineChart>
      </ChartContainer>
      <label className="absolute inset-0 pointer-events-none has-focus-visible:rounded-md has-focus-visible:outline-2 has-focus-visible:outline-offset-4 has-focus-visible:outline-ring">
        <span className="sr-only">{t("dashboard.inspectDay")}</span>
        <input
          type="range"
          min={0}
          max={lastIndex}
          value={activeIndex ?? lastIndex}
          aria-valuetext={readout}
          onFocus={() => onInspect(activeIndex ?? lastIndex, "keyboard")}
          onChange={(event) => onInspect(Number(event.target.value), "keyboard")}
          onKeyDown={(event) => {
            if (event.key === "Escape") onInspect(null, "keyboard");
          }}
          onBlur={() => onInspect(null, "keyboard")}
          className="absolute inset-0 size-full opacity-0 pointer-events-none"
        />
      </label>
    </figure>
  );
}
