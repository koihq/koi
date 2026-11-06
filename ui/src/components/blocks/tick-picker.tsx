import NumberFlow from "@number-flow/react";
import { cn } from "cn";

const MINOR_TICKS = 4;

/**
 * Picks one of a few uneven amounts on a ruler, with the amount large above it. Each amount sits
 * on a long tick, evenly spaced whatever its value, so 15 and 500 are equally easy to reach.
 * A transparent native range does the dragging, clicking, keyboard and screen reader work.
 * Takes plain values and copy, so it also works outside the app.
 */
export function TickPicker({
  options,
  value,
  onChange,
  unit,
  label,
  locale,
  mark,
  markLabel,
  disabled = false,
}: {
  options: number[];
  value: number;
  onChange: (value: number) => void;
  unit: string;
  label: string;
  locale?: string;
  /** An amount worth finding again, such as the current plan. */
  mark?: number;
  markLabel?: string;
  disabled?: boolean;
}) {
  const last = options.length - 1;
  const selected = Math.max(0, options.indexOf(value));
  const ticks = Array.from({ length: last * (MINOR_TICKS + 1) + 1 }, (_, index) => index);

  return (
    <div className={cn("grid gap-3", disabled && "opacity-50")}>
      <p className="text-metric font-medium tabular-nums">
        <NumberFlow value={value} locales={locale} />
        <span className="ms-1.5 text-section font-normal text-muted-foreground">{unit}</span>
      </p>

      <div className="relative rounded-sm px-2 has-focus-visible:outline-2 has-focus-visible:outline-offset-4 has-focus-visible:outline-ring">
        <div aria-hidden="true" className="flex h-8 items-end justify-between">
          {ticks.map((tick) => {
            const stop = tick / (MINOR_TICKS + 1);
            const major = Number.isInteger(stop);

            return (
              <span
                key={tick}
                className={cn(
                  "w-0.5 rounded-full transition-colors duration-150",
                  stop === selected ? "h-8" : major ? "h-5" : "h-3",
                  stop <= selected ? "bg-foreground" : "bg-foreground/15",
                )}
              />
            );
          })}
        </div>

        <div aria-hidden="true" className="relative mt-2 h-7 text-xs tabular-nums">
          {options.map((option, index) => (
            <span
              key={option}
              title={option === mark ? markLabel : undefined}
              className={cn(
                "absolute top-0 grid -translate-x-1/2 justify-items-center gap-1 transition-colors duration-150",
                index === selected ? "font-semibold text-foreground" : "text-muted-foreground",
              )}
              style={{ left: `${(index / last) * 100}%` }}
            >
              {option}
              {option === mark && <span className="size-1 rounded-full bg-current" />}
            </span>
          ))}
        </div>

        <input
          type="range"
          min={0}
          max={last}
          step={1}
          value={selected}
          disabled={disabled}
          aria-label={label}
          aria-valuetext={`${value} ${unit}`}
          onChange={(event) => onChange(options[Number(event.target.value)])}
          className="absolute inset-0 m-0 size-full cursor-pointer appearance-none bg-transparent opacity-0 disabled:cursor-not-allowed [&::-moz-range-thumb]:size-4 [&::-webkit-slider-thumb]:size-4 [&::-webkit-slider-thumb]:appearance-none"
        />
      </div>
    </div>
  );
}
