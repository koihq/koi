import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { Competitor } from "@/api/competitors.api";
import type { Prompt } from "@/api/prompts.api";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { mockRate, mockSeries, type VisibilityScope } from "@/lib/mock-visibility";
import { trendPath } from "@/lib/trend-path";

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
  const link = (
    <Link
      to="/prompts/$promptId"
      params={{ promptId: prompt.id }}
      className="block max-w-2xl whitespace-normal text-sm underline-offset-4 hover:underline"
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
  const line = trendPath(
    series.map((point, index) => {
      const x = (index / Math.max(1, series.length - 1)) * 280;
      const y = 70 - (point.rate / 100) * 70;
      return { x, y };
    }),
  );

  return (
    <Tooltip>
      <TooltipTrigger render={link} />
      <TooltipContent
        side="bottom"
        align="start"
        className="w-80 max-w-[calc(100vw-2rem)] flex-col items-stretch gap-3 p-4"
      >
        <div className="flex items-baseline justify-between gap-3">
          <span className="font-medium">{t("dashboard.title")}</span>
          <strong className="text-lg font-medium tabular-nums">{`${rate?.toFixed(1)}%`}</strong>
        </div>
        <svg
          viewBox="0 0 280 70"
          preserveAspectRatio="none"
          className="h-16 w-full"
          role="img"
          aria-label={t("dashboard.chartDescription")}
        >
          <path
            d={line}
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            vectorEffect="non-scaling-stroke"
          />
        </svg>
        {rivals.length > 0 && (
          <div className="space-y-1 border-t border-background/20 pt-2">
            {rivals.map(({ item, rate: rivalRate }) => (
              <p key={item.id} className="flex justify-between gap-3">
                <span className="truncate">{item.name}</span>
                <span className="tabular-nums">{`${rivalRate.toFixed(1)}%`}</span>
              </p>
            ))}
          </div>
        )}
      </TooltipContent>
    </Tooltip>
  );
}
