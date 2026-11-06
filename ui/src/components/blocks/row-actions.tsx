import type { IconSvgElement } from "@hugeicons/react";
import { HugeiconsIcon } from "@hugeicons/react";
import { cn } from "cn";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

/**
 * Row actions as direct icon buttons, evenly spaced, in a fixed order: state → edit →
 * toggle → delete. Delete is marked by its red hover, not by extra space.
 * Lists with a handful of actions skip the "more" menu so every action is one click away.
 * The row must carry Tailwind's `group` class (DataTable `interactive` rows already do).
 */
export function RowActions({ children }: { children: ReactNode }) {
  return (
    <TooltipProvider delay={300}>
      <div className="inline-flex items-center justify-end gap-1">{children}</div>
    </TooltipProvider>
  );
}

/**
 * One icon action. Hidden until the row is hovered or focused (always shown on touch),
 * unless `persistent`: state actions such as Confirm stay visible because they tell the
 * reader there is work to do.
 */
export function RowAction({
  label,
  icon,
  onClick,
  disabled,
  destructive = false,
  persistent = false,
  pressed,
  large = false,
}: {
  label: string;
  icon: IconSvgElement;
  onClick: () => void;
  disabled?: boolean;
  destructive?: boolean;
  persistent?: boolean;
  pressed?: boolean;
  /** For a page's own actions, beside its title instead of inside a row. */
  large?: boolean;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="circle-sm"
            aria-label={label}
            aria-pressed={pressed}
            disabled={disabled}
            onClick={onClick}
            className={cn(
              "text-muted-foreground hover:text-foreground",
              destructive && "hover:bg-destructive/10 hover:text-destructive",
              !persistent &&
                "opacity-0 transition-opacity duration-150 group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100 [@media(hover:none)]:opacity-100",
            )}
          />
        }
      >
        <HugeiconsIcon
          icon={icon}
          className={large ? "size-[18px]" : undefined}
          aria-hidden="true"
        />
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
