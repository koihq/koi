import { type ReactNode, useLayoutEffect, useMemo, useRef, useState } from "react";
import type { VisibilityEngine } from "@/api/generated";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Button } from "@/components/ui/button";
import { ENGINES } from "../-lib/data";
import type { ExplorationSearch } from "../-lib/exploration-fixtures";
import type { HomeBrand } from "../-lib/home-evidence";
import {
  dailySeries,
  type EvidenceRecord,
  type PromptRow,
  percent,
  points,
  promptRuns,
  summarize,
} from "../-lib/prompt-visibility";
import { trendPath } from "../-lib/trend-path";
import M from "../-locales/mvp.en.json";
import C from "../-locales/prompt-visibility.en.json";
import { AnswerBody, SourceList } from "./answer-panel";
import { HomeBrandText } from "./home-brand-text";
import { copy } from "./list-controls";
import { PromptEditor } from "./prompt-editor";

function formatRunDate(date: string) {
  return new Date(`${date}T12:00:00Z`).toLocaleDateString("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

function outcome(record?: EvidenceRecord) {
  if (!record) return { short: "?", label: C.notAvailable };
  if (record.stopped) return { short: "N", label: M.notCollected };

  switch (record.result.outcome) {
    case "cited":
      return { short: "C", label: M.cited };
    case "mentioned":
      return { short: "M", label: M.mentioned };
    case "miss":
      return { short: "—", label: M.miss };
    case "error":
      return { short: "!", label: M.error };
    default:
      return { short: "…", label: M.pending };
  }
}

function PromptVisibilityStrip({
  row,
  records,
  brand,
  days,
  onDaysChange,
}: {
  row: PromptRow;
  records: EvidenceRecord[];
  brand: HomeBrand;
  days: 7 | 30 | 90;
  onDaysChange: (days: 7 | 30 | 90) => void;
}) {
  const scope = { days, promptIds: [row.prompt.id] };
  const target = { id: brand, name: brand, measured: true };
  const stats = summarize(records, brand, scope, target);
  const series = dailySeries(records, brand, scope, target);
  const line = trendPath(
    series.map((point, index) =>
      point.rate === null
        ? null
        : {
            x: (index / Math.max(1, series.length - 1)) * 180,
            y: 4 + ((100 - point.rate) / 100) * 36,
          },
    ),
  );
  const change =
    stats.changePoints === null
      ? C.noComparison
      : copy(stats.changePoints > 0 ? C.up : stats.changePoints < 0 ? C.down : C.unchanged, {
          points: points(stats.changePoints),
        });

  return (
    <section className="hc-prompt-signal" aria-label={C.visibilityTitle}>
      <span className="hc-prompt-signal-label">{C.visibilityAllEngines}</span>
      <strong className="hc-prompt-signal-rate">{percent(stats.rate)}</strong>
      <span className="hc-prompt-signal-change">{change}</span>
      {line ? (
        <svg
          className="hc-prompt-signal-chart"
          viewBox="0 0 180 44"
          preserveAspectRatio="none"
          role="img"
          aria-label={copy(C.previewTrend, {
            days,
            start: percent(series.find((point) => point.rate !== null)?.rate ?? null),
            end: percent([...series].reverse().find((point) => point.rate !== null)?.rate ?? null),
          })}
        >
          <path d={line} />
        </svg>
      ) : (
        <span className="hc-prompt-signal-empty">{C.noAnswers}</span>
      )}
      <fieldset className="hc-prompt-signal-periods" aria-label={C.chartRange}>
        {([7, 30, 90] as const).map((value) => (
          <button
            key={value}
            type="button"
            aria-pressed={days === value}
            onClick={() => onDaysChange(value)}
          >
            {copy(C.periodLabel, { days: value })}
          </button>
        ))}
      </fieldset>
    </section>
  );
}

export function PromptDetail({
  row,
  records,
  brand,
  brandName,
  search,
  onSearch,
  onSave,
  back,
}: {
  row: PromptRow;
  records: EvidenceRecord[];
  brand: HomeBrand;
  brandName: string;
  search: ExplorationSearch;
  onSearch: (patch: Partial<ExplorationSearch>) => void;
  onSave: (text: string, id: string) => void;
  back: ReactNode;
}) {
  const [editing, setEditing] = useState(false);
  const [limit, setLimit] = useState(20);
  const [mobileReader, setMobileReader] = useState(Boolean(search.runId));
  const editTrigger = useRef<HTMLButtonElement | null>(null);
  const reader = useRef<HTMLElement | null>(null);
  const lastTrigger = useRef<HTMLButtonElement | null>(null);
  const selectedRunTrigger = useRef<HTMLButtonElement | null>(null);
  const runs = useMemo(() => promptRuns(records, row.prompt.id), [records, row.prompt.id]);
  const run = search.runId ? runs.find((item) => item.id === search.runId) : runs[0];
  const selectedIndex = run ? runs.indexOf(run) : -1;
  const visibleCount = Math.max(limit, Math.ceil((selectedIndex + 1) / 20) * 20);
  const engine: VisibilityEngine = search.responseEngine ?? "chatgpt";
  const selected = run?.results[engine];
  const engineInfo = ENGINES.find((item) => item.id === engine);

  useLayoutEffect(() => {
    if (mobileReader && window.matchMedia("(max-width: 959px)").matches)
      reader.current?.focus({ preventScroll: true });
  }, [mobileReader]);

  return (
    <section className="hc-prompt-detail">
      {back}
      <div className="hc-section-heading hc-prompt-title">
        <h1 id="hc-page-heading" tabIndex={-1}>
          <HomeBrandText text={row.prompt.text} />
        </h1>
        {!row.past && (
          <Button ref={editTrigger} variant="ghost" size="sm" onClick={() => setEditing(true)}>
            {M.editPrompt}
          </Button>
        )}
      </div>
      {row.past && <p className="hc-caption">{C.pastPrompt}</p>}
      <PromptVisibilityStrip
        row={row}
        records={records}
        brand={brand}
        days={search.days ?? 30}
        onDaysChange={(days) => onSearch({ days })}
      />
      <section className="hc-responses" aria-labelledby="hc-responses-heading">
        <h2 id="hc-responses-heading" className="sr-only">
          {C.responses}
        </h2>
        {!runs.length ? (
          <p className="hc-empty">{C.noRuns}</p>
        ) : (
          <div className="hc-response-layout" data-reading={mobileReader || undefined}>
            <div className="hc-response-history">
              <h3>{C.runHistory}</h3>
              <div className="hc-run-columns" aria-hidden="true">
                <span />
                {ENGINES.map((item) => (
                  <abbr key={item.id} title={item.name}>
                    {item.short}
                  </abbr>
                ))}
              </div>
              <ol aria-label={C.runHistory}>
                {runs.slice(0, visibleCount).map((item) => (
                  <li key={item.id}>
                    <button
                      ref={run?.id === item.id ? selectedRunTrigger : undefined}
                      type="button"
                      aria-pressed={run?.id === item.id}
                      aria-label={`${formatRunDate(item.date)}. ${ENGINES.map(
                        (engine) => `${engine.name}: ${outcome(item.results[engine.id]).label}`,
                      ).join("; ")}`}
                      onClick={(event) => {
                        lastTrigger.current = event.currentTarget;
                        setMobileReader(true);
                        onSearch({ runId: item.id });
                      }}
                    >
                      <time dateTime={item.date}>{formatRunDate(item.date)}</time>
                      {ENGINES.map((engine) => (
                        <span
                          key={engine.id}
                          aria-hidden="true"
                          data-outcome={
                            !item.results[engine.id]
                              ? "unavailable"
                              : item.results[engine.id]?.stopped
                                ? "not-collected"
                                : (item.results[engine.id]?.result.outcome ?? "pending")
                          }
                        >
                          {outcome(item.results[engine.id]).short}
                        </span>
                      ))}
                    </button>
                  </li>
                ))}
              </ol>
              <p className="hc-run-legend">{C.runLegend}</p>
              {runs.length > visibleCount && (
                <Button variant="ghost" size="sm" onClick={() => setLimit(visibleCount + 20)}>
                  {C.olderRuns}
                </Button>
              )}
            </div>
            <section
              className="hc-response-main"
              ref={reader}
              tabIndex={-1}
              aria-label={C.responses}
            >
              <output className="sr-only" aria-live="polite">
                {run &&
                  `${formatRunDate(run.date)}, ${engineInfo?.name}: ${outcome(selected).label}`}
              </output>
              <Button
                variant="ghost"
                size="sm"
                className="hc-back-responses"
                onClick={() => {
                  setMobileReader(false);
                  requestAnimationFrame(() =>
                    (lastTrigger.current ?? selectedRunTrigger.current)?.focus(),
                  );
                }}
              >
                {C.backRuns}
              </Button>
              {run ? (
                <>
                  <time className="hc-response-date" dateTime={run.date}>
                    {formatRunDate(run.date)}
                  </time>
                  <fieldset className="hc-response-engines" aria-label={M.engine}>
                    {ENGINES.map((item) => (
                      <button
                        type="button"
                        key={item.id}
                        aria-pressed={engine === item.id}
                        onClick={() => onSearch({ responseEngine: item.id })}
                      >
                        <BrandPill name={item.name} domain={item.domain} />
                        <span className="hc-engine-outcome">
                          {outcome(run.results[item.id]).label}
                        </span>
                      </button>
                    ))}
                  </fieldset>
                  <div className="hc-response-inspection">
                    <div className="hc-response-reader">
                      {selected ? (
                        <>
                          <div className="hc-answer-meta">
                            {engineInfo && (
                              <BrandPill name={engineInfo.name} domain={engineInfo.domain} />
                            )}
                            <span>{M.synthetic}</span>
                          </div>
                          <div className="hc-answer-content">
                            <AnswerBody
                              result={selected.result}
                              brandName={brandName}
                              stopped={selected.stopped}
                            />
                          </div>
                        </>
                      ) : (
                        <p>{C.noRunResponse}</p>
                      )}
                    </div>
                    {selected && (
                      <aside className="hc-response-sources">
                        <SourceList
                          result={selected.result}
                          sample
                          emptyMessage={
                            selected.stopped
                              ? C.sourcesNotCollected
                              : selected.result.outcome === null
                                ? C.sourcesPending
                                : selected.result.outcome === "error"
                                  ? C.sourcesUnavailable
                                  : undefined
                          }
                        />
                      </aside>
                    )}
                  </div>
                </>
              ) : (
                <div className="hc-run-missing">
                  <p>{C.runMissing}</p>
                  <Button variant="ghost" size="sm" onClick={() => onSearch({ runId: undefined })}>
                    {C.latestRun}
                  </Button>
                </div>
              )}
            </section>
          </div>
        )}
      </section>
      {editing && (
        <PromptEditor
          prompt={row.prompt}
          onClose={() => setEditing(false)}
          onSave={(text) => onSave(text, row.prompt.id)}
          finalFocus={() => editTrigger.current}
        />
      )}
    </section>
  );
}
