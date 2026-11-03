import { cn } from "cn";
import type { ReactNode } from "react";
import { BrandIcon } from "@/components/blocks/brand-icon";

const sizes = {
  md: { icon: "size-6", gap: "gap-2.5", text: "text-body font-semibold" },
  sm: { icon: "size-4.5", gap: "gap-2", text: "text-body" },
} as const;

/**
 * A brand as the subject of a row: its coin and its name, no container. The row's own
 * hover and selection frame it. Use `BrandPill` only where a brand sits inline among
 * other content and needs to stand apart.
 */
export function BrandLabel({
  name,
  domain,
  lookupLogo = true,
  size = "md",
  detail,
  className,
}: {
  name: string;
  domain?: string;
  /** Off where the brand's own identity is its initial (the user's brands). */
  lookupLogo?: boolean;
  size?: keyof typeof sizes;
  /** Quiet trailing context, such as a rate. */
  detail?: ReactNode;
  className?: string;
}) {
  const style = sizes[size];

  return (
    <span
      data-slot="brand-label"
      className={cn("inline-flex min-w-0 max-w-full items-center", style.gap, className)}
    >
      <BrandIcon name={name} domain={domain} lookupLogo={lookupLogo} className={style.icon} />
      <span className={cn("min-w-0 truncate text-foreground", style.text)}>{name}</span>
      {detail ? (
        <span className="shrink-0 text-xs text-muted-foreground tabular-nums">{detail}</span>
      ) : null}
    </span>
  );
}
