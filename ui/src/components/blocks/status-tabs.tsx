import { Button } from "@/components/ui/button";

export type StatusTab<T extends string> = { value: T; label: string; count?: number };

/** List status as quiet text tabs, each with how many items it holds. */
export function StatusTabs<T extends string>({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: T;
  options: StatusTab<T>[];
  onChange: (value: T) => void;
}) {
  return (
    <fieldset className="flex flex-wrap gap-0.5" aria-label={label}>
      {options.map((option) => (
        <Button
          key={option.value}
          type="button"
          variant="ghost"
          size="pill"
          aria-pressed={value === option.value}
          className="gap-1.5 text-muted-foreground"
          onClick={() => onChange(option.value)}
        >
          {option.label}
          {option.count === undefined ? null : (
            <span className="text-muted-foreground tabular-nums">{option.count}</span>
          )}
        </Button>
      ))}
    </fieldset>
  );
}
