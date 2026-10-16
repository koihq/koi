import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { trendPath } from "@/lib/trend-path";

export type VisibilityPoint = { date: Date; rate: number };

export function VisibilityChart({
  series,
  comparison,
  inspected,
  onInspect,
  locale,
  variant = "home",
}: {
  series: VisibilityPoint[];
  comparison: VisibilityPoint[];
  inspected: number | null;
  onInspect: (index: number | null) => void;
  locale?: string;
  variant?: "home" | "detail";
}) {
  const { t } = useTranslation("home");
  const chart = useRef<HTMLDivElement>(null);
  const plot = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(1000);
  const height = 220;
  const end = Math.min(width * 0.85, width - 84);
  const x = (index: number) => (index / Math.max(1, series.length - 1)) * end;
  const y = (rate: number) => 16 + ((100 - rate) / 100) * (height - 32);
  const path = (items: VisibilityPoint[]) =>
    trendPath(items.map((item, index) => ({ x: x(index), y: y(item.rate) })));
  const activeIndex = inspected ?? series.length - 1;
  const active = series[activeIndex];
  const other = comparison[activeIndex];

  useEffect(() => {
    if (!plot.current) return;

    const observer = new ResizeObserver(([entry]) => {
      setWidth(entry.contentRect.width);
    });
    observer.observe(plot.current);
    return () => observer.disconnect();
  }, []);

  return (
    <div className={variant === "detail" ? "-mx-5 mt-5 mb-2" : "-mx-4 mt-6 sm:-mx-7"} ref={chart}>
      <div
        className="relative h-[220px] touch-pan-y"
        onPointerLeave={() => onInspect(null)}
        ref={plot}
      >
        <svg
          viewBox={`0 0 ${width} ${height}`}
          preserveAspectRatio="none"
          className="size-full overflow-visible"
          role="img"
          aria-label={t("dashboard.chartDescription")}
        >
          {comparison.length > 0 && (
            <path
              d={path(comparison)}
              fill="none"
              stroke="var(--muted-foreground)"
              strokeWidth="1.75"
              strokeDasharray="5 5"
              vectorEffect="non-scaling-stroke"
            />
          )}
          <path
            d={path(series)}
            fill="none"
            stroke="var(--foreground)"
            strokeWidth="4"
            strokeLinecap="round"
            strokeLinejoin="round"
            vectorEffect="non-scaling-stroke"
          />
          {active && (
            <circle cx={x(activeIndex)} cy={y(active.rate)} r="6" fill="var(--foreground)" />
          )}
          {other && (
            <circle cx={x(activeIndex)} cy={y(other.rate)} r="5" fill="var(--muted-foreground)" />
          )}
        </svg>
        {active && (
          <output
            className="pointer-events-none absolute grid h-7 w-[72px] place-items-center rounded-full bg-muted text-xs text-foreground tabular-nums"
            style={{ left: x(activeIndex) + 8, top: y(active.rate) - 14 }}
          >
            {`${active.rate.toFixed(1)}%`}
          </output>
        )}
        {other && (
          <output
            className="pointer-events-none absolute grid h-7 w-[72px] place-items-center rounded-full bg-muted text-xs text-foreground tabular-nums text-muted-foreground"
            style={{
              left: x(activeIndex) + 8,
              top: y(other.rate) - 14 + (Math.abs(other.rate - (active?.rate ?? 0)) < 14 ? 33 : 0),
            }}
          >
            {`${other.rate.toFixed(1)}%`}
          </output>
        )}
        <input
          type="range"
          min={0}
          max={series.length - 1}
          value={activeIndex}
          onChange={(event) => onInspect(Number(event.target.value))}
          onPointerMove={(event) => {
            if (!chart.current) return;
            const bounds = chart.current.getBoundingClientRect();
            const fraction = (event.clientX - bounds.left) / end;
            onInspect(
              Math.min(series.length - 1, Math.max(0, Math.round(fraction * (series.length - 1)))),
            );
          }}
          aria-label={t("dashboard.inspectDay")}
          className="absolute inset-0 size-full cursor-crosshair opacity-0 focus-visible:opacity-20"
        />
      </div>
      <div className="sr-only">
        <time dateTime={series[0]?.date.toISOString()}>
          {series[0]?.date.toLocaleDateString(locale, { month: "short", day: "numeric" })}
        </time>
        <span aria-hidden="true">{"–"}</span>
        <time dateTime={series.at(-1)?.date.toISOString()}>
          {series.at(-1)?.date.toLocaleDateString(locale, { month: "short", day: "numeric" })}
        </time>
      </div>
    </div>
  );
}
