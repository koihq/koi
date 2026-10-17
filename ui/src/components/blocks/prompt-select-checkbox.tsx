import { useEffect, useRef } from "react";

export function PromptSelectCheckbox({
  label,
  checked,
  indeterminate = false,
  onChange,
}: {
  label: string;
  checked: boolean;
  indeterminate?: boolean;
  onChange: (checked: boolean) => void;
}) {
  const input = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (input.current) input.current.indeterminate = indeterminate;
  }, [indeterminate]);

  return (
    <span className="prompt-check-wrap relative inline-grid size-7 place-items-center align-middle">
      <input
        ref={input}
        type="checkbox"
        className="peer absolute inset-0 z-10 m-0 size-full cursor-pointer appearance-none opacity-0"
        aria-label={label}
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
      />
      <span
        className="pointer-events-none grid size-4 place-items-center rounded-[3px] border border-muted-foreground peer-checked:border-foreground peer-checked:bg-foreground peer-checked:text-background peer-indeterminate:border-foreground peer-indeterminate:bg-foreground peer-indeterminate:text-background peer-focus-visible:outline-2 peer-focus-visible:outline-offset-[3px] peer-focus-visible:outline-ring forced-colors:forced-color-adjust-none forced-colors:border-[CanvasText] forced-colors:peer-checked:bg-[CanvasText] forced-colors:peer-checked:text-[Canvas] forced-colors:peer-indeterminate:bg-[CanvasText] forced-colors:peer-indeterminate:text-[Canvas]"
        aria-hidden="true"
      >
        {(checked || indeterminate) && (
          <svg className="size-3.5" viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <path
              d={indeterminate ? "M4 8h8" : "M3.5 8l3 3 6-6"}
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        )}
      </span>
    </span>
  );
}
