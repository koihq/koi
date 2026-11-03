import { cn } from "cn";
import { useTranslation } from "react-i18next";
import type { VisibilityEngine } from "@/api/generated";
import { EngineIcon } from "@/components/blocks/engine-icon";
import type { AnswerState } from "@/lib/answer-state";
import { engines } from "@/lib/visibility";

const tones: Record<AnswerState, string> = {
  cited: "bg-foreground text-background",
  mentioned: "bg-foreground/15 text-foreground",
  miss: "text-muted-foreground ring-1 ring-inset ring-border",
  error: "text-destructive ring-1 ring-inset ring-destructive/40",
  pending: "text-muted-foreground/60",
  "not-collected": "text-muted-foreground/60",
  "not-requested": "text-muted-foreground/40",
};

/** One engine's outcome as a filled, tinted or outlined disc around its glyph. */
export function OutcomeMark({
  engine,
  state,
  className,
}: {
  engine: VisibilityEngine;
  state: AnswerState;
  className?: string;
}) {
  const { t } = useTranslation(["home", "visibility"]);
  const name = engines.find((item) => item.id === engine)?.name ?? engine;
  const label = t("home:detail.outcomeSummary", {
    engine: name,
    outcome: t(`visibility:outcomes.${state}`),
  });

  return (
    <span
      role="img"
      aria-label={label}
      title={label}
      className={cn(
        "inline-grid size-6 shrink-0 place-items-center rounded-full",
        tones[state],
        className,
      )}
    >
      <EngineIcon engine={engine} className="size-3.5" />
    </span>
  );
}

/** Key for the three outcomes that carry meaning. */
export function OutcomeLegend({ className }: { className?: string }) {
  const { t } = useTranslation(["home", "visibility"]);

  return (
    <ul
      aria-label={t("home:detail.legend")}
      className={cn(
        "flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground",
        className,
      )}
    >
      {(["cited", "mentioned", "miss"] as const).map((state) => (
        <li key={state} className="flex items-center gap-1.5">
          <span aria-hidden="true" className={cn("size-2.5 rounded-full", tones[state])} />
          {t(`visibility:outcomes.${state}`)}
        </li>
      ))}
    </ul>
  );
}
