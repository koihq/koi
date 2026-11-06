import type { CSSProperties } from "react";

/**
 * A quiet wash of a brand color for the surface that is about that brand: a glow from the
 * top-left corner over the panel's own background. Null color keeps the panel neutral.
 */
export function brandGlow(
  color: string | null,
  base = "var(--surface)",
): CSSProperties | undefined {
  if (!color) return undefined;

  return {
    background: `radial-gradient(120% 90% at 0% 0%, color-mix(in oklch, ${color} 14%, transparent), transparent 60%), ${base}`,
  };
}

/** Fill for a selected row that stands for a brand. */
export function brandRowTint(color: string | null): CSSProperties | undefined {
  if (!color) return undefined;

  return { backgroundColor: `color-mix(in oklch, ${color} 14%, transparent)` };
}
