import { cn } from "cn";
import { type ReactNode, useState } from "react";
import type { VisibilityEngine } from "@/api/generated";
import { EngineIcon } from "@/components/blocks/engine-icon";
import { formatRate } from "@/lib/format";
import { engines } from "@/lib/visibility";

const TICKS = 5;
const ticks = Array.from({ length: TICKS }, (_, index) => index);

const shortNames: Record<VisibilityEngine, string> = {
  chatgpt: "ChatGPT",
  gemini: "Gemini",
  google_ai_mode: "AI Mode",
  google_ai_overviews: "AI Overviews",
};

/** Ticks filled for a rate: any nonzero rate shows at least one, so presence is never hidden. */
export function filledTicks(rate: number | null) {
  if (rate === null || rate <= 0) return 0;

  return Math.min(TICKS, Math.ceil(rate / (100 / TICKS)));
}

/**
 * A prompt's visibility split by engine: four groups of five ticks, one group per engine in
 * the fixed engine order, each filled by that engine's rate. Decorative; the exact values
 * live in the cell's accessible label and tooltip.
 */
export function EngineMeter({
  rateFor,
  active,
  onActiveChange,
}: {
  rateFor: (engine: VisibilityEngine) => number | null;
  active: VisibilityEngine | null;
  onActiveChange: (engine: VisibilityEngine | null) => void;
}) {
  return (
    <span
      aria-hidden="true"
      className="inline-flex shrink-0 items-center gap-1.5"
      onPointerLeave={() => onActiveChange(null)}
    >
      {engines.map((engine) => {
        const filled = filledTicks(rateFor(engine.id));

        return (
          <span
            key={engine.id}
            data-engine={engine.id}
            className={cn(
              "inline-flex gap-0.5 py-1 transition-opacity duration-150",
              active !== null && active !== engine.id && "opacity-35",
            )}
            onPointerEnter={(event) => {
              if (event.pointerType === "mouse") onActiveChange(engine.id);
            }}
          >
            {ticks.map((tick) => (
              <span
                key={tick}
                className={cn(
                  "h-3.5 w-0.75 rounded-full",
                  tick < filled ? "bg-foreground" : "bg-foreground/12",
                )}
              />
            ))}
          </span>
        );
      })}
    </span>
  );
}

/**
 * The visibility cell: overall rate and the engine meter. The slot after the meter names
 * the engine under the pointer with its rate; otherwise it shows `change` while the row is
 * hovered or focused (always on touch).
 */
export function EngineMeterCell({
  rate,
  rateFor,
  locale,
  change,
}: {
  rate: number;
  rateFor: (engine: VisibilityEngine) => number | null;
  locale: string;
  change: ReactNode;
}) {
  const [active, setActive] = useState<VisibilityEngine | null>(null);
  const activeRate = active ? rateFor(active) : null;

  return (
    <>
      <span className="w-12 whitespace-nowrap font-semibold tabular-nums @5xl:text-end">
        {formatRate(rate, locale)}
      </span>
      <EngineMeter rateFor={rateFor} active={active} onActiveChange={setActive} />
      <span className="hidden items-center @5xl:inline-flex @5xl:min-w-28 [@media(hover:none)]:inline-flex">
        {active ? (
          <span className="inline-flex items-center gap-1.5 text-xs text-foreground">
            <EngineIcon engine={active} className="size-3.5" />
            {shortNames[active]}
            <span className="text-muted-foreground tabular-nums">
              {activeRate === null ? "—" : formatRate(activeRate, locale)}
            </span>
          </span>
        ) : (
          <span className="opacity-0 transition-opacity duration-150 group-hover:opacity-100 group-focus-within:opacity-100 [@media(hover:none)]:opacity-100">
            {change}
          </span>
        )}
      </span>
    </>
  );
}
