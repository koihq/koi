import {
  ChatGptIcon,
  GoogleGeminiIcon,
  GoogleIcon,
  PerplexityAiIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { cn } from "cn";
import type { VisibilityEngine } from "@/api/generated";
import { engines } from "@/lib/visibility";

const icons = {
  chatgpt: ChatGptIcon,
  perplexity: PerplexityAiIcon,
  gemini: GoogleGeminiIcon,
  google_ai_overviews: GoogleIcon,
} satisfies Record<VisibilityEngine, unknown>;

/** Monochrome engine glyph. Engines keep one identity everywhere, never a favicon. */
export function EngineIcon({
  engine,
  className,
  labelled = false,
}: {
  engine: VisibilityEngine;
  className?: string;
  /** Expose the engine name to assistive technology when the icon stands alone. */
  labelled?: boolean;
}) {
  const name = engines.find((item) => item.id === engine)?.name ?? engine;

  return (
    <HugeiconsIcon
      icon={icons[engine]}
      className={cn("size-4 shrink-0", className)}
      aria-hidden={labelled ? undefined : true}
      aria-label={labelled ? name : undefined}
      role={labelled ? "img" : undefined}
    />
  );
}
