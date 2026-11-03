import { type ReactNode, useRef, useState } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

export type PromptBreakdownItem = {
  id: string;
  name: string;
  rate: number | null;
  icon: ReactNode;
};

const percent = (rate: number | null) => (rate === null ? "—" : `${rate.toFixed(1)}%`);

/** Cell content that reveals a per-item breakdown on hover, focus or tap. */
export function PromptRateBreakdown({
  label,
  items,
  emptyLabel = "—",
  openOnHover = true,
  children,
}: {
  label: string;
  items: PromptBreakdownItem[];
  emptyLabel?: string;
  /** Open on mouse hover. Off when the cell explains itself inline on hover. */
  openOnHover?: boolean;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const touch = useRef(false);
  const openOnPointerDown = useRef(false);

  if (!items.length) return <span className="text-xs text-muted-foreground">{emptyLabel}</span>;

  return (
    <Tooltip
      open={open}
      onOpenChange={(next, details) => {
        if (next && !openOnHover && details.reason === "trigger-hover") return;
        setOpen(next);
      }}
    >
      <TooltipTrigger
        render={
          <button
            type="button"
            className="inline-flex min-h-8 cursor-default items-center gap-2 rounded-md text-start text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          />
        }
        aria-label={`${label}: ${items.map((item) => `${item.name} ${percent(item.rate)}`).join(", ")}`}
        aria-expanded={open}
        onPointerDown={(event) => {
          touch.current = event.pointerType === "touch";
          openOnPointerDown.current = open;
        }}
        onPointerEnter={(event) => {
          if (openOnHover && event.pointerType === "mouse") setOpen(true);
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
        {children}
      </TooltipTrigger>
      <TooltipContent
        side="bottom"
        align="start"
        className="surface-float grid w-56 max-w-[calc(100vw-2rem)] gap-2 bg-transparent! p-3 text-foreground animate-none! [&>[data-slot=tooltip-arrow]]:hidden"
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
