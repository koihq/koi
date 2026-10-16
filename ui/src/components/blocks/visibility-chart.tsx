import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  LabelList,
  type LabelProps,
  Line,
  LineChart,
  type MouseHandlerDataParam,
  ReferenceDot,
  XAxis,
  YAxis,
} from "recharts";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { type ChartConfig, ChartContainer, ChartTooltip } from "@/components/ui/chart";

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
            <BrandIcon name={brand.name} domain={brand.domain} className="size-3" />
          )}
          <span>{`${rate.toLocaleString(locale, { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%`}</span>
        </div>
      </foreignObject>
    );
  };
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
  variant?: "home" | "detail";
}) {
  const { t } = useTranslation("home");
  const figure = useRef<HTMLElement>(null);
  const [width, setWidth] = useState(320);
  const activeIndex = inspected === null ? null : clamp(inspected, 0, series.length - 1);
  const lastIndex = series.length - 1;
  const last = series[lastIndex];
  const otherLast = comparison[lastIndex];
  const rightPadding = Math.max(width * 0.15, PILL_WIDTH + 16);
  const data = series.map((point, index) => ({
    day: point.date.toISOString(),
    own: point.rate,
    other: comparison[index]?.rate ?? null,
    ownPast: activeIndex !== null && index <= activeIndex ? point.rate : null,
    otherPast:
      activeIndex !== null && index <= activeIndex ? (comparison[index]?.rate ?? null) : null,
  }));
  const selected = activeIndex === null ? null : data[activeIndex];
  const selectedDate = activeIndex === null ? null : series[activeIndex]?.date;
  const readout = selectedDate
    ? `${selectedDate.toLocaleDateString(locale, { month: "short", day: "numeric" })}: ${brand.name} ${selected?.own == null ? "—" : `${selected.own.toFixed(1)}%`}${competitor && selected?.other != null ? `, ${competitor.name} ${selected.other.toFixed(1)}%` : ""}`
    : undefined;

  useEffect(() => {
    const element = figure.current;
    if (!element) return;

    const observer = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width));
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

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

  function inspect(payload: MouseHandlerDataParam, input: InspectionInput) {
    const index = Number(payload.activeTooltipIndex);
    if (payload.isTooltipActive && Number.isInteger(index) && index >= 0 && index < series.length) {
      onInspect(index, input);
    }
  }

  return (
    <figure
      ref={figure}
      className={variant === "detail" ? "relative -mx-5 mt-5 mb-2" : "relative -mx-4 mt-6 sm:-mx-7"}
    >
      <ChartContainer
        config={chartConfig}
        initialDimension={{ width: 320, height: HEIGHT }}
        className="h-[220px] w-full aspect-auto touch-pan-y [&_.recharts-surface]:overflow-visible"
        role="img"
        aria-label={t("dashboard.chartDescription")}
        onTouchCancel={() => onInspect(null, "touch")}
      >
        <LineChart
          data={data}
          margin={{ top: 16, right: 0, bottom: 16, left: 0 }}
          onMouseMove={(payload) => inspect(payload, "pointer")}
          onMouseLeave={() => {
            if (inspectionInput === "pointer") onInspect(null, "pointer");
          }}
          onTouchStart={(payload) => inspect(payload, "touch")}
          onTouchMove={(payload) => inspect(payload, "touch")}
        >
          <XAxis dataKey="day" hide height={0} padding={{ right: rightPadding }} />
          <YAxis hide width={0} domain={[0, 100]} allowDataOverflow />
          <ChartTooltip content={() => null} cursor={false} />
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
                  r={6}
                  fill="var(--color-own)"
                  stroke="var(--signal-pill)"
                  strokeWidth={2}
                />
              )}
              {selected?.other !== null && selected?.other !== undefined && (
                <ReferenceDot
                  x={selected.day}
                  y={selected.other}
                  r={6}
                  fill="var(--color-other)"
                  stroke="var(--signal-pill)"
                  strokeWidth={2}
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
