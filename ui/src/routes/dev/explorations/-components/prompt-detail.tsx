import NumberFlow, { useCanAnimate } from "@number-flow/react";
import { type ReactNode, useMemo, useRef, useState } from "react";
import type { VisibilityEngine } from "@/api/generated";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ENGINES } from "../-lib/data";
import type { ExplorationSearch, PreviewBrand } from "../-lib/exploration-fixtures";
import type { HomeBrand } from "../-lib/home-evidence";
import {
  comparedBrands,
  dailySeries,
  type EvidenceRecord,
  type PromptRow,
  percent,
  points,
  promptRuns,
  runCompetitors,
  summarize,
} from "../-lib/prompt-visibility";
import M from "../-locales/mvp.en.json";
import C from "../-locales/prompt-visibility.en.json";
import { AnswerBody, SourceList } from "./answer-panel";
import { HomeBrandText } from "./home-brand-text";
import { copy } from "./list-controls";
import { PromptEditor } from "./prompt-editor";
import { SignalChart, type SignalInspectionInput } from "./signal-chart";

function formatRunDate(date: string) {
  return new Date(`${date}T12:00:00Z`).toLocaleDateString("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

function formatRunTime(timestamp?: string) {
  if (!timestamp) return undefined;

  return new Date(timestamp).toLocaleTimeString("en", {
    hour: "numeric",
    minute: "2-digit",
    timeZone: "Europe/Rome",
  });
}

function outcome(record?: EvidenceRecord) {
  if (!record) return { label: C.notAvailable, value: "unavailable" };
  if (record.stopped) return { label: M.notCollected, value: "not-collected" };

  switch (record.result.outcome) {
    case "cited":
      return { label: M.cited, value: "cited" };
    case "mentioned":
      return { label: M.mentioned, value: "mentioned" };
    case "miss":
      return { label: M.miss, value: "miss" };
    case "error":
      return { label: M.error, value: "error" };
    default:
      return { label: M.pending, value: "pending" };
  }
}

function VisibilityHistory({
  row,
  records,
  brand,
  data,
  days,
  compareId,
  onDaysChange,
  onCompare,
}: {
  row: PromptRow;
  records: EvidenceRecord[];
  brand: HomeBrand;
  data: PreviewBrand;
  days: 7 | 30 | 90;
  compareId?: string;
  onDaysChange: (days: 7 | 30 | 90) => void;
  onCompare: (id?: string) => void;
}) {
  const scope = { days, promptIds: [row.prompt.id] };
  const brands = comparedBrands(brand, data);
  const you = brands[0];
  const ownStats = summarize(records, brand, scope, you);
  const rivals = brands
    .slice(1)
    .map((item) => ({ ...item, stats: summarize(records, brand, scope, item) }))
    .sort((a, b) => (b.stats.rate ?? -1) - (a.stats.rate ?? -1) || a.name.localeCompare(b.name));
  const selected = rivals.find((item) => item.id === compareId && item.stats.rate !== null);
  const series = dailySeries(records, brand, scope, you);
  const comparison = selected ? dailySeries(records, brand, scope, selected) : undefined;
  const [inspection, setInspection] = useState<{
    scopeKey: string;
    index: number;
    input: SignalInspectionInput;
  } | null>(null);
  const scopeKey = `${row.prompt.id}:${days}:${selected?.id ?? ""}`;
  const activeInspection = inspection?.scopeKey === scopeKey ? inspection : null;
  const canAnimateNumbers = useCanAnimate();
  const metricRate = activeInspection
    ? (series[activeInspection.index]?.rate ?? null)
    : ownStats.rate;

  return (
    <section className="hc-prompt-history-overview" aria-label={C.visibilityTitle}>
      <div className="hc-prompt-trend-panel">
        <div className="hc-prompt-trend-heading">
          <div>
            <h2>{C.visibilityTitle}</h2>
            <div className="hc-prompt-trend-metric hc-rate">
              {metricRate === null ? (
                "—"
              ) : (
                <>
                  <NumberFlow
                    className="hc-signal-number"
                    value={Number(metricRate.toFixed(1))}
                    format={{ maximumFractionDigits: 1 }}
                    animated={canAnimateNumbers && activeInspection?.input === "pointer"}
                  />
                  <span>{C.percentSign}</span>
                </>
              )}
            </div>
            <p>
              {copy(C.sampleCount, {
                mentioned: ownStats.mentioned,
                completed: ownStats.completed,
              })}
            </p>
          </div>
          <fieldset className="hc-prompt-periods" aria-label={C.chartRange}>
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
        </div>
        {ownStats.rate === null ? (
          <p className="hc-prompt-chart-empty">{C.noAnswers}</p>
        ) : (
          <SignalChart
            series={series}
            brand={you}
            comparison={comparison}
            competitor={selected}
            scopeKey={scopeKey}
            animate={false}
            selectedIndex={activeInspection?.index ?? null}
            inspectionInput={activeInspection?.input ?? null}
            onSelectedIndexChange={(index, input) =>
              setInspection(index === null ? null : { scopeKey, index, input })
            }
          />
        )}
        <div className="hc-prompt-trend-dates">
          <time dateTime={series[0].date}>{series[0].date}</time>
          <span aria-hidden="true">{C.dateRangeSeparator}</span>
          <time dateTime={series[series.length - 1].date}>{series[series.length - 1].date}</time>
        </div>
      </div>
      <aside className="hc-prompt-competitors" aria-labelledby="hc-prompt-competitors-heading">
        <h2 id="hc-prompt-competitors-heading">{C.compareWith}</h2>
        <div className="hc-prompt-competitor-list">
          {rivals.map((item) => {
            const available = item.stats.rate !== null;
            const difference =
              item.stats.rate !== null && ownStats.rate !== null
                ? item.stats.rate - ownStats.rate
                : null;
            const explanation =
              difference === null
                ? C.noObservations
                : difference > 0
                  ? copy(C.competitorAhead, { points: points(difference) })
                  : difference < 0
                    ? copy(C.competitorBehind, { points: points(Math.abs(difference)) })
                    : C.level;

            return (
              <button
                key={item.id}
                type="button"
                className="hc-prompt-competitor"
                aria-pressed={selected?.id === item.id}
                aria-disabled={!available}
                aria-label={`${copy(C.compare, { name: item.name })}. ${percent(item.stats.rate)}. ${explanation}`}
                title={available ? explanation : C.noObservations}
                onClick={() => {
                  if (available) onCompare(selected?.id === item.id ? undefined : item.id);
                }}
              >
                <BrandPill name={item.name} domain={item.domain} />
                <span className="hc-prompt-competitor-rate">{percent(item.stats.rate)}</span>
                <span className="hc-prompt-competitor-delta">{explanation}</span>
              </button>
            );
          })}
          {!rivals.length && <p className="hc-caption">{C.noCompetitors}</p>}
        </div>
      </aside>
    </section>
  );
}

export function PromptDetail({
  row,
  records,
  brand,
  brandName,
  data,
  search,
  onSearch,
  onSave,
  back,
}: {
  row: PromptRow;
  records: EvidenceRecord[];
  brand: HomeBrand;
  brandName: string;
  data: PreviewBrand;
  search: ExplorationSearch;
  onSearch: (patch: Partial<ExplorationSearch>) => void;
  onSave: (text: string, id: string) => void;
  back: ReactNode;
}) {
  const [editing, setEditing] = useState(false);
  const [limit, setLimit] = useState(20);
  const editTrigger = useRef<HTMLButtonElement | null>(null);
  const selectedRunTrigger = useRef<HTMLButtonElement | null>(null);
  const dialogTrigger = useRef<HTMLButtonElement | null>(null);
  const runs = useMemo(() => promptRuns(records, row.prompt.id), [records, row.prompt.id]);
  const run = search.runId ? runs.find((item) => item.id === search.runId) : undefined;
  const engine: VisibilityEngine = search.responseEngine ?? "chatgpt";
  const selected = run?.results[engine];
  const engineInfo = ENGINES.find((item) => item.id === engine);
  const visibleRuns = runs.slice(0, limit);

  function openRun(runId: string, trigger: HTMLButtonElement, responseEngine?: VisibilityEngine) {
    dialogTrigger.current = trigger;
    onSearch({ runId, ...(responseEngine ? { responseEngine } : {}) });
  }

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
      <VisibilityHistory
        row={row}
        records={records}
        brand={brand}
        data={data}
        days={search.days ?? 30}
        compareId={search.compareId}
        onDaysChange={(days) => onSearch({ days })}
        onCompare={(compareId) => onSearch({ compareId })}
      />

      <section className="hc-run-history" aria-labelledby="hc-run-history-heading">
        <div className="hc-run-history-heading">
          <h2 id="hc-run-history-heading">{C.runHistory}</h2>
          <span>{copy(C.runsCount, { count: runs.length })}</span>
        </div>
        {!runs.length ? (
          <p className="hc-empty">{C.noRuns}</p>
        ) : (
          <>
            <p className="hc-run-scroll-hint">{C.scrollRunTable}</p>
            <Table className="hc-run-table">
              <TableHeader>
                <TableRow>
                  <TableHead>{C.runDate}</TableHead>
                  <TableHead>{C.engineResults}</TableHead>
                  <TableHead>{C.competitorsFound}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {visibleRuns.map((item) => {
                  const competitors = runCompetitors(item.results).map((appearance) => {
                    const competitor =
                      data.competitors.find((candidate) => candidate.id === appearance.id) ??
                      data.competitors.find(
                        (candidate) =>
                          candidate.name.toLocaleLowerCase() ===
                          appearance.name.toLocaleLowerCase(),
                      );

                    return {
                      ...appearance,
                      name: competitor?.name ?? appearance.name,
                      domain: competitor?.domain || undefined,
                    };
                  });
                  const time = formatRunTime(item.timestamp);

                  return (
                    <TableRow key={item.id}>
                      <TableCell>
                        <button
                          ref={search.runId === item.id ? selectedRunTrigger : undefined}
                          type="button"
                          className="hc-run-date-button"
                          onClick={(event) => openRun(item.id, event.currentTarget)}
                        >
                          <time dateTime={item.timestamp ?? item.date}>
                            {formatRunDate(item.date)}
                          </time>
                          {time && <span>{copy(C.runAt, { time })}</span>}
                        </button>
                      </TableCell>
                      <TableCell>
                        <fieldset className="hc-run-statuses" aria-label={C.engineResults}>
                          {ENGINES.map((itemEngine) => {
                            const status = outcome(item.results[itemEngine.id]);

                            return (
                              <button
                                key={itemEngine.id}
                                type="button"
                                className="hc-run-status"
                                data-outcome={status.value}
                                title={`${itemEngine.name}: ${status.label}`}
                                aria-label={`${itemEngine.name}: ${status.label}. ${C.inspectRun}`}
                                onClick={(event) =>
                                  openRun(item.id, event.currentTarget, itemEngine.id)
                                }
                              >
                                <BrandPill
                                  name={itemEngine.name}
                                  domain={itemEngine.domain}
                                  iconOnly
                                />
                                <span>{status.label}</span>
                              </button>
                            );
                          })}
                        </fieldset>
                      </TableCell>
                      <TableCell>
                        {competitors.length ? (
                          <div className="hc-run-competitors">
                            {competitors.map((competitor) => (
                              <BrandPill
                                key={competitor.id}
                                name={competitor.name}
                                domain={competitor.domain}
                              />
                            ))}
                          </div>
                        ) : (
                          <span className="hc-run-no-competitors">{C.noCompetitorsFound}</span>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </>
        )}
        {runs.length > visibleRuns.length && (
          <Button variant="ghost" size="sm" onClick={() => setLimit((value) => value + 20)}>
            {C.olderRuns}
          </Button>
        )}
      </section>

      <Dialog
        open={Boolean(search.runId)}
        onOpenChange={(open) => {
          if (!open) onSearch({ runId: undefined });
        }}
      >
        <DialogContent
          className="hc-prompt-run-dialog"
          finalFocus={() => dialogTrigger.current ?? selectedRunTrigger.current}
        >
          {run ? (
            <>
              <DialogHeader className="hc-prompt-run-dialog-header">
                <DialogTitle>
                  <HomeBrandText text={row.prompt.text} />
                </DialogTitle>
                <DialogDescription>
                  <span>
                    {formatRunTime(run.timestamp)
                      ? copy(C.runDialogWhen, {
                          date: formatRunDate(run.date),
                          time: formatRunTime(run.timestamp) ?? "",
                        })
                      : formatRunDate(run.date)}
                  </span>
                  <span>
                    {copy(C.enginesRan, {
                      count: ENGINES.filter((item) => run.results[item.id]).length,
                      total: ENGINES.length,
                    })}
                    {!run.timestamp && ` · ${M.synthetic}`}
                  </span>
                </DialogDescription>
              </DialogHeader>
              <fieldset className="hc-run-dialog-engines" aria-label={M.engine}>
                {ENGINES.map((item) => {
                  const status = outcome(run.results[item.id]);

                  return (
                    <button
                      type="button"
                      key={item.id}
                      aria-pressed={engine === item.id}
                      onClick={() => onSearch({ responseEngine: item.id })}
                    >
                      <BrandPill name={item.name} domain={item.domain} />
                      <span data-outcome={status.value}>{status.label}</span>
                    </button>
                  );
                })}
              </fieldset>
              {selected ? (
                <div className="hc-prompt-run-content">
                  <article className="hc-prompt-run-answer">
                    <div className="hc-answer-meta">
                      {engineInfo && (
                        <BrandPill name={engineInfo.name} domain={engineInfo.domain} />
                      )}
                      <span>
                        {selected.result.runId.startsWith("recorded:")
                          ? M.recordedSample
                          : M.synthetic}
                      </span>
                    </div>
                    <AnswerBody
                      result={selected.result}
                      brandName={brandName}
                      stopped={selected.stopped}
                    />
                  </article>
                  <aside className="hc-prompt-run-sources">
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
                </div>
              ) : (
                <div className="hc-run-missing">
                  <p>{run ? C.noRunResponse : C.runMissing}</p>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onSearch({ runId: runs[0]?.id })}
                  >
                    {C.latestRun}
                  </Button>
                </div>
              )}
            </>
          ) : (
            <div className="hc-run-missing">
              <p>{C.runMissing}</p>
              <Button variant="ghost" size="sm" onClick={() => onSearch({ runId: runs[0]?.id })}>
                {C.latestRun}
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
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
