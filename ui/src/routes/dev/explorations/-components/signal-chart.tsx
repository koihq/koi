import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { type ComparedBrand, type dailySeries, percent, points } from "../-lib/prompt-visibility";
import { trendPath } from "../-lib/trend-path";
import C from "../-locales/prompt-visibility.en.json";
import { copy } from "./list-controls";

type Series = ReturnType<typeof dailySeries>;
export type SignalInspectionInput = "pointer" | "touch" | "keyboard";
const PILL_WIDTH = 72;
const PILL_HEIGHT = 28;
const PILL_GAP = 8;
const PILL_RIGHT_INSET = 8;
const CHART_END_PADDING = 0.15;
const clamp = (value: number, min: number, max: number) => Math.max(min, Math.min(max, value));

function signedPoints(value: number) {
  const rounded = Number(value.toFixed(1));
  return copy(C.shortPoints, {
    points: `${rounded > 0 ? "+" : rounded < 0 ? "−" : ""}${points(rounded)}`,
  });
}

export function formatSignalDate(value: string) {
  return new Date(`${value}T12:00:00Z`).toLocaleDateString("en", {
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  });
}

export function SignalChart({
  series,
  brand,
  comparison,
  competitor,
  scopeKey,
  animate,
  selectedIndex,
  inspectionInput,
  onSelectedIndexChange,
}: {
  series: Series;
  brand: ComparedBrand;
  comparison?: Series;
  competitor?: ComparedBrand;
  scopeKey: string;
  animate: boolean;
  selectedIndex: number | null;
  inspectionInput: SignalInspectionInput | null;
  onSelectedIndexChange: (index: number | null, input: SignalInspectionInput) => void;
}) {
  const futureClipId = useId().replaceAll(":", "");
  const figure = useRef<HTMLElement>(null);
  const plot = useRef<SVGSVGElement>(null);
  const previousView = useRef<{
    scopeKey: string;
    endpoints: Map<string, { left: number; top: number }>;
  } | null>(null);
  const [size, setSize] = useState({ width: 1000, height: 220 });
  const active = selectedIndex !== null;
  const index = clamp(selectedIndex ?? series.length - 1, 0, series.length - 1);

  useLayoutEffect(() => {
    const element = plot.current;
    if (!element) return;

    const resize = () => {
      const { width, height } = element.getBoundingClientRect();
      setSize({ width: Math.max(1, width), height });
    };
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const touchInspecting = inspectionInput === "touch" && active;
  useEffect(() => {
    if (!touchInspecting) return;

    const dismiss = (event: PointerEvent) => {
      if (event.target instanceof Node && !figure.current?.contains(event.target))
        onSelectedIndexChange(null, "touch");
    };
    const onEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") onSelectedIndexChange(null, "touch");
    };
    document.addEventListener("pointerdown", dismiss);
    document.addEventListener("keydown", onEscape);
    return () => {
      document.removeEventListener("pointerdown", dismiss);
      document.removeEventListener("keydown", onEscape);
    };
  }, [onSelectedIndexChange, touchInspecting]);

  const chartWidth = Math.max(
    1,
    Math.min(
      size.width * (1 - CHART_END_PADDING),
      size.width - PILL_WIDTH - PILL_GAP - PILL_RIGHT_INSET,
    ),
  );
  const x = (i: number) => (i / Math.max(1, series.length - 1)) * chartWidth;
  const y = (rate: number) => 16 + ((100 - rate) / 100) * (size.height - 32);
  const selectedX = x(index);
  const lines = [
    { brand, series, other: false },
    ...(comparison && competitor ? [{ brand: competitor, series: comparison, other: true }] : []),
  ];
  const endpoints = lines.flatMap((line) => {
    const last = line.series.reduce((found, point, i) => (point.rate !== null ? i : found), -1);
    if (last < 0) return [];

    const point = line.series[last];
    const pointX = x(last);
    const pointY = y(point.rate ?? 0);
    return [
      {
        ...line,
        point,
        pointX,
        pointY,
        left: Math.min(pointX + PILL_GAP, size.width - PILL_WIDTH - PILL_RIGHT_INSET),
        top: clamp(pointY - PILL_HEIGHT / 2, 8, size.height - PILL_HEIGHT - 8),
      },
    ];
  });
  const [first, second] = endpoints;
  if (
    first &&
    second &&
    Math.abs(first.left - second.left) < PILL_WIDTH + 8 &&
    Math.abs(first.top - second.top) < PILL_HEIGHT + 8
  ) {
    const below = first.top + PILL_HEIGHT + 8;
    second.top = below <= size.height - PILL_HEIGHT - 8 ? below : first.top - PILL_HEIGHT - 8;
  }
  const endpointLayout = endpoints.map((end) => ({
    brandId: end.brand.id,
    left: end.left,
    top: end.top,
  }));
  const endpointLayoutKey = JSON.stringify(endpointLayout);

  useLayoutEffect(() => {
    const previous = previousView.current;
    const currentLayout = JSON.parse(endpointLayoutKey) as typeof endpointLayout;
    const positions = new Map(
      currentLayout.map(({ brandId, left, top }) => [brandId, { left, top }]),
    );
    previousView.current = { scopeKey, endpoints: positions };
    if (!previous || previous.scopeKey === scopeKey || !animate) return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

    const animations: Animation[] = [];
    const plotElement = figure.current?.querySelector<HTMLElement>(".hc-trend-plot");
    if (plotElement) {
      animations.push(
        plotElement.animate(
          [
            { opacity: 0.55, transform: "translateY(5px) scaleY(0.96)" },
            { opacity: 1, transform: "translateY(0) scaleY(1)" },
          ],
          { duration: 260, easing: "cubic-bezier(0.23, 1, 0.32, 1)" },
        ),
      );
    }

    const pills = figure.current?.querySelectorAll<HTMLElement>(".hc-trend-pill");
    currentLayout.forEach((end) => {
      const pill = Array.from(pills ?? []).find((item) => item.dataset.brandId === end.brandId);
      if (!pill) return;

      const start = previous.endpoints.get(end.brandId);
      const from = start ?? { left: end.left, top: end.top + 5 };
      animations.push(
        pill.animate(
          [
            {
              opacity: start ? 0.75 : 0,
              transform: `translate(${from.left}px, ${from.top}px) scale(${start ? 0.94 : 0.88})`,
            },
            { opacity: 1, transform: `translate(${end.left}px, ${end.top}px) scale(1)` },
          ],
          { duration: 260, easing: "cubic-bezier(0.23, 1, 0.32, 1)" },
        ),
      );
    });

    return () => {
      animations.forEach((animation) => {
        animation.cancel();
      });
    };
  }, [animate, endpointLayoutKey, scopeKey]);

  const readout = lines
    .map((line) => {
      const point = line.series[index];
      const change =
        point.changePoints === null
          ? C.noDailyComparison
          : copy(C.dailyChange, { points: signedPoints(point.changePoints) });

      return copy(C.point, {
        date: point.date,
        brand: line.brand.name,
        rate: point.rate === null ? C.noPeriodAnswers : percent(point.rate),
        change,
        count: point.completed,
      });
    })
    .join(". ");

  function inspect(clientX: number, input: SignalInspectionInput) {
    const bounds = plot.current?.getBoundingClientRect();
    if (!bounds) return;

    const next = clamp(
      Math.round(
        ((clientX - bounds.left) / Math.max(1, bounds.width * (chartWidth / size.width))) *
          (series.length - 1),
      ),
      0,
      series.length - 1,
    );
    if (selectedIndex !== next || inspectionInput !== input) onSelectedIndexChange(next, input);
  }

  return (
    <figure ref={figure} className="hc-trend" data-inspecting={active} data-animate={animate}>
      <div className="hc-trend-plot">
        <svg
          ref={plot}
          viewBox={`0 0 ${size.width} ${size.height}`}
          preserveAspectRatio="none"
          role="img"
          aria-label={C.chartDescription}
          onPointerDown={(event) => {
            if (!event.isPrimary) return;
            if (event.pointerType === "touch")
              event.currentTarget.setPointerCapture(event.pointerId);
            inspect(event.clientX, event.pointerType === "touch" ? "touch" : "pointer");
          }}
          onPointerMove={(event) => {
            if (!event.isPrimary || (event.pointerType === "touch" && !event.buttons)) return;
            inspect(event.clientX, event.pointerType === "touch" ? "touch" : "pointer");
          }}
          onPointerCancel={() => onSelectedIndexChange(null, "pointer")}
          onPointerLeave={() => {
            if (inspectionInput === "pointer") onSelectedIndexChange(null, "pointer");
          }}
        >
          {active && (
            <defs>
              <clipPath id={futureClipId}>
                <rect x="0" y="0" width={selectedX + 2} height={size.height} />
              </clipPath>
            </defs>
          )}
          {lines.map((line) => {
            const coordinates = line.series.map((point, i) =>
              point.rate === null ? null : { x: x(i), y: y(point.rate) },
            );
            const path = trendPath(coordinates);
            const lineClass = line.other ? "hc-trend-line hc-trend-comparison" : "hc-trend-line";

            return (
              <g key={line.brand.id}>
                <path d={path} className={active ? `${lineClass} hc-trend-future` : lineClass} />
                {active && (
                  <path d={path} className={lineClass} clipPath={`url(#${futureClipId})`} />
                )}
                {coordinates.map((point, i) =>
                  point && !coordinates[i - 1] && !coordinates[i + 1] ? (
                    <circle
                      key={line.series[i].day}
                      cx={point.x}
                      cy={point.y}
                      r="3"
                      className={line.other ? "hc-trend-other-dot" : "hc-trend-dot"}
                    />
                  ) : null,
                )}
              </g>
            );
          })}
          {endpoints.map((end) => (
            <g key={end.brand.id} className="hc-trend-endpoint">
              <circle cx={end.pointX} cy={end.pointY} r="8" className="hc-trend-marker-halo" />
              <circle
                cx={end.pointX}
                cy={end.pointY}
                r="6"
                className={end.other ? "hc-trend-other-dot" : "hc-trend-dot"}
              />
            </g>
          ))}
          {active &&
            lines.map((line) =>
              line.series[index].rate !== null ? (
                <g
                  key={line.brand.id}
                  className="hc-trend-active-marker"
                  data-motion={inspectionInput === "pointer" && animate}
                  style={{
                    transform: `translate(${x(index)}px, ${y(line.series[index].rate ?? 0)}px)`,
                  }}
                >
                  <circle r="8" className="hc-trend-marker-halo" />
                  <circle r="6" className={line.other ? "hc-trend-other-dot" : "hc-trend-dot"} />
                </g>
              ) : null,
            )}
        </svg>
      </div>
      {endpoints.map((end) => (
        <div
          key={end.brand.id}
          className="hc-trend-pill"
          data-other={end.other}
          data-brand-id={end.brand.id}
          style={{ transform: `translate(${end.left}px, ${end.top}px)` }}
          aria-hidden="true"
        >
          {comparison && competitor && (
            <BrandIcon
              name={end.brand.name}
              domain={end.brand.domain}
              src={end.brand.src}
              className="size-3"
            />
          )}
          <span aria-hidden="true">{percent(end.point.rate)}</span>
        </div>
      ))}
      <label className="hc-inspect">
        <span className="sr-only">{C.inspectDate}</span>
        <input
          type="range"
          min="0"
          max={series.length - 1}
          value={index}
          aria-valuetext={readout}
          onFocus={() => onSelectedIndexChange(index, "keyboard")}
          onChange={(event) => onSelectedIndexChange(Number(event.target.value), "keyboard")}
          onKeyDown={(event) => {
            if (event.key === "Escape") onSelectedIndexChange(null, "keyboard");
          }}
          onBlur={() => onSelectedIndexChange(null, "keyboard")}
        />
      </label>
    </figure>
  );
}
