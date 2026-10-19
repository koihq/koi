import { type ReactNode, useRef, useState } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

export type PromptBreakdownItem = {
  id: string;
  name: string;
  rate: number | null;
  icon: ReactNode;
};

const percent = (rate: number | null) => (rate === null ? "—" : `${rate.toFixed(1)}%`);

export function PromptRateBreakdown({
  label,
  items,
  emptyLabel = "—",
}: {
  label: string;
  items: PromptBreakdownItem[];
  emptyLabel?: string;
}) {
  const [open, setOpen] = useState(false);
  const touch = useRef(false);
  const openOnPointerDown = useRef(false);

  if (!items.length) return <span className="text-xs text-muted-foreground">{emptyLabel}</span>;

  const summary = items.reduce<number | null>(
    (highest, item) =>
      item.rate !== null && (highest === null || item.rate > highest) ? item.rate : highest,
    null,
  );

  return (
    <Tooltip open={open} onOpenChange={setOpen}>
      <TooltipTrigger
        render={
          <button
            type="button"
            className="inline-flex min-h-8 cursor-pointer items-center gap-1.5 rounded-md text-foreground whitespace-nowrap focus-visible:outline-2 focus-visible:outline-offset-[3px] focus-visible:outline-ring"
          />
        }
        aria-label={`${label}: ${items.map((item) => `${item.name} ${percent(item.rate)}`).join(", ")}`}
        aria-expanded={open}
        onPointerDown={(event) => {
          touch.current = event.pointerType === "touch";
          openOnPointerDown.current = open;
        }}
        onPointerEnter={(event) => {
          if (event.pointerType === "mouse") setOpen(true);
        }}
        onPointerLeave={(event) => {
          if (event.pointerType === "mouse") setOpen(false);
        }}
        onFocus={(event) => {
          if (event.currentTarget.matches(":focus-visible")) setOpen(true);
        }}
        onBlur={() => setOpen(false)}
        onClick={() => {
          if (touch.current) setOpen(!openOnPointerDown.current);
        }}
        onKeyDown={(event) => {
          touch.current = false;
          if (event.key === "Escape") setOpen(false);
        }}
      >
        <span className="pointer-events-none inline-flex pe-2" aria-hidden="true">
          {items.map((item) => (
            <span
              key={item.id}
              className="relative -me-[7px] grid size-6 place-items-center rounded-full bg-surface shadow-[0_0_0_1px_var(--surface)] [&_[data-slot=brand-icon]]:size-4"
            >
              {item.icon}
            </span>
          ))}
        </span>
        <span className="tabular-nums">{percent(summary)}</span>
      </TooltipTrigger>
      <TooltipContent
        side="bottom"
        align="start"
        className="grid min-w-[210px] max-w-[min(320px,calc(100vw-32px))] gap-2 bg-[var(--metric-tooltip)] p-3 text-foreground shadow-[0_10px_30px_var(--metric-tooltip-shadow)] animate-none! [&>[data-slot=tooltip-arrow]]:hidden"
      >
        <span className="text-xs text-muted-foreground">{label}</span>
        {items.map((item) => (
          <span key={item.id} className="flex items-center justify-between gap-4">
            <span className="flex min-w-0 items-center gap-2">
              <span aria-hidden="true" className="[&_[data-slot=brand-icon]]:size-4">
                {item.icon}
              </span>
              <span className="truncate">{item.name}</span>
            </span>
            <span className="tabular-nums">{percent(item.rate)}</span>
          </span>
        ))}
      </TooltipContent>
    </Tooltip>
  );
}
