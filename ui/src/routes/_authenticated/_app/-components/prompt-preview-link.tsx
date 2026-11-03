import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Prompt } from "@/api/prompts.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatRate } from "@/lib/format";
import { trendPath } from "@/lib/trend-path";
import { rate, series as visibilitySeries } from "@/lib/visibility";

export function PromptPreviewLink({ prompt }: { prompt: Prompt }) {
  const { t, i18n } = useTranslation("home");
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
  const height = 64;
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
          className="surface-float flex w-72 max-w-[calc(100vw-2rem)] flex-col items-stretch gap-0 overflow-hidden bg-transparent! px-3.5 pt-3.5 pb-0 text-foreground animate-none! [&>[data-slot=tooltip-arrow]]:hidden"
        >
          <div className="flex items-start justify-between gap-4">
            <div className="grid gap-1.5">
              <span className="text-xs text-muted-foreground">{t("dashboard.yourRate")}</span>
              <strong className="text-section font-semibold tabular-nums">
                {currentRate === null ? "—" : formatRate(currentRate, i18n.language)}
              </strong>
            </div>
            <div className="grid justify-items-end gap-1.5 text-right">
              <span className="text-xs text-muted-foreground">{t("dashboard.competitors")}</span>
              {rivals.length ? (
                <span className="pointer-events-none inline-flex gap-1">
                  {rivals.map(({ item }) => (
                    <BrandPill key={item.id} name={item.name} domain={item.domain} iconOnly />
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
              className="mt-3 -mx-3.5 block h-16 w-[calc(100%+28px)]"
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
            <div className="mt-3 -mx-3.5 grid h-16 w-[calc(100%+28px)] place-items-center text-xs text-muted-foreground">
              {t("dashboard.noComparison")}
            </div>
          )}
        </TooltipContent>
      )}
    </Tooltip>
  );
}
