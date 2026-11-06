import { Cancel01Icon, Search01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

/**
 * Search as an icon until it is needed. Opens into a field; stays open while it holds a
 * query; Escape clears it and closes.
 */
export function SearchToggle({
  label,
  clearLabel,
  value,
  onChange,
}: {
  label: string;
  clearLabel: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const [opened, setOpened] = useState(false);
  const trigger = useRef<HTMLButtonElement>(null);
  const open = opened || value !== "";

  function close() {
    onChange("");
    setOpened(false);
    requestAnimationFrame(() => trigger.current?.focus());
  }

  if (!open) {
    return (
      <Button
        ref={trigger}
        type="button"
        variant="ghost"
        size="circle-sm"
        aria-label={label}
        title={label}
        className="text-muted-foreground"
        onClick={() => setOpened(true)}
      >
        <HugeiconsIcon icon={Search01Icon} aria-hidden="true" />
      </Button>
    );
  }

  return (
    <div className="relative w-full sm:w-56">
      <Input
        type="search"
        autoFocus
        value={value}
        placeholder={label}
        aria-label={label}
        className="pe-8 [&::-webkit-search-cancel-button]:appearance-none"
        onChange={(event) => onChange(event.target.value)}
        onBlur={() => {
          if (value === "") setOpened(false);
        }}
        onKeyDown={(event) => {
          if (event.key === "Escape") close();
        }}
      />
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-label={clearLabel}
        className="absolute end-1 top-1/2 -translate-y-1/2 text-muted-foreground"
        onClick={close}
      >
        <HugeiconsIcon icon={Cancel01Icon} aria-hidden="true" />
      </Button>
    </div>
  );
}
