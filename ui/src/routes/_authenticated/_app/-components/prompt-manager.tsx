import {
  ChatGptIcon,
  GoogleGeminiIcon,
  GoogleIcon,
  MoreHorizontalIcon,
  PerplexityAiIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Link } from "@tanstack/react-router";
import { parseAsBoolean, parseAsNumberLiteral, parseAsStringLiteral, useQueryStates } from "nuqs";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Prompt } from "@/api/prompts.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { DeleteItemDialog } from "@/components/blocks/delete-item-dialog";
import { GenerationControls } from "@/components/blocks/generation-controls";
import { PromptDialog } from "@/components/blocks/prompt-dialog";
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
  DropdownMenuItem,
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { TooltipProvider } from "@/components/ui/tooltip";
import { usePromptMutations } from "@/hooks/use-prompt-mutations";
import { tableSearchParams } from "@/hooks/use-table-state";
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

const engineIcons = {
  chatgpt: ChatGptIcon,
  perplexity: PerplexityAiIcon,
  gemini: GoogleGeminiIcon,
  google_ai_overviews: GoogleIcon,
};

const rateSegments = Array.from({ length: 32 }, (_, index) => `segment-${index}`);
const promptHeadClass =
  "h-auto px-2 py-1.5 text-start text-xs font-normal text-muted-foreground whitespace-normal [overflow-wrap:anywhere]";
const promptCellClass =
  "px-2 py-3.5 align-middle text-start text-[13px]/[1.5] whitespace-normal [overflow-wrap:anywhere] group-data-[selected]:bg-signal-pill motion-safe:transition-[background-color] motion-safe:duration-[120ms]";
const promptRowLinkClass =
  "block max-w-2xl text-sm/normal font-medium text-foreground underline-offset-[3px] [overflow-wrap:anywhere] hover:underline";

function rateBars(rate: number | null) {
  return (
    <span className="inline-flex max-w-[140px] items-center gap-0.5" aria-hidden="true">
      {rateSegments.map((segment, index) => (
        <span
          key={segment}
          className="h-4 w-0.5 flex-[0_0_2px] rounded-[1px] bg-signal-pill data-[filled=true]:bg-positive"
          data-filled={rate !== null && index < Math.round((rate / 100) * 32)}
        />
      ))}
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
  const { t } = useTranslation(["management", "home"]);
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
    if (active.isSuccess && page > Math.max(1, totalPages)) {
      void setSearch({ page: Math.max(1, totalPages) }, { history: "replace" });
    }
  }, [active.isSuccess, page, setSearch, totalPages]);

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
    const currentChange = change(row.visibility);
    return (
      <div className="grid gap-[3px] text-[13px]">
        <div className="flex flex-wrap items-center gap-2">
          <span className="tabular-nums">
            {currentRate === null ? t("home:dashboard.noAnswersYet") : `${currentRate.toFixed(1)}%`}
          </span>
          {rateBars(currentRate)}
        </div>
        <small
          className={`text-xs font-normal ${currentChange === null ? "" : currentChange >= 0 ? "text-positive" : "text-negative"}`}
        >
          {currentChange === null
            ? t("home:dashboard.noPreviousData")
            : `${currentChange > 0 ? "+" : ""}${currentChange.toFixed(1)} points`}
        </small>
      </div>
    );
  };

  const renderEngines = (row: PromptRow) => (
    <PromptRateBreakdown
      label={t("home:dashboard.byEngine")}
      items={engines.map((engine) => ({
        id: engine.id,
        name: engine.name,
        rate: rate(row.visibility, engine.id),
        icon: <HugeiconsIcon icon={engineIcons[engine.id]} size={16} aria-hidden="true" />,
      }))}
    />
  );

  const renderBrands = (row: PromptRow) => {
    if (!row.visibility?.metric.completed)
      return (
        <span className="text-xs font-normal text-muted-foreground">
          {t("home:dashboard.noData")}
        </span>
      );
    const top = row.visibility.competitors
      .map((competitor) => ({
        competitor,
        rate: competitor.metric.rate,
      }))
      .filter(({ rate: competitorRate }) => competitorRate !== null && competitorRate > 0)
      .sort((left, right) => (right.rate ?? -1) - (left.rate ?? -1))
      .slice(0, 3);
    return (
      <PromptRateBreakdown
        label={t("home:dashboard.mostMentionedBrands")}
        emptyLabel={t("home:dashboard.noBrandMentions")}
        items={top.map(({ competitor, rate }) => ({
          id: competitor.id,
          name: competitor.name,
          rate,
          icon: <BrandPill name={competitor.name} domain={competitor.domain} iconOnly />,
        }))}
      />
    );
  };

  const renderActions = (row: PromptRow) =>
    row.prompt ? (
      <div className="flex flex-wrap items-center gap-2">
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

  return (
    <section
      id="prompts"
      aria-labelledby="prompts-heading"
      className="min-w-0 rounded-lg bg-surface p-4 sm:p-7 scroll-mt-8"
    >
      <div className="mb-4 flex items-center justify-between gap-4">
        <h2 id="prompts-heading" ref={heading} tabIndex={-1} className="text-[13px] font-medium">
          {t("prompts")}
        </h2>
        <PromptDialog
          label={t("addPrompt")}
          pending={mutations.create.isPending}
          onSave={(request) => mutations.create.mutateAsync(request).then(() => undefined)}
          variant="ghost"
        />
      </div>
      <div className="[&:not(:empty)]:mb-3">
        <TrackingReview brandID={brandID} kind="prompts" disabled={pending} />
      </div>
      {mutations.confirm.isError && (
        <p role="alert" className="mb-3 text-sm text-destructive">
          {t("mutationError")}
        </p>
      )}
      {mutations.update.isError && (
        <p role="alert" className="mb-3 text-sm text-destructive">
          {t("mutationError")}
        </p>
      )}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <Input
          value={search.q}
          onChange={(event) => changeSearch({ q: event.target.value })}
          placeholder={t("searchPrompts")}
          aria-label={t("searchPrompts")}
          className="w-full max-w-80"
        />
        <PromptChoice
          label={t("statusLabel")}
          value={search.status}
          options={[
            { value: "all", label: t("all") },
            { value: "confirmed", label: t("confirmed") },
            { value: "unconfirmed", label: t("unconfirmed") },
            { value: "past", label: t("home:history.title") },
          ]}
          onChange={(value) => changeSearch({ status: value as typeof search.status })}
        />
        <PromptChoice
          label={t("home:dashboard.sortPrompts")}
          value={search.sort}
          options={[
            { value: "original", label: t("home:dashboard.originalOrder") },
            { value: "prompt", label: t("prompt") },
            { value: "rate", label: t("home:dashboard.mentionRate") },
            { value: "change", label: t("home:dashboard.sortChange") },
          ]}
          onChange={(value) => changeSearch({ sort: value as typeof search.sort })}
        />
        {search.sort !== "original" && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => changeSearch({ descending: !search.descending })}
          >
            {t(search.descending ? "home:dashboard.descending" : "home:dashboard.ascending")}
          </Button>
        )}
        <div className="[&>div]:m-0 [&_output:empty]:hidden">
          <GenerationControls
            brandID={brandID}
            mode="prompts"
            setupStartFailed={draftStartFailed}
            hideSuccessMessage
          />
        </div>
      </div>

      <output className="sr-only" aria-live="polite">
        {selectedVisible.length
          ? t("home:dashboard.selectedCount", { count: selectedVisible.length })
          : ""}
      </output>
      {selectedVisible.length > 0 && (
        <div className="fixed bottom-[max(24px,env(safe-area-inset-bottom))] left-1/2 z-30 flex w-max max-w-[calc(100vw-32px)] -translate-x-1/2 flex-wrap items-center justify-center gap-1 rounded-[12px] border border-[var(--metric-tooltip-border)] [background:linear-gradient(165deg,#ffffff0c,#ffffff02_55%),color-mix(in_srgb,var(--popover)_80%,transparent)] px-2.5 py-2 text-[13px] text-muted-foreground shadow-[0_16px_60px_var(--metric-tooltip-shadow)] backdrop-blur-[24px] backdrop-saturate-[130%]">
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
      {bulkFailed && !bulkDeleting && (
        <p role="alert" className="text-sm text-destructive">
          {t("home:dashboard.bulkError")}
        </p>
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
          <>
            <div className="hidden min-[1101px]:block">
              <Table
                aria-label={t("tableLabel")}
                className="prompt-table table-fixed border-separate border-spacing-y-1"
              >
                <TableHeader>
                  <TableRow className="border-0 hover:bg-transparent">
                    <TableHead className={`${promptHeadClass} w-[4%] text-center`}>
                      {visibleLiveIDs.length > 0 && (
                        <PromptSelectCheckbox
                          label={t("home:dashboard.selectPage")}
                          checked={
                            visibleLiveIDs.length > 0 &&
                            selectedVisible.length === visibleLiveIDs.length
                          }
                          indeterminate={
                            selectedVisible.length > 0 &&
                            selectedVisible.length < visibleLiveIDs.length
                          }
                          onChange={(checked) =>
                            setSelected((current) =>
                              checked
                                ? [...new Set([...current, ...visibleLiveIDs])]
                                : current.filter((id) => !visibleLiveIDs.includes(id)),
                            )
                          }
                        />
                      )}
                    </TableHead>
                    <TableHead className={`${promptHeadClass} w-[31%]`}>{t("prompt")}</TableHead>
                    <TableHead className={`${promptHeadClass} w-[23%]`}>
                      {t("home:dashboard.yourRate")}
                    </TableHead>
                    <TableHead className={`${promptHeadClass} w-[14%]`}>
                      {t("home:dashboard.byEngine")}
                    </TableHead>
                    <TableHead className={`${promptHeadClass} w-[19%]`}>
                      {t("home:dashboard.mostMentionedBrands")}
                    </TableHead>
                    <TableHead className={`${promptHeadClass} w-[9%]`}>{t("actions")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visible.map((row) => (
                    <TableRow
                      key={row.id}
                      className="group border-0 transition-none hover:bg-transparent"
                      data-selected={selectedVisible.includes(row.id) || undefined}
                    >
                      <TableCell className={`${promptCellClass} rounded-s-md text-center`}>
                        {!row.past && (
                          <PromptSelectCheckbox
                            label={t("home:dashboard.selectPrompt", { text: row.text })}
                            checked={selectedVisible.includes(row.id)}
                            onChange={() => select(row.id)}
                          />
                        )}
                      </TableCell>
                      <TableCell className={promptCellClass}>{renderPrompt(row)}</TableCell>
                      <TableCell className={promptCellClass}>{renderRate(row)}</TableCell>
                      <TableCell className={promptCellClass}>{renderEngines(row)}</TableCell>
                      <TableCell className={promptCellClass}>{renderBrands(row)}</TableCell>
                      <TableCell className={`${promptCellClass} rounded-e-md`}>
                        {renderActions(row)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            {visibleLiveIDs.length > 0 && (
              <div className="mb-2 flex items-center gap-2 text-xs text-muted-foreground min-[1101px]:hidden">
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
                {t("home:dashboard.selectPage")}
              </div>
            )}
            <ul className="grid gap-1 min-[1101px]:hidden">
              {visible.map((row) => (
                <li
                  key={row.id}
                  data-selected={selectedVisible.includes(row.id) || undefined}
                  className="min-w-0 rounded-md p-3 data-[selected]:bg-signal-pill"
                >
                  <div className="flex items-start gap-3">
                    {!row.past && (
                      <PromptSelectCheckbox
                        label={t("home:dashboard.selectPrompt", { text: row.text })}
                        checked={selectedVisible.includes(row.id)}
                        onChange={() => select(row.id)}
                      />
                    )}
                    <div className="min-w-0 flex-1">{renderPrompt(row)}</div>
                  </div>
                  <dl className="my-4 grid gap-3">
                    <div className="grid grid-cols-1 gap-1 min-[441px]:grid-cols-[120px_minmax(0,1fr)] min-[441px]:gap-3">
                      <dt className="text-xs font-normal text-muted-foreground">
                        {t("home:dashboard.yourRate")}
                      </dt>
                      <dd>{renderRate(row)}</dd>
                    </div>
                    <div className="grid grid-cols-1 gap-1 min-[441px]:grid-cols-[120px_minmax(0,1fr)] min-[441px]:gap-3">
                      <dt className="text-xs font-normal text-muted-foreground">
                        {t("home:dashboard.byEngine")}
                      </dt>
                      <dd>{renderEngines(row)}</dd>
                    </div>
                    <div className="grid grid-cols-1 gap-1 min-[441px]:grid-cols-[120px_minmax(0,1fr)] min-[441px]:gap-3">
                      <dt className="text-xs font-normal text-muted-foreground">
                        {t("home:dashboard.mostMentionedBrands")}
                      </dt>
                      <dd>{renderBrands(row)}</dd>
                    </div>
                  </dl>
                  {renderActions(row)}
                </li>
              ))}
            </ul>
          </>
        )}
      </TooltipProvider>
      <div className="mt-6 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
        <span className="whitespace-nowrap">
          {t("home:dashboard.page", { current: page, total: Math.max(1, totalPages) })}
        </span>
        <PromptChoice
          label={t("pageSizeLabel")}
          value={String(search.pageSize)}
          options={[10, 20, 50].map((value) => ({ value: String(value), label: String(value) }))}
          onChange={(value) => changeSearch({ pageSize: Number(value) as 10 | 20 | 50 })}
        />
        <div className="flex gap-2 sm:ms-auto">
          <Button
            variant="ghost"
            size="sm"
            disabled={page <= 1}
            onClick={() => void setSearch({ page: page - 1 }, { history: "push" })}
          >
            {t("home:dashboard.previous")}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => void setSearch({ page: page + 1 }, { history: "push" })}
          >
            {t("home:dashboard.next")}
          </Button>
        </div>
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
