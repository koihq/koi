import {
  Delete02Icon,
  PauseIcon,
  PencilEdit02Icon,
  PlayIcon,
  Tick02Icon,
} from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";
import { parseAsBoolean, parseAsNumberLiteral, parseAsStringLiteral, useQueryStates } from "nuqs";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Prompt } from "@/api/prompts.api";
import { BrandLabel } from "@/components/blocks/brand-label";
import { BrandPill } from "@/components/blocks/brand-pill";
import { DataTable, type DataTableColumn } from "@/components/blocks/data-table";
import { DeleteItemDialog } from "@/components/blocks/delete-item-dialog";
import { EngineIcon } from "@/components/blocks/engine-icon";
import { GenerationControls } from "@/components/blocks/generation-controls";
import { PageNav } from "@/components/blocks/page-nav";
import { PromptDialog } from "@/components/blocks/prompt-dialog";
import { QueryRefreshIndicator } from "@/components/blocks/query-refresh-indicator";
import { RowAction, RowActions } from "@/components/blocks/row-actions";
import { SearchToggle } from "@/components/blocks/search-toggle";
import { type SortDirection, SortHeader } from "@/components/blocks/sort-header";
import { StatusTabs } from "@/components/blocks/status-tabs";
import { TrackingReview } from "@/components/blocks/tracking-review";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { TooltipProvider } from "@/components/ui/tooltip";
import { usePromptMutations } from "@/hooks/use-prompt-mutations";
import { tableSearchParams } from "@/hooks/use-table-state";
import { formatPoints, formatRate } from "@/lib/format";
import {
  change,
  engines,
  rate,
  type VisibilityReport,
  type VisibilityScope,
} from "@/lib/visibility";
import { PromptPreviewLink } from "@/routes/_authenticated/_app/-components/prompt-preview-link";
import { PromptRateBreakdown } from "@/routes/_authenticated/_app/-components/prompt-rate-breakdown";
import { PromptSelectCheckbox } from "@/routes/_authenticated/_app/-components/prompt-select-checkbox";
import { EngineMeterCell } from "./engine-meter";
import { usePromptBulkActions } from "./use-prompt-bulk-actions";
import { usePromptCounts } from "./use-prompt-counts";
import { usePromptList } from "./use-prompt-list";

type PromptRow = {
  id: string;
  text: string;
  prompt?: Prompt;
  visibility?: VisibilityReport;
  past: boolean;
};

export const promptSearchParams = {
  ...tableSearchParams,
  pageSize: parseAsNumberLiteral([10, 20, 50] as const)
    .withDefault(20)
    .withOptions({ clearOnDefault: true }),
  status: parseAsStringLiteral(["all", "confirmed", "unconfirmed", "past"] as const)
    .withDefault("all")
    .withOptions({ clearOnDefault: true }),
  sort: parseAsStringLiteral(["original", "prompt", "rate"] as const)
    .withDefault("original")
    .withOptions({ clearOnDefault: true }),
  descending: parseAsBoolean.withDefault(false).withOptions({ clearOnDefault: true }),
};

const promptHeadClass =
  "h-auto px-3 py-1.5 text-start text-xs text-muted-foreground whitespace-normal";
const promptCellClass =
  "px-3 py-4 align-middle text-start text-body whitespace-normal [overflow-wrap:anywhere]";
const promptRowLinkClass =
  "block max-w-2xl text-sm/normal font-semibold text-foreground underline-offset-[3px] [overflow-wrap:anywhere] hover:underline";

/** Signed change as a tinted chip: green up, red down. */
export function DeltaChip({ value, locale }: { value: number | null; locale: string }) {
  if (value === null) return null;

  return (
    <span
      className={`rounded-chip px-1.5 py-0.5 text-xs tabular-nums ${value > 0 ? "bg-positive/12 text-positive" : value < 0 ? "bg-negative/12 text-negative" : "bg-muted text-muted-foreground"}`}
    >
      {formatPoints(value, locale)}
    </span>
  );
}

export function PromptManager({
  brandID,
  scope,
  collecting,
  draftStartFailed = false,
}: {
  brandID: string;
  scope: VisibilityScope;
  collecting: boolean;
  draftStartFailed?: boolean;
}) {
  const { t, i18n } = useTranslation(["management", "home"]);
  const mutations = usePromptMutations(brandID);
  const [search, setSearch] = useQueryStates(promptSearchParams);
  const [selected, setSelected] = useState<string[]>([]);
  const [editing, setEditing] = useState<Prompt>();
  const [deleting, setDeleting] = useState<Prompt>();
  const [bulkDeleting, setBulkDeleting] = useState(false);
  const heading = useRef<HTMLHeadingElement>(null);
  const trigger = useRef<HTMLElement | null>(null);
  const { active, live, history, page, totalPages } = usePromptList(
    brandID,
    scope,
    search,
    collecting,
  );
  const counts = usePromptCounts(brandID, scope);
  const visible: PromptRow[] =
    search.status === "past"
      ? (history.data?.items ?? []).map((prompt) => ({
          id: prompt.promptId,
          text: prompt.promptText,
          visibility: prompt.visibility,
          past: true,
        }))
      : (live.data?.items ?? []).map((prompt) => ({
          id: prompt.id,
          text: prompt.text,
          prompt,
          visibility: prompt.visibility,
          past: false,
        }));
  const previousPage = useRef(page);
  const visibleLiveIDs = visible.filter((row) => !row.past).map((row) => row.id);
  const selectedVisible = selected.filter((id) => visibleLiveIDs.includes(id));
  const selectedPrompts = visible.flatMap((row) =>
    row.prompt && selectedVisible.includes(row.id) ? [row.prompt] : [],
  );
  const bulk = usePromptBulkActions(brandID, selectedPrompts, setSelected);
  const bulkPending = bulk.pending;
  const bulkFailed = bulk.failed;
  const pending = Object.values(mutations).some((mutation) => mutation.isPending) || bulkPending;

  useEffect(() => {
    if (
      active.isSuccess &&
      !active.isPlaceholderData &&
      !active.isFetching &&
      page > Math.max(1, totalPages)
    ) {
      void setSearch({ page: Math.max(1, totalPages) }, { history: "replace" });
    }
  }, [active.isSuccess, active.isPlaceholderData, active.isFetching, page, setSearch, totalPages]);

  useEffect(() => {
    if (previousPage.current === page) return;

    previousPage.current = page;
    setSelected([]);
  }, [page]);

  async function bulkAction(kind: "confirm" | "delete") {
    const succeeded = await bulk.run(kind);
    if (kind === "delete" && succeeded) setBulkDeleting(false);
  }

  function changeSearch(patch: Partial<typeof search>) {
    void setSearch({ ...patch, page: 1 }, { history: "replace" });
    setSelected([]);
  }

  function sortDirection(column: "prompt" | "rate"): SortDirection {
    if (search.sort !== column) return "none";

    return search.descending ? "descending" : "ascending";
  }

  /** Each click moves a column through its natural order, its reverse, then back to none. */
  function cycleSort(column: "prompt" | "rate") {
    const natural = column === "rate" ? "descending" : "ascending";
    const current = sortDirection(column);

    if (current === "none") {
      changeSearch({ sort: column, descending: natural === "descending" });
    } else if (current === natural) {
      changeSearch({ sort: column, descending: natural !== "descending" });
    } else {
      changeSearch({ sort: "original", descending: false });
    }
  }

  function select(id: string) {
    setSelected((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id],
    );
  }

  const renderPrompt = (row: PromptRow) => (
    <div>
      {row.prompt ? (
        <PromptPreviewLink prompt={row.prompt} />
      ) : (
        <Link to="/prompts/$promptId" params={{ promptId: row.id }} className={promptRowLinkClass}>
          {row.text}
        </Link>
      )}
      {(row.past || !row.prompt?.confirmed || row.prompt.inactive) && (
        <small className="mt-1.5 block text-xs text-muted-foreground">
          {row.past
            ? t("home:dashboard.pastPrompt")
            : row.prompt?.inactive
              ? t("inactivePrompt")
              : t("home:dashboard.draft")}
        </small>
      )}
    </div>
  );

  const renderRate = (row: PromptRow) => {
    const currentRate = rate(row.visibility);

    if (currentRate === null) {
      return (
        <span className="text-xs text-muted-foreground">{t("home:dashboard.noAnswersYet")}</span>
      );
    }

    return (
      <PromptRateBreakdown
        label={t("home:dashboard.byEngine")}
        items={engines.map((engine) => ({
          id: engine.id,
          name: engine.name,
          rate: rate(row.visibility, engine.id),
          icon: <EngineIcon engine={engine.id} />,
        }))}
        openOnHover={false}
      >
        <EngineMeterCell
          rate={currentRate}
          rateFor={(engine) => rate(row.visibility, engine)}
          locale={i18n.language}
          change={<DeltaChip value={change(row.visibility)} locale={i18n.language} />}
        />
      </PromptRateBreakdown>
    );
  };

  const renderRival = (row: PromptRow) => {
    if (!row.visibility?.metric.completed) {
      return <span className="text-xs text-muted-foreground">{t("home:dashboard.noData")}</span>;
    }

    const top = row.visibility.competitors
      .map((competitor) => ({
        competitor,
        rate: competitor.metric.rate,
      }))
      .filter(({ rate: competitorRate }) => competitorRate !== null && competitorRate > 0)
      .sort((left, right) => (right.rate ?? -1) - (left.rate ?? -1))
      .slice(0, 3);
    const first = top[0];

    if (!first) {
      return <span className="text-xs text-muted-foreground">{t("home:dashboard.noRival")}</span>;
    }

    return (
      <PromptRateBreakdown
        label={t("home:dashboard.rivalsByRate")}
        items={top.map(({ competitor, rate: competitorRate }) => ({
          id: competitor.id,
          name: competitor.name,
          rate: competitorRate,
          icon: <BrandPill name={competitor.name} domain={competitor.domain} iconOnly />,
        }))}
      >
        <BrandLabel
          size="sm"
          name={first.competitor.name}
          domain={first.competitor.domain}
          detail={first.rate === null ? "—" : formatRate(first.rate, i18n.language)}
        />
      </PromptRateBreakdown>
    );
  };

  const renderActions = (row: PromptRow) => {
    const prompt = row.prompt;
    if (!prompt) return null;

    return (
      <RowActions>
        {!prompt.confirmed && (
          <RowAction
            persistent
            label={t("confirm")}
            icon={Tick02Icon}
            disabled={pending}
            onClick={() => mutations.confirm.mutate(row.id)}
          />
        )}
        <RowAction
          label={t("edit")}
          icon={PencilEdit02Icon}
          disabled={pending}
          onClick={() => setEditing(prompt)}
        />
        {prompt.confirmed && (
          <RowAction
            label={prompt.inactive ? t("resumePrompt") : t("pausePrompt")}
            icon={prompt.inactive ? PlayIcon : PauseIcon}
            pressed={prompt.inactive}
            disabled={pending}
            onClick={() =>
              mutations.update.mutate({ id: row.id, request: { inactive: !prompt.inactive } })
            }
          />
        )}
        <RowAction
          destructive
          label={t("delete")}
          icon={Delete02Icon}
          disabled={pending}
          onClick={() => {
            trigger.current = document.activeElement as HTMLElement;
            setDeleting(prompt);
          }}
        />
      </RowActions>
    );
  };

  const columns: DataTableColumn<PromptRow>[] = [
    {
      id: "select",
      header: visibleLiveIDs.length > 0 && (
        <PromptSelectCheckbox
          label={t("home:dashboard.selectPage")}
          checked={selectedVisible.length === visibleLiveIDs.length}
          indeterminate={
            selectedVisible.length > 0 && selectedVisible.length < visibleLiveIDs.length
          }
          onChange={(checked) =>
            setSelected((current) =>
              checked
                ? [...new Set([...current, ...visibleLiveIDs])]
                : current.filter((id) => !visibleLiveIDs.includes(id)),
            )
          }
        />
      ),
      headClassName: `${promptHeadClass} w-12 text-center`,
      cellClassName: `${promptCellClass} text-center`,
      cell: (row) =>
        !row.past && (
          <PromptSelectCheckbox
            label={t("home:dashboard.selectPrompt", { text: row.text })}
            checked={selectedVisible.includes(row.id)}
            onChange={() => select(row.id)}
          />
        ),
    },
    {
      id: "prompt",
      header: (
        <SortHeader
          label={t("prompt")}
          direction={sortDirection("prompt")}
          onSort={() => cycleSort("prompt")}
        />
      ),
      sort: sortDirection("prompt"),
      headClassName: `${promptHeadClass} w-[44%]`,
      cellClassName: promptCellClass,
      cell: renderPrompt,
    },
    {
      id: "rate",
      header: (
        <SortHeader
          label={t("home:dashboard.visibility")}
          direction={sortDirection("rate")}
          onSort={() => cycleSort("rate")}
        />
      ),
      sort: sortDirection("rate"),
      headClassName: `${promptHeadClass} w-[26%]`,
      cellClassName: promptCellClass,
      cell: renderRate,
    },
    {
      id: "rival",
      header: t("home:dashboard.topRival"),
      headClassName: `${promptHeadClass} w-[20%]`,
      cellClassName: promptCellClass,
      cell: renderRival,
    },
    {
      id: "actions",
      header: <span className="sr-only">{t("actions")}</span>,
      headClassName: `${promptHeadClass} w-[10%]`,
      cellClassName: `${promptCellClass} text-end`,
      cell: renderActions,
    },
  ];

  const totalItems = active.data?.totalItems ?? 0;
  const mutationFailed = mutations.confirm.isError || mutations.update.isError || bulkFailed;
  const statusOptions = [
    { value: "all", label: t("all"), count: counts.all },
    { value: "confirmed", label: t("confirmed"), count: counts.confirmed },
    { value: "unconfirmed", label: t("unconfirmed"), count: counts.unconfirmed },
    { value: "past", label: t("home:history.title"), count: counts.past },
  ] as const;

  return (
    <section
      id="prompts"
      aria-labelledby="prompts-heading"
      className="group/refresh relative min-w-0 rounded-panel bg-surface p-4 sm:p-7 scroll-mt-8"
      aria-busy={active.isFetching}
      data-refreshing={active.isFetching && Boolean(active.data)}
    >
      <QueryRefreshIndicator active={active.isFetching} label={t("loading")} />
      <div className="mb-5 flex flex-wrap items-center justify-between gap-x-6 gap-y-3">
        <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
          <h2
            id="prompts-heading"
            ref={heading}
            tabIndex={-1}
            className="text-section font-semibold"
          >
            {t("prompts")}
          </h2>
          <StatusTabs
            label={t("statusLabel")}
            value={search.status}
            options={[...statusOptions]}
            onChange={(status) => changeSearch({ status })}
          />
        </div>
        <div className="flex flex-wrap items-center gap-2 [&>div]:m-0 [&_output:empty]:hidden">
          <SearchToggle
            label={t("searchPrompts")}
            clearLabel={t("home:dashboard.clearSearch")}
            value={search.q}
            onChange={(q) => changeSearch({ q })}
          />
          <GenerationControls
            brandID={brandID}
            mode="prompts"
            setupStartFailed={draftStartFailed}
            hideSuccessMessage
          />
          <PromptDialog
            label={t("addPrompt")}
            pending={mutations.create.isPending}
            onSave={(request) => mutations.create.mutateAsync(request).then(() => undefined)}
            variant="outline"
          />
        </div>
      </div>
      <div className="[&:not(:empty)]:mb-4">
        <TrackingReview
          brandID={brandID}
          kind="prompts"
          disabled={pending}
          showTrackingError={false}
        />
      </div>

      {mutationFailed && !bulkDeleting && (
        <p role="alert" className="mb-4 text-sm text-destructive">
          {bulkFailed ? t("home:dashboard.bulkError") : t("mutationError")}
        </p>
      )}

      <output className="sr-only" aria-live="polite">
        {selectedVisible.length
          ? t("home:dashboard.selectedCount", { count: selectedVisible.length })
          : ""}
      </output>
      {selectedVisible.length > 0 && (
        <div className="surface-float fixed bottom-[max(1.5rem,env(safe-area-inset-bottom))] left-1/2 z-30 flex w-max max-w-[calc(100vw-2rem)] -translate-x-1/2 flex-wrap items-center justify-center gap-1 px-2.5 py-2 text-body text-muted-foreground">
          <span className="px-2 text-foreground">
            {t("home:dashboard.selectedCount", { count: selectedVisible.length })}
          </span>
          {selectedPrompts.some((prompt) => !prompt.confirmed) && (
            <Button
              type="button"
              size="sm"
              disabled={pending}
              onClick={() => void bulkAction("confirm")}
            >
              {t("confirm")}
            </Button>
          )}
          <Button
            type="button"
            size="sm"
            variant="destructive"
            disabled={pending}
            onClick={() => setBulkDeleting(true)}
          >
            {t("delete")}
          </Button>
          <Button type="button" size="sm" variant="ghost" onClick={() => setSelected([])}>
            {t("home:dashboard.clearSelection")}
          </Button>
        </div>
      )}

      <TooltipProvider delay={0}>
        {active.isError ? (
          <p role="alert" className="py-12 text-sm text-destructive">
            {t("loadError")}{" "}
            <Button variant="link" onClick={() => void active.refetch()}>
              {t("retry")}
            </Button>
          </p>
        ) : active.isPending ? (
          <p className="py-12 text-sm text-muted-foreground">{t("loading")}</p>
        ) : visible.length === 0 ? (
          <p className="py-12 text-sm text-muted-foreground">
            {search.q || search.status !== "all" ? t("noStatusResults") : t("promptEmpty")}
          </p>
        ) : (
          <div className="@container group-data-[refreshing=true]/refresh:opacity-50">
            <div className="hidden @4xl:block">
              <DataTable
                interactive
                ariaLabel={t("tableLabel")}
                className="table-fixed"
                data={visible}
                columns={columns}
                getRowId={(row) => row.id}
                isRowSelected={(row) => selectedVisible.includes(row.id)}
              />
            </div>
            <ul className="list-rows grid gap-0.5 @4xl:hidden">
              {visible.map((row) => (
                <li
                  key={row.id}
                  data-selected={selectedVisible.includes(row.id) || undefined}
                  className="group flex min-w-0 items-start gap-2 rounded-control px-2 py-3 hover:bg-signal-pill data-[selected]:bg-signal-pill"
                >
                  {!row.past && (
                    <PromptSelectCheckbox
                      label={t("home:dashboard.selectPrompt", { text: row.text })}
                      checked={selectedVisible.includes(row.id)}
                      onChange={() => select(row.id)}
                    />
                  )}
                  <div className="grid min-w-0 flex-1 gap-2">
                    {renderPrompt(row)}
                    <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
                      {renderRate(row)}
                      {renderRival(row)}
                    </div>
                  </div>
                  {renderActions(row)}
                </li>
              ))}
            </ul>
          </div>
        )}
      </TooltipProvider>
      <PageNav
        page={page}
        pageSize={search.pageSize}
        total={totalItems}
        hasPrev={page > 1}
        hasNext={page < totalPages}
        onPage={(next) => void setSearch({ page: next }, { history: "push" })}
      />

      <PromptDialog
        prompt={editing}
        label={t("edit")}
        hideTrigger
        open={Boolean(editing)}
        onOpenChange={(open) => {
          if (!open) setEditing(undefined);
        }}
        pending={mutations.update.isPending}
        onSave={(request) =>
          editing
            ? mutations.update.mutateAsync({ id: editing.id, request }).then(() => undefined)
            : Promise.resolve()
        }
      />
      <DeleteItemDialog
        itemID={deleting?.id}
        title={t("deletePromptTitle")}
        description={t("promptDeleteDescription")}
        actionLabel={t("deletePrompt")}
        onClose={() => setDeleting(undefined)}
        pending={mutations.remove.isPending}
        failed={mutations.remove.isError}
        onDelete={mutations.remove.mutateAsync}
        finalFocus={() => (trigger.current?.isConnected ? trigger.current : heading.current)}
      />
      <AlertDialog
        open={bulkDeleting}
        onOpenChange={(open) => {
          if (!bulkPending) setBulkDeleting(open);
        }}
      >
        <AlertDialogContent finalFocus={() => heading.current}>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("home:dashboard.deleteSelectedTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("home:dashboard.deleteSelectedDescription", { count: selectedVisible.length })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {bulkFailed && (
            <p role="alert" className="text-sm text-destructive">
              {t("home:dashboard.bulkError")}
            </p>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={bulkPending}>{t("cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={bulkPending}
              onClick={() => void bulkAction("delete")}
            >
              {t("delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
