import {
  ArrowLeft01Icon,
  ArrowRight01Icon,
  MoreHorizontalIcon,
  SortingIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
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
import { PromptDialog } from "@/components/blocks/prompt-dialog";
import { QueryRefreshIndicator } from "@/components/blocks/query-refresh-indicator";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
  sort: parseAsStringLiteral(["original", "prompt", "rate", "change"] as const)
    .withDefault("original")
    .withOptions({ clearOnDefault: true }),
  descending: parseAsBoolean.withDefault(false).withOptions({ clearOnDefault: true }),
};

const promptHeadClass =
  "h-auto px-3 py-1.5 text-start text-xs font-normal text-muted-foreground whitespace-normal";
const promptCellClass =
  "px-3 py-4 align-middle text-start text-body whitespace-normal [overflow-wrap:anywhere]";
const promptRowLinkClass =
  "block max-w-2xl text-sm/normal font-medium text-foreground underline-offset-[3px] [overflow-wrap:anywhere] hover:underline";

/** Signed change as a tinted chip: green up, red down. */
export function DeltaChip({ value, locale }: { value: number | null; locale: string }) {
  if (value === null) return null;

  return (
    <span
      className={`rounded-sm px-1.5 py-0.5 text-xs tabular-nums ${value > 0 ? "bg-positive/12 text-positive" : value < 0 ? "bg-negative/12 text-negative" : "bg-muted text-muted-foreground"}`}
    >
      {formatPoints(value, locale)}
    </span>
  );
}

function PromptChoice({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
}) {
  return (
    <Select
      value={value}
      items={options}
      onValueChange={(next) => {
        if (next !== null) onChange(next);
      }}
    >
      <SelectTrigger aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
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

  const renderActions = (row: PromptRow) =>
    row.prompt ? (
      <div className="flex flex-wrap items-center justify-end gap-2">
        {!row.prompt.confirmed && (
          <Button
            variant="ghost"
            size="sm"
            disabled={pending}
            onClick={() => mutations.confirm.mutate(row.id)}
          >
            {t("confirm")}
          </Button>
        )}
        <DropdownMenu>
          <DropdownMenuTrigger
            aria-label={t("home:dashboard.promptActions", { text: row.text })}
            className="grid size-6 place-items-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring"
          >
            <HugeiconsIcon icon={MoreHorizontalIcon} size={16} aria-hidden="true" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem disabled={pending} onClick={() => setEditing(row.prompt)}>
              {t("edit")}
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={pending || !row.prompt.confirmed}
              onClick={() =>
                mutations.update.mutate({
                  id: row.id,
                  request: { inactive: !row.prompt?.inactive },
                })
              }
            >
              {row.prompt.inactive ? t("resumePrompt") : t("pausePrompt")}
            </DropdownMenuItem>
            <DropdownMenuItem
              variant="destructive"
              disabled={pending}
              onClick={() => {
                trigger.current = document.activeElement as HTMLElement;
                setDeleting(row.prompt);
              }}
            >
              {t("delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    ) : null;

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
      header: t("prompt"),
      headClassName: `${promptHeadClass} w-[44%]`,
      cellClassName: promptCellClass,
      cell: renderPrompt,
    },
    {
      id: "rate",
      header: t("home:dashboard.visibility"),
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
  const firstItem = totalItems === 0 ? 0 : (page - 1) * search.pageSize + 1;
  const lastItem = Math.min(totalItems, page * search.pageSize);
  const mutationFailed = mutations.confirm.isError || mutations.update.isError || bulkFailed;
  const sortOptions = [
    { value: "original", label: t("home:dashboard.originalOrder") },
    { value: "prompt", label: t("prompt") },
    { value: "rate", label: t("home:dashboard.mentionRate") },
    { value: "change", label: t("home:dashboard.sortChange") },
  ] as const;
  const statusOptions = [
    { value: "all", label: t("all") },
    { value: "confirmed", label: t("confirmed") },
    { value: "unconfirmed", label: t("unconfirmed") },
    { value: "past", label: t("home:history.title") },
  ] as const;

  return (
    <section
      id="prompts"
      aria-labelledby="prompts-heading"
      className="group/refresh relative min-w-0 rounded-lg bg-surface p-4 sm:p-7 scroll-mt-8"
      aria-busy={active.isFetching}
      data-refreshing={active.isFetching && Boolean(active.data)}
    >
      <QueryRefreshIndicator active={active.isFetching} label={t("loading")} />
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <h2 id="prompts-heading" ref={heading} tabIndex={-1} className="text-section font-semibold">
          {t("prompts")}
        </h2>
        <div className="flex flex-wrap items-center gap-2 [&>div]:m-0 [&_output:empty]:hidden">
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

      <div className="mb-4 flex flex-wrap items-center gap-x-4 gap-y-3">
        <Input
          type="search"
          value={search.q}
          onChange={(event) => changeSearch({ q: event.target.value })}
          placeholder={t("searchPrompts")}
          aria-label={t("searchPrompts")}
          className="w-full sm:max-w-64"
        />
        <fieldset className="flex flex-wrap gap-0.5" aria-label={t("statusLabel")}>
          {statusOptions.map((option) => (
            <Button
              key={option.value}
              type="button"
              variant="ghost"
              size="pill"
              aria-pressed={search.status === option.value}
              className="text-muted-foreground"
              onClick={() => changeSearch({ status: option.value })}
            >
              {option.label}
            </Button>
          ))}
        </fieldset>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="pill"
                className="gap-1.5 text-muted-foreground sm:ms-auto"
              />
            }
          >
            <HugeiconsIcon icon={SortingIcon} data-icon="inline-start" aria-hidden="true" />
            {search.sort === "original"
              ? t("home:dashboard.sortBy")
              : sortOptions.find((option) => option.value === search.sort)?.label}
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuGroup>
              <DropdownMenuLabel>{t("home:dashboard.sortPrompts")}</DropdownMenuLabel>
              <DropdownMenuRadioGroup
                value={search.sort}
                onValueChange={(value) => changeSearch({ sort: value as typeof search.sort })}
              >
                {sortOptions.map((option) => (
                  <DropdownMenuRadioItem key={option.value} value={option.value}>
                    {option.label}
                  </DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuGroup>
            {search.sort !== "original" && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuRadioGroup
                  value={search.descending ? "descending" : "ascending"}
                  onValueChange={(value) => changeSearch({ descending: value === "descending" })}
                >
                  <DropdownMenuRadioItem value="ascending">
                    {t("home:dashboard.ascending")}
                  </DropdownMenuRadioItem>
                  <DropdownMenuRadioItem value="descending">
                    {t("home:dashboard.descending")}
                  </DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
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
                  className="group flex min-w-0 items-start gap-2 rounded-md px-2 py-3 hover:bg-signal-pill data-[selected]:bg-signal-pill"
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
      <div className="mt-6 flex flex-wrap items-center gap-3 text-xs text-muted-foreground empty:hidden">
        {totalItems > 0 && (
          <>
            <span className="whitespace-nowrap tabular-nums">
              {t("home:dashboard.pageRange", {
                start: firstItem,
                end: lastItem,
                total: totalItems,
              })}
            </span>
            <PromptChoice
              label={t("pageSizeLabel")}
              value={String(search.pageSize)}
              options={[10, 20, 50].map((value) => ({
                value: String(value),
                label: String(value),
              }))}
              onChange={(value) => changeSearch({ pageSize: Number(value) as 10 | 20 | 50 })}
            />
            <div className="ms-auto flex gap-1">
              <Button
                variant="ghost"
                size="circle-sm"
                aria-label={t("home:dashboard.previous")}
                disabled={page <= 1}
                onClick={() => void setSearch({ page: page - 1 }, { history: "push" })}
              >
                <HugeiconsIcon icon={ArrowLeft01Icon} aria-hidden="true" />
              </Button>
              <Button
                variant="ghost"
                size="circle-sm"
                aria-label={t("home:dashboard.next")}
                disabled={page >= totalPages}
                onClick={() => void setSearch({ page: page + 1 }, { history: "push" })}
              >
                <HugeiconsIcon icon={ArrowRight01Icon} aria-hidden="true" />
              </Button>
            </div>
          </>
        )}
      </div>

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
