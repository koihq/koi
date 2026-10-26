import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Prompt } from "@/api/prompts.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { trendPath } from "@/lib/trend-path";
import { rate, series as visibilitySeries } from "@/lib/visibility";

export function PromptPreviewLink({ prompt }: { prompt: Prompt }) {
  const { t } = useTranslation("home");
  const [open, setOpen] = useState(false);
  const link = (
    <Link
      to="/prompts/$promptId"
      params={{ promptId: prompt.id }}
      className="block max-w-2xl text-sm/normal font-medium text-foreground underline-offset-[3px] [overflow-wrap:anywhere] hover:underline"
    >
      {prompt.text}
    </Link>
  );

  if (!prompt.visibility?.metric.completed) return link;

  const series = visibilitySeries(prompt.visibility);
  const currentRate = rate(prompt.visibility);
  const rivals = prompt.visibility.competitors
    .map((item) => ({ item, rate: item.metric.rate }))
    .sort((left, right) => (right.rate ?? -1) - (left.rate ?? -1))
    .slice(0, 3);
  const width = 320;
  const height = 96;
  const points = series.map((point, index) =>
    point.rate === null
      ? null
      : {
          x: (index / Math.max(1, series.length - 1)) * width,
          y: 8 + ((100 - point.rate) / 100) * (height - 16),
        },
  );
  const segments: { x: number; y: number }[][] = [];
  for (const [index, point] of points.entries()) {
    if (point) {
      if (!segments.length || points[index - 1] === null) segments.push([]);
      segments.at(-1)?.push(point);
    }
  }

  return (
    <Tooltip open={open} onOpenChange={(next) => !next && setOpen(false)}>
      <TooltipTrigger
        render={link}
        onPointerEnter={(event) => {
          if (event.pointerType === "mouse") setOpen(true);
        }}
        onPointerLeave={() => setOpen(false)}
        onFocus={(event) => {
          if (event.currentTarget.matches(":focus-visible")) setOpen(true);
        }}
        onBlur={() => setOpen(false)}
        onKeyDown={(event) => {
          if (event.key === "Escape") setOpen(false);
        }}
        onClick={() => setOpen(false)}
      />
      {open && (
        <TooltipContent
          side="right"
          align="start"
          sideOffset={14}
          className="flex w-[min(290px,calc(100vw-32px))] max-w-[calc(100vw-32px)] flex-col items-stretch gap-0 overflow-hidden rounded-[12px] border border-[var(--metric-tooltip-border)] [background:linear-gradient(165deg,#ffffff0c,#ffffff02_55%),color-mix(in_srgb,var(--popover)_80%,transparent)]! p-[14px_14px_0] text-foreground shadow-[0_16px_60px_var(--metric-tooltip-shadow)] backdrop-blur-[24px] backdrop-saturate-[130%] animate-none! [&>[data-slot=tooltip-arrow]]:hidden"
        >
          <div className="flex items-start justify-between gap-4">
            <div className="grid gap-1.5">
              <span className="text-xs text-muted-foreground">{t("dashboard.yourRate")}</span>
              <strong className="text-xl font-medium tabular-nums leading-none">
                {currentRate === null ? "—" : `${currentRate.toFixed(1)}%`}
              </strong>
            </div>
            <div className="grid justify-items-end gap-1.5 text-right">
              <span className="text-xs text-muted-foreground">{t("dashboard.competitors")}</span>
              {rivals.length ? (
                <span className="pointer-events-none inline-flex pe-2">
                  {rivals.map(({ item }) => (
                    <span
                      key={item.id}
                      className="relative -me-[7px] grid size-6 place-items-center rounded-full bg-surface shadow-[0_0_0_1px_var(--surface)] [&_[data-slot=brand-icon]]:size-4"
                    >
                      <BrandPill name={item.name} domain={item.domain} iconOnly />
                    </span>
                  ))}
                </span>
              ) : (
                <span className="text-xs text-muted-foreground">
                  {t("dashboard.noCompetitors")}
                </span>
              )}
            </div>
          </div>
          {segments.length ? (
            <svg
              viewBox={`0 0 ${width} ${height}`}
              preserveAspectRatio="none"
              className="mt-3 -mx-3.5 block h-24 w-[calc(100%+28px)]"
              role="img"
              aria-label={t("dashboard.chartDescription")}
            >
              {segments.map((segment) =>
                segment.length > 1 ? (
                  <path
                    key={segment[0].x}
                    d={trendPath(segment)}
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="3"
                    strokeLinecap="round"
                    vectorEffect="non-scaling-stroke"
                  />
                ) : (
                  <circle
                    key={segment[0].x}
                    cx={segment[0].x}
                    cy={segment[0].y}
                    r="3"
                    fill="currentColor"
                  />
                ),
              )}
            </svg>
          ) : (
            <div className="mt-3 -mx-3.5 grid h-24 w-[calc(100%+28px)] place-items-center text-xs text-muted-foreground">
              {t("dashboard.noComparison")}
            </div>
          )}
        </TooltipContent>
      )}
    </Tooltip>
  );
}
