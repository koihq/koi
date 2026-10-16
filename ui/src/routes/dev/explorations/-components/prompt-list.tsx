import { Link } from "@tanstack/react-router";
import { useEffect, useId, useRef, useState } from "react";
import type { Prompt } from "@/api/prompts.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { DeleteItemDialog } from "@/components/blocks/delete-item-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { ENGINES } from "../-lib/data";
import type { ExplorationSearch } from "../-lib/exploration-fixtures";
import type { HomeBrand } from "../-lib/home-evidence";
import {
  dailySeries,
  type EvidenceRecord,
  type PromptRow,
  percent,
  points,
} from "../-lib/prompt-visibility";
import { trendAreaPath, trendPath } from "../-lib/trend-path";
import C from "../-locales/mvp.en.json";
import V from "../-locales/prompt-visibility.en.json";
import { Icon } from "./common";
import { HomeBrandText } from "./home-brand-text";
import { copy } from "./list-controls";

type BreakdownItem = {
  id: string;
  name: string;
  domain?: string;
  src?: string;
  rate: number | null;
};

const rateSegments = Array.from({ length: 32 }, (_, index) => `segment-${index}`);

function RateBreakdown({
  label,
  items,
  emptyLabel = V.noData,
}: {
  label: string;
  items: BreakdownItem[];
  emptyLabel?: string;
}) {
  const [open, setOpen] = useState(false);
  const touch = useRef(false);
  const openOnPointerDown = useRef(false);

  if (!items.length) return <span className="hc-caption">{emptyLabel}</span>;

  const summary = items.reduce<number | null>(
    (highest, item) =>
      item.rate !== null && (highest === null || item.rate > highest) ? item.rate : highest,
    null,
  );

  return (
    <Tooltip open={open} onOpenChange={setOpen}>
      <TooltipTrigger
        render={<button type="button" className="hc-prompt-breakdown-trigger" />}
        aria-label={`${label}: ${items.map((item) => `${item.name} ${percent(item.rate)}`).join(", ")}`}
        aria-expanded={open}
        onPointerDown={(event) => {
          touch.current = event.pointerType === "touch";
          openOnPointerDown.current = open;
        }}
        onPointerEnter={(event) => {
          if (event.pointerType === "mouse") setOpen(true);
        }}
        onPointerLeave={(event) => {
          if (event.pointerType === "mouse") setOpen(false);
        }}
        onFocus={(event) => {
          if (event.currentTarget.matches(":focus-visible")) setOpen(true);
        }}
        onBlur={() => setOpen(false)}
        onClick={() => {
          if (touch.current) setOpen(!openOnPointerDown.current);
        }}
        onKeyDown={(event) => {
          touch.current = false;
          if (event.key === "Escape") setOpen(false);
        }}
      >
        <span className="hc-prompt-stack" aria-hidden="true">
          {items.map((item) => (
            <BrandPill
              key={item.id}
              name={item.name}
              domain={item.domain}
              src={item.src}
              iconOnly
            />
          ))}
        </span>
        <span className="hc-number">{percent(summary)}</span>
      </TooltipTrigger>
      <TooltipContent side="bottom" align="start" className="hc-prompt-breakdown">
        <span className="hc-prompt-breakdown-title">{label}</span>
        {items.map((item) => (
          <span key={item.id} className="hc-prompt-breakdown-row">
            <BrandPill name={item.name} domain={item.domain} src={item.src} />
            <span className="hc-number">{percent(item.rate)}</span>
          </span>
        ))}
      </TooltipContent>
    </Tooltip>
  );
}

function MentionRate({ row }: { row: PromptRow }) {
  const change = row.stats.changePoints;
  const filled = Math.round(((row.stats.rate ?? 0) / 100) * 32);

  return (
    <div className="hc-prompt-rate">
      <div className="hc-prompt-rate-main">
        <span className="hc-number">
          {row.stats.rate === null ? V.noAnswers : percent(row.stats.rate)}
        </span>
        <span className="hc-prompt-rate-bars" aria-hidden="true">
          {rateSegments.map((segment, index) => (
            <span key={segment} data-filled={index < filled || undefined} />
          ))}
        </span>
      </div>
      <span
        className="hc-prompt-change"
        data-direction={
          change === null ? undefined : change > 0 ? "up" : change < 0 ? "down" : undefined
        }
      >
        {change === null
          ? V.noComparison
          : Math.abs(change) < 0.05
            ? V.unchanged
            : copy(change > 0 ? V.up : V.down, { points: points(change) })}
      </span>
    </div>
  );
}

function PromptPreviewContent({
  row,
  records,
  brand,
  search,
}: {
  row: PromptRow;
  records: EvidenceRecord[];
  brand: HomeBrand;
  search: ExplorationSearch;
}) {
  const gradientId = useId().replaceAll(":", "");
  const days = search.days ?? 30;
  const competitors = row.topBrands.filter((item) => item.brand.id !== brand).slice(0, 3);
  const series = dailySeries(
    records,
    brand,
    { days, engines: search.engines, promptIds: [row.prompt.id] },
    { id: brand, name: brand, measured: true },
  );
  const width = 320;
  const height = 96;
  const coordinates = series.map((point, index) =>
    point.rate === null
      ? null
      : {
          x: (index / Math.max(1, series.length - 1)) * width,
          y: 8 + ((100 - point.rate) / 100) * (height - 16),
        },
  );
  const line = trendPath(coordinates);
  const area = trendAreaPath(coordinates, height);
  const observed = series.filter((point) => point.rate !== null);
  const trendDescription = copy(V.previewTrend, {
    days,
    start: percent(observed[0]?.rate ?? null),
    end: percent(observed.at(-1)?.rate ?? null),
  });

  return (
    <>
      <div className="hc-prompt-preview-summary">
        <div>
          <span className="hc-prompt-preview-label">{V.yourRate}</span>
          <strong className="hc-prompt-preview-rate hc-number">{percent(row.stats.rate)}</strong>
        </div>
        <div className="hc-prompt-preview-competitors">
          <span className="hc-prompt-preview-label">{V.competitors}</span>
          {competitors.length ? (
            <span className="hc-prompt-stack">
              {competitors.map(({ brand: competitor }) => (
                <BrandPill
                  key={competitor.id}
                  name={competitor.name}
                  domain={competitor.domain}
                  src={competitor.src}
                  iconOnly
                />
              ))}
            </span>
          ) : (
            <span className="hc-prompt-preview-empty-competitors">{V.previewNoCompetitors}</span>
          )}
        </div>
      </div>
      {line ? (
        <svg
          className="hc-prompt-preview-chart"
          viewBox={`0 0 ${width} ${height}`}
          preserveAspectRatio="none"
          role="img"
          aria-label={trendDescription}
        >
          <defs>
            <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="var(--hc-text)" stopOpacity="0.2" />
              <stop offset="100%" stopColor="var(--hc-text)" stopOpacity="0.08" />
            </linearGradient>
          </defs>
          {area && <path d={area} fill={`url(#${gradientId})`} />}
          <path d={line} className="hc-prompt-preview-line" />
        </svg>
      ) : (
        <div className="hc-prompt-preview-no-chart">{V.noAnswers}</div>
      )}
    </>
  );
}

function PromptPreviewLink({
  row,
  records,
  brand,
  search,
  onOpen,
}: {
  row: PromptRow;
  records: EvidenceRecord[];
  brand: HomeBrand;
  search: ExplorationSearch;
  onOpen: (id: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const details = {
    ...search,
    view: "visibility" as const,
    promptId: row.prompt.id,
    runId: undefined,
    responseEngine: undefined,
    compareId: undefined,
  };

  return (
    <Tooltip open={open} onOpenChange={(next) => !next && setOpen(false)}>
      <TooltipTrigger
        render={
          <Link
            from="/dev/explorations/home-context"
            to="/dev/explorations/home-context"
            search={details}
            data-prompt-id={row.prompt.id}
            className="hc-prompt-link"
            onClick={(event) => {
              setOpen(false);
              if (
                !event.metaKey &&
                !event.ctrlKey &&
                !event.shiftKey &&
                !event.altKey &&
                event.button === 0
              )
                onOpen(row.prompt.id);
            }}
          />
        }
        onPointerEnter={(event) => {
          if (event.pointerType === "mouse") setOpen(true);
        }}
        onPointerLeave={() => setOpen(false)}
        onFocus={(event) => {
          if (event.currentTarget.matches(":focus-visible")) setOpen(true);
        }}
        onBlur={() => setOpen(false)}
        onKeyDown={(event) => {
          if (event.key === "Escape") setOpen(false);
        }}
      >
        <HomeBrandText text={row.prompt.text} />
      </TooltipTrigger>
      {open && (
        <TooltipContent side="right" align="start" sideOffset={14} className="hc-prompt-preview">
          <PromptPreviewContent row={row} records={records} brand={brand} search={search} />
        </TooltipContent>
      )}
    </Tooltip>
  );
}

export function PromptList({
  rows,
  records,
  brand,
  search,
  onOpen,
  onEdit,
  onConfirm,
  onDelete,
  onConfirmMany,
  onDeleteMany,
}: {
  rows: PromptRow[];
  records: EvidenceRecord[];
  brand: HomeBrand;
  search: ExplorationSearch;
  onOpen: (id: string) => void;
  onEdit: (prompt: Prompt) => void;
  onConfirm: (id: string) => void;
  onDelete: (prompt: Prompt) => void;
  onConfirmMany: (ids: string[]) => void;
  onDeleteMany: (ids: string[]) => Promise<void>;
}) {
  const [selectedIDs, setSelected] = useState<string[]>([]);
  const [bulkDeleting, setBulkDeleting] = useState(false);
  const [bulkFailed, setBulkFailed] = useState(false);
  const selectAll = useRef<HTMLInputElement>(null);
  const selectAllMobile = useRef<HTMLInputElement>(null);
  const deleteTrigger = useRef<HTMLButtonElement>(null);
  const eligible = rows.filter((row) => !row.past);
  const eligibleIDs = new Set(eligible.map((row) => row.prompt.id));
  const selected = selectedIDs.filter((id) => eligibleIDs.has(id));
  const selectedSet = new Set(selected);
  const unconfirmed = eligible.filter(
    (row) => selectedSet.has(row.prompt.id) && !row.prompt.confirmed,
  );

  useEffect(() => {
    for (const checkbox of [selectAll.current, selectAllMobile.current]) {
      if (checkbox)
        checkbox.indeterminate = selected.length > 0 && selected.length < eligible.length;
    }
  }, [eligible.length, selected.length]);

  function toggle(id: string) {
    setSelected((current) => {
      const visible = current.filter((item) => eligibleIDs.has(item));
      return visible.includes(id) ? visible.filter((item) => item !== id) : [...visible, id];
    });
  }

  function cells(row: PromptRow) {
    const { prompt, past } = row;
    const engines = row.engines.flatMap((item) => {
      const engine = ENGINES.find((entry) => entry.id === item.id);
      return engine ? [{ ...engine, rate: item.rate }] : [];
    });
    const topBrands = row.topBrands.map(({ brand, rate }) => ({ ...brand, rate }));

    return [
      <div key="prompt">
        <PromptPreviewLink
          row={row}
          records={records}
          brand={brand}
          search={search}
          onOpen={onOpen}
        />
        {(past || !prompt.confirmed) && (
          <small className="hc-prompt-status">{past ? V.pastPrompt : C.draft}</small>
        )}
      </div>,
      <MentionRate key="rate" row={row} />,
      <RateBreakdown key="engines" label={V.byEngine} items={engines} />,
      <RateBreakdown
        key="brands"
        label={V.mostMentionedBrands}
        items={topBrands}
        emptyLabel={row.stats.rate === null ? V.noData : V.noBrandMentions}
      />,
      <div key="actions" className="hc-actions">
        {!past && !prompt.confirmed && (
          <Button variant="ghost" size="sm" onClick={() => onConfirm(prompt.id)}>
            {C.confirm}
          </Button>
        )}
        {!past && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={copy(C.actions, { name: prompt.text })}
                />
              }
            >
              <Icon name="more" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="hc-brand-menu">
              <DropdownMenuItem onClick={() => onEdit(prompt)}>{C.edit}</DropdownMenuItem>
              <DropdownMenuItem onClick={() => onDelete(prompt)}>{C.delete}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>,
    ];
  }

  const headers = [C.prompt, V.yourRate, V.byEngine, V.mostMentionedBrands, V.actions];

  return (
    <TooltipProvider delay={0}>
      <output className="sr-only" aria-live="polite">
        {selected.length ? copy(V.selectedCount, { count: selected.length }) : ""}
      </output>
      {selected.length > 0 && (
        <div className="hc-prompt-selection-actions">
          <span>{copy(V.selectedCount, { count: selected.length })}</span>
          {unconfirmed.length > 0 && (
            <Button
              size="sm"
              onClick={() => {
                onConfirmMany(unconfirmed.map((row) => row.prompt.id));
                setSelected([]);
                document.getElementById("hc-page-heading")?.focus({ preventScroll: true });
              }}
            >
              {unconfirmed.length === selected.length
                ? C.confirm
                : copy(V.confirmSelected, { count: unconfirmed.length })}
            </Button>
          )}
          <Button
            ref={deleteTrigger}
            variant="destructive"
            size="sm"
            onClick={() => {
              setBulkFailed(false);
              setBulkDeleting(true);
            }}
          >
            {C.delete}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setSelected([]);
              document.getElementById("hc-page-heading")?.focus({ preventScroll: true });
            }}
          >
            {V.clearSelection}
          </Button>
        </div>
      )}
      <div className="hc-prompt-table">
        <Table aria-label={V.tableLabel}>
          <TableHeader>
            <TableRow>
              <TableHead className="hc-prompt-select-cell">
                {eligible.length > 0 && (
                  <span className="hc-prompt-check-wrap">
                    <input
                      ref={selectAll}
                      type="checkbox"
                      className="hc-prompt-check"
                      aria-label={V.selectPage}
                      checked={selected.length === eligible.length}
                      onChange={(event) =>
                        setSelected(
                          event.target.checked ? eligible.map((row) => row.prompt.id) : [],
                        )
                      }
                    />
                    <span className="hc-prompt-check-visual" aria-hidden="true" />
                  </span>
                )}
              </TableHead>
              {headers.map((header) => (
                <TableHead key={header}>{header}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow
                key={row.prompt.id}
                data-selected={selectedSet.has(row.prompt.id) || undefined}
              >
                <TableCell className="hc-prompt-select-cell">
                  {!row.past && (
                    <span className="hc-prompt-check-wrap">
                      <input
                        type="checkbox"
                        className="hc-prompt-check"
                        aria-label={copy(V.selectPrompt, { prompt: row.prompt.text })}
                        checked={selectedSet.has(row.prompt.id)}
                        onChange={() => toggle(row.prompt.id)}
                      />
                      <span className="hc-prompt-check-visual" aria-hidden="true" />
                    </span>
                  )}
                </TableCell>
                {cells(row).map((cell, index) => (
                  <TableCell key={headers[index]}>{cell}</TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {eligible.length > 0 && (
        <label className="hc-prompt-mobile-select-all">
          <span className="hc-prompt-check-wrap">
            <input
              ref={selectAllMobile}
              type="checkbox"
              className="hc-prompt-check"
              checked={selected.length === eligible.length}
              onChange={(event) =>
                setSelected(event.target.checked ? eligible.map((row) => row.prompt.id) : [])
              }
            />
            <span className="hc-prompt-check-visual" aria-hidden="true" />
          </span>
          {V.selectPage}
        </label>
      )}
      <ul className="hc-prompt-mobile" aria-label={V.tableLabel}>
        {rows.map((row) => {
          const content = cells(row);

          return (
            <li key={row.prompt.id} data-selected={selectedSet.has(row.prompt.id) || undefined}>
              <div className="hc-prompt-mobile-heading">
                {!row.past && (
                  <span className="hc-prompt-check-wrap">
                    <input
                      type="checkbox"
                      className="hc-prompt-check"
                      aria-label={copy(V.selectPrompt, { prompt: row.prompt.text })}
                      checked={selectedSet.has(row.prompt.id)}
                      onChange={() => toggle(row.prompt.id)}
                    />
                    <span className="hc-prompt-check-visual" aria-hidden="true" />
                  </span>
                )}
                {content[0]}
              </div>
              <dl>
                {headers.slice(1, 4).map((header, index) => (
                  <div key={header}>
                    <dt>{header}</dt>
                    <dd>{content[index + 1]}</dd>
                  </div>
                ))}
              </dl>
              {content[4]}
            </li>
          );
        })}
      </ul>
      <DeleteItemDialog
        itemID={bulkDeleting ? "selected" : undefined}
        title={copy(V.deleteSelectedTitle, { count: selected.length })}
        description={V.deleteSelectedDescription}
        actionLabel={copy(V.deleteSelected, { count: selected.length })}
        onClose={() => setBulkDeleting(false)}
        pending={false}
        failed={bulkFailed}
        onDelete={async () => {
          try {
            await onDeleteMany(selected);
            setSelected([]);
          } catch (error) {
            setBulkFailed(true);
            throw error;
          }
        }}
        finalFocus={() =>
          deleteTrigger.current?.isConnected
            ? deleteTrigger.current
            : document.getElementById("hc-page-heading")
        }
      />
    </TooltipProvider>
  );
}
