import { useTranslation } from "react-i18next";

/** The promise that setup has an end, and how close it is. */
export function SetupProgress({ current, total }: { current: number; total: number }) {
  const { t } = useTranslation("brand");

  return (
    <div className="flex items-center gap-3">
      <p className="text-xs text-muted-foreground tabular-nums">
        {t("setup.step", { current, total })}
      </p>
      <span aria-hidden="true" className="flex gap-1">
        {Array.from({ length: total }, (_, index) => (
          <span
            // Steps never reorder, so their position is their identity.
            // biome-ignore lint/suspicious/noArrayIndexKey: a fixed row of segments
            key={index}
            className={`h-1 w-5 rounded-full transition-colors duration-200 ${index < current ? "bg-foreground" : "bg-muted"}`}
          />
        ))}
      </span>
    </div>
  );
}
