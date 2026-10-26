import { cn } from "cn";
import type { ComponentProps } from "react";
import { BrandIcon } from "@/components/blocks/brand-icon";

export function BrandPill({
  className,
  iconOnly = false,
  ...props
}: ComponentProps<typeof BrandIcon> & { iconOnly?: boolean }) {
  return (
    <span
      data-slot="brand-pill"
      data-icon-only={iconOnly || undefined}
      title={iconOnly ? props.name : undefined}
      className={cn(
        "inline-flex max-w-full items-center gap-1 align-middle text-xs font-medium leading-normal text-foreground",
        !iconOnly && "rounded-full bg-muted py-0.5 pl-1 pr-2",
        className,
      )}
    >
      <BrandIcon {...props} className="size-4" />
      <span className={iconOnly ? "sr-only" : "min-w-0 [overflow-wrap:anywhere]"}>
        {props.name}
      </span>
    </span>
  );
}
