import { useTranslation } from "react-i18next";
import type { PromptHistoryRunPage } from "@/api/generated";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { OutcomeMark } from "@/components/blocks/outcome-mark";
import { getAnswerState } from "@/lib/answer-state";
import { formatDay, formatTime } from "@/lib/format";
import { engines } from "@/lib/visibility";

type RunItem = PromptHistoryRunPage["items"][number];

const VISIBLE_RIVALS = 3;

/** Saved runs as quiet rows: when, how each engine answered, and who else appeared. */
export function RunHistory({
  runs,
  rivalDomain,
  onOpen,
}: {
  runs: RunItem[];
  rivalDomain: (name: string) => string | undefined;
  onOpen: (runId: string, trigger: HTMLButtonElement) => void;
}) {
  const { t, i18n } = useTranslation(["home", "visibility"]);

  return (
    <ul className="list-rows -mx-3 mt-4 grid gap-0.5">
      {runs.map((item) => {
        const rivals = [
          ...new Set(item.results.flatMap((cell) => cell.appearances.map((entry) => entry.name))),
        ];
        const day = formatDay(item.run.createdAt, i18n.language);
        const time = formatTime(item.run.createdAt, i18n.language);

        return (
          <li key={item.run.id}>
            <button
              type="button"
              aria-label={t("home:detail.openRun", { date: `${day}, ${time}` })}
              onClick={(event) => onOpen(item.run.id, event.currentTarget)}
              className="grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-x-6 gap-y-2.5 rounded-control px-3 py-3 text-start text-body hover:bg-signal-pill focus-visible:bg-signal-pill focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring sm:grid-cols-[9rem_minmax(0,1fr)_12rem]"
            >
              <time
                dateTime={item.run.createdAt.toISOString()}
                className="font-semibold tabular-nums"
              >
                {day}
                <span className="ms-2 text-muted-foreground">{time}</span>
              </time>
              <span className="col-span-full row-start-2 flex gap-1.5 sm:col-span-1 sm:row-start-auto">
                {engines.map((engine) => (
                  <OutcomeMark
                    key={engine.id}
                    engine={engine.id}
                    state={getAnswerState(
                      item.run,
                      engine.id,
                      item.results.find((entry) => entry.engine === engine.id),
                    )}
                  />
                ))}
              </span>
              <span
                className="flex items-center justify-end gap-1"
                title={rivals.join(", ") || undefined}
              >
                {rivals.length === 0 ? (
                  <span className="text-xs text-muted-foreground">{t("home:detail.noRivals")}</span>
                ) : (
                  <>
                    <span className="sr-only">{rivals.join(", ")}</span>
                    {rivals.slice(0, VISIBLE_RIVALS).map((name) => (
                      <BrandIcon
                        key={name}
                        name={name}
                        domain={rivalDomain(name)}
                        className="size-5"
                      />
                    ))}
                    {rivals.length > VISIBLE_RIVALS && (
                      <span
                        aria-hidden="true"
                        className="ms-1 text-xs text-muted-foreground tabular-nums"
                      >
                        {t("home:detail.moreRivals", { count: rivals.length - VISIBLE_RIVALS })}
                      </span>
                    )}
                  </>
                )}
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
