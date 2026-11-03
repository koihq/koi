import { ArrowDown01Icon, ArrowUp01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

export type SortDirection = "ascending" | "descending" | "none";

/** A column header that sorts its column; the arrow shows only while it is sorting. */
export function SortHeader({
  label,
  direction,
  onSort,
}: {
  label: string;
  direction: SortDirection;
  onSort: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSort}
      className="-mx-1 inline-flex items-center gap-1 rounded-control px-1 hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring aria-[pressed=true]:text-foreground"
      aria-pressed={direction !== "none"}
    >
      {label}
      {direction === "none" ? null : (
        <HugeiconsIcon
          icon={direction === "ascending" ? ArrowUp01Icon : ArrowDown01Icon}
          className="size-3"
          aria-hidden="true"
        />
      )}
    </button>
  );
}
