import { Link } from "@tanstack/react-router";
import { useId, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Competitor } from "@/api/competitors.api";
import type { Prompt } from "@/api/prompts.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { mockRate, mockSeries, type VisibilityScope } from "@/lib/mock-visibility";
import { trendAreaPath, trendPath } from "@/lib/trend-path";

export function PromptPreviewLink({
  prompt,
  brandID,
  competitors,
  scope,
}: {
  prompt: Prompt;
  brandID: string;
  competitors: Competitor[];
  scope: VisibilityScope;
}) {
  const { t } = useTranslation("home");
  const gradientId = useId().replaceAll(":", "");
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

  if (!prompt.confirmed) return link;

  const series = mockSeries(brandID, [prompt], scope);
  const rate = mockRate(brandID, [prompt], scope);
  const rivals = competitors
    .filter((item) => item.confirmed)
    .map((item) => ({ item, rate: mockRate(brandID, [prompt], scope, item) ?? 0 }))
    .sort((left, right) => right.rate - left.rate)
    .slice(0, 3);
  const width = 320;
  const height = 96;
  const points = series.map((point, index) => ({
    x: (index / Math.max(1, series.length - 1)) * width,
    y: 8 + ((100 - point.rate) / 100) * (height - 16),
  }));
  const line = trendPath(points);
  const area = trendAreaPath(points, height);

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
                {rate === null ? "—" : `${rate.toFixed(1)}%`}
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
          {line ? (
            <svg
              viewBox={`0 0 ${width} ${height}`}
              preserveAspectRatio="none"
              className="mt-3 -mx-3.5 block h-24 w-[calc(100%+28px)]"
              role="img"
              aria-label={t("dashboard.chartDescription")}
            >
              <defs>
                <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="currentColor" stopOpacity="0.2" />
                  <stop offset="100%" stopColor="currentColor" stopOpacity="0.08" />
                </linearGradient>
              </defs>
              {area && <path d={area} fill={`url(#${gradientId})`} />}
              <path
                d={line}
                fill="none"
                stroke="currentColor"
                strokeWidth="4"
                strokeLinecap="round"
                strokeLinejoin="round"
                vectorEffect="non-scaling-stroke"
              />
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
