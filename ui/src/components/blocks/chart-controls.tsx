import { cn } from "cn";
import { useTranslation } from "react-i18next";
import type { VisibilityEngine } from "@/api/generated";
import { EngineIcon } from "@/components/blocks/engine-icon";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { engines } from "@/lib/visibility";

export const chartRanges = [7, 30, 90] as const;
export type ChartRange = (typeof chartRanges)[number];

/** The chart's period, as quiet text pills. */
export function RangeControl({
  days,
  onChange,
}: {
  days: ChartRange;
  onChange: (days: ChartRange) => void;
}) {
  const { t } = useTranslation("home");

  return (
    <fieldset className="flex gap-0.5" aria-label={t("dashboard.range")}>
      {chartRanges.map((range) => (
        <Button
          key={range}
          type="button"
          aria-pressed={days === range}
          variant="ghost"
          size="pill"
          className="text-muted-foreground"
          onClick={() => onChange(range)}
        >
          {`${range}D`}
        </Button>
      ))}
    </fieldset>
  );
}

/**
 * Narrows the chart to some engines. No selection means every engine, so every icon reads as
 * included; once one is chosen, the excluded ones recede.
 */
export function EngineFilter({
  selected,
  onToggle,
}: {
  selected: VisibilityEngine[];
  onToggle: (engine: VisibilityEngine) => void;
}) {
  const { t } = useTranslation("home");

  return (
    <TooltipProvider delay={300}>
      <fieldset className="flex gap-0.5" aria-label={t("dashboard.filterEngines")}>
        {engines.map((engine) => {
          const pressed = selected.includes(engine.id);

          return (
            <Tooltip key={engine.id}>
              <TooltipTrigger
                render={
                  <Button
                    type="button"
                    aria-label={engine.name}
                    aria-pressed={pressed}
                    variant="ghost"
                    size="circle-sm"
                    className={cn(
                      "text-muted-foreground",
                      selected.length > 0 && !pressed && "opacity-40 hover:opacity-100",
                    )}
                    onClick={() => onToggle(engine.id)}
                  />
                }
              >
                <EngineIcon engine={engine.id} className="size-4" />
              </TooltipTrigger>
              <TooltipContent>{engine.name}</TooltipContent>
            </Tooltip>
          );
        })}
      </fieldset>
    </TooltipProvider>
  );
}
