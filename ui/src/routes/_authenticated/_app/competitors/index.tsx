import { ArrowLeft01Icon, ArrowRight01Icon, MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Competitor, UpdateCompetitor } from "@/api/competitors.api";
import { BrandLabel } from "@/components/blocks/brand-label";
import { DataTable, type DataTableColumn } from "@/components/blocks/data-table";
import { DeleteItemDialog } from "@/components/blocks/delete-item-dialog";
import { GenerationControls } from "@/components/blocks/generation-controls";
import { QueryRefreshIndicator } from "@/components/blocks/query-refresh-indicator";
import { TrackingReview } from "@/components/blocks/tracking-review";
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
import { validateTableSearch } from "@/hooks/use-table-state";
import { formatPoints, formatRate } from "@/lib/format";
import { visibilitySummaryOptions } from "@/lib/queries";
import { rate as visibilityRate } from "@/lib/visibility";
import { CompetitorDialog } from "@/routes/_authenticated/_app/competitors/-components/competitor-dialog";
import { useCompetitorList } from "@/routes/_authenticated/_app/competitors/-hooks/use-competitor-list";
import { useCompetitorMutations } from "@/routes/_authenticated/_app/competitors/-hooks/use-competitor-mutations";

export const Route = createFileRoute("/_authenticated/_app/competitors/")({
  validateSearch: validateTableSearch,
  component: CompetitorPage,
});

function CompetitorPage() {
  const { brand } = Route.useRouteContext();
  const { t, i18n } = useTranslation(["management", "home"]);
  const api = useApi();
  const summary = useQuery(visibilitySummaryOptions(api, brand.id, { days: 30, engines: [] }));
  const ownRate = visibilityRate(summary.data);
  const list = useCompetitorList(brand.id);
  const mutations = useCompetitorMutations(brand.id);
  const [editing, setEditing] = useState<Competitor>();
  const [deleting, setDeleting] = useState<Competitor>();
  const trigger = useRef<HTMLElement | null>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const pending = Object.values(mutations).some((mutation) => mutation.isPending);
  const items = list.competitors.data?.items ?? [];
  const response = list.competitors.data;
  const dismissing = deleting?.source === "observed" && !deleting.confirmed;

  function rivalRate(id: string) {
    return (
      summary.data?.competitors.find((competitor) => competitor.id === id)?.metric.rate ?? null
    );
  }

  function renderStatus(item: Competitor) {
    if (item.confirmed) return null;

    return (
      <span className="rounded-sm bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
        {t(item.source === "observed" ? "suggested" : "unconfirmed")}
      </span>
    );
  }

  function renderGap(item: Competitor) {
    const value = rivalRate(item.id);
    const gap = value !== null && ownRate !== null ? value - ownRate : null;

    return (
      <span
        className={`tabular-nums ${gap === null ? "text-muted-foreground" : gap > 0 ? "text-negative" : "text-positive"}`}
      >
        {gap === null ? "—" : formatPoints(gap, i18n.language)}
      </span>
    );
  }

  function renderRate(item: Competitor) {
    const value = rivalRate(item.id);

    return (
      <span className={`tabular-nums ${value === null ? "text-muted-foreground" : ""}`}>
        {value === null ? "—" : formatRate(value, i18n.language)}
      </span>
    );
  }

  function edit(item: Competitor) {
    trigger.current = document.activeElement as HTMLElement;
    setEditing(item);
  }

  function remove(item: Competitor) {
    trigger.current = document.activeElement as HTMLElement;
    mutations.remove.reset();
    setDeleting(item);
  }

  function update(id: string, request: UpdateCompetitor) {
    return mutations.update.mutateAsync({ id, request }).then(() => undefined);
  }

  const headClass = "h-auto px-3 py-1.5 text-xs font-normal text-muted-foreground";
  const cellClass = "px-3 py-3.5 text-body";
  const columns: DataTableColumn<Competitor>[] = [
    {
      id: "name",
      header: t("name"),
      headClassName: `${headClass} w-[32%]`,
      cellClassName: `${cellClass} whitespace-normal [overflow-wrap:anywhere]`,
      cell: (item) => (
        <span className="flex flex-wrap items-center gap-2">
          <BrandLabel name={item.name} domain={item.domain} />
          {renderStatus(item)}
        </span>
      ),
    },
    {
      id: "identity",
      header: t("domain"),
      headClassName: `${headClass} w-[26%]`,
      cellClassName: `${cellClass} whitespace-normal break-all text-muted-foreground`,
      cell: (item) => item.domain || item.githubRepo || t("competitorIdentityNeeded"),
    },
    {
      id: "rate",
      header: t("home:dashboard.visibility"),
      headClassName: `${headClass} w-[14%] text-end`,
      cellClassName: `${cellClass} text-end`,
      cell: renderRate,
    },
    {
      id: "gap",
      header: t("home:dashboard.vsYou"),
      headClassName: `${headClass} w-[14%] text-end`,
      cellClassName: `${cellClass} text-end`,
      cell: renderGap,
    },
    {
      id: "actions",
      header: <span className="sr-only">{t("actions")}</span>,
      headClassName: `${headClass} w-[14%] text-end`,
      cellClassName: `${cellClass} text-end`,
      cell: (item) => renderActions(item),
    },
  ];

  function renderActions(item: Competitor) {
    return (
      <div className="inline-flex items-center gap-2">
        {!item.domain && !item.githubRepo && (
          <Button variant="ghost" size="sm" onClick={() => edit(item)}>
            {t("addIdentity")}
          </Button>
        )}
        <DropdownMenu>
          <DropdownMenuTrigger
            aria-label={t("home:dashboard.competitorActions", { name: item.name })}
            className="grid size-8 place-items-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring"
          >
            <HugeiconsIcon icon={MoreHorizontalIcon} size={18} aria-hidden="true" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {!item.confirmed && (item.domain || item.githubRepo) && (
              <DropdownMenuItem
                disabled={pending}
                onClick={() => mutations.confirm.mutate(item.id)}
              >
                {t("confirm")}
              </DropdownMenuItem>
            )}
            <DropdownMenuItem disabled={pending} onClick={() => edit(item)}>
              {t("edit")}
            </DropdownMenuItem>
            <DropdownMenuItem variant="destructive" disabled={pending} onClick={() => remove(item)}>
              {t(item.source === "observed" && !item.confirmed ? "dismiss" : "delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    );
  }

  return (
    <div className="pb-24">
      <Link
        to="/"
        className="text-xs font-normal text-muted-foreground mb-6 inline-block hover:text-foreground"
      >
        {t("home:dashboard.backToVisibility")}
      </Link>
      <section
        className="group/refresh relative min-w-0 rounded-lg bg-surface p-4 sm:p-7"
        aria-busy={list.competitors.isFetching}
        data-refreshing={list.competitors.isFetching && Boolean(list.competitors.data)}
        aria-labelledby="competitors-heading"
      >
        <QueryRefreshIndicator active={list.competitors.isFetching} label={t("loading")} />
        <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
          <h1
            id="competitors-heading"
            ref={heading}
            tabIndex={-1}
            className="text-section font-semibold"
          >
            {t("competitors")}
          </h1>
          <div className="flex flex-wrap items-center gap-2 [&>div]:m-0 [&_output:empty]:hidden">
            <GenerationControls brandID={brand.id} mode="competitors" />
            <CompetitorDialog
              brandID={brand.id}
              label={t("addCompetitor")}
              pending={mutations.create.isPending}
              triggerVariant="outline"
              onSave={(request) => mutations.create.mutateAsync(request).then(() => undefined)}
            />
          </div>
        </div>
        <div className="[&:not(:empty)]:mb-4">
          <TrackingReview brandID={brand.id} kind="competitors" disabled={pending} />
        </div>
        {mutations.confirm.isError && (
          <p role="alert" className="mb-4 text-sm text-destructive">
            {t("mutationError")}
          </p>
        )}
        <div className="mb-4 flex flex-wrap items-center gap-x-4 gap-y-3">
          <Input
            type="search"
            value={list.table.q}
            onChange={(event) => list.table.setSearch(event.target.value)}
            aria-label={t("searchCompetitors")}
            placeholder={t("searchCompetitors")}
            className="w-full sm:max-w-64"
          />
          <fieldset className="flex flex-wrap gap-0.5" aria-label={t("statusLabel")}>
            {(["all", "confirmed", "unconfirmed"] as const).map((status) => (
              <Button
                key={status}
                type="button"
                variant="ghost"
                size="pill"
                aria-pressed={list.table.status === status}
                className="text-muted-foreground"
                onClick={() => list.table.setStatus(status)}
              >
                {t(status)}
              </Button>
            ))}
          </fieldset>
        </div>
        {list.competitors.isError ? (
          <p role="alert" className="py-8 text-sm text-destructive">
            {t("loadError")}{" "}
            <Button variant="link" onClick={() => void list.competitors.refetch()}>
              {t("retry")}
            </Button>
          </p>
        ) : list.competitors.isPending ? (
          <p className="py-8 text-sm text-muted-foreground">{t("loading")}</p>
        ) : items.length ? (
          <div className="@container group-data-[refreshing=true]/refresh:opacity-50">
            <div className="hidden @2xl:block">
              <DataTable
                interactive
                ariaLabel={t("competitors")}
                className="table-fixed"
                data={items}
                columns={columns}
                getRowId={(item) => item.id}
              />
            </div>
            <ul className="list-rows grid gap-0.5 @2xl:hidden">
              {items.map((item) => (
                <li
                  key={item.id}
                  className="flex min-w-0 items-center gap-3 rounded-md px-2 py-3 hover:bg-signal-pill"
                >
                  <div className="grid min-w-0 flex-1 gap-1">
                    <span className="flex flex-wrap items-center gap-2">
                      <BrandLabel name={item.name} domain={item.domain} />
                      {renderStatus(item)}
                    </span>
                    <span className="flex gap-3 text-xs">
                      {renderRate(item)}
                      {renderGap(item)}
                    </span>
                  </div>
                  {renderActions(item)}
                </li>
              ))}
            </ul>
          </div>
        ) : (
          <p className="py-8 text-sm text-muted-foreground">
            {list.table.q || list.table.status !== "all"
              ? t("noStatusResults")
              : t("competitorEmpty")}
          </p>
        )}
        {response && response.totalItems > 0 && (
          <div className="mt-6 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
            <span className="whitespace-nowrap tabular-nums">
              {t("home:dashboard.pageRange", {
                start: (list.table.page - 1) * list.table.pageSize + 1,
                end: Math.min(response.totalItems, list.table.page * list.table.pageSize),
                total: response.totalItems,
              })}
            </span>
            <Select
              value={list.table.pageSize}
              onValueChange={(value) => {
                if (value !== null) list.table.setPageSize(value);
              }}
            >
              <SelectTrigger aria-label={t("pageSizeLabel")}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={10}>{10}</SelectItem>
                <SelectItem value={20}>{20}</SelectItem>
                <SelectItem value={50}>{50}</SelectItem>
              </SelectContent>
            </Select>
            <div className="ms-auto flex gap-1">
              <Button
                variant="ghost"
                size="circle-sm"
                aria-label={t("home:dashboard.previous")}
                disabled={!response.hasPrev}
                onClick={() => list.table.setPage(list.table.page - 1)}
              >
                <HugeiconsIcon icon={ArrowLeft01Icon} aria-hidden="true" />
              </Button>
              <Button
                variant="ghost"
                size="circle-sm"
                aria-label={t("home:dashboard.next")}
                disabled={!response.hasNext}
                onClick={() => list.table.setPage(list.table.page + 1)}
              >
                <HugeiconsIcon icon={ArrowRight01Icon} aria-hidden="true" />
              </Button>
            </div>
          </div>
        )}
      </section>
      {editing && (
        <CompetitorDialog
          key={editing.id}
          brandID={brand.id}
          competitor={editing}
          label={t("edit")}
          pending={mutations.update.isPending}
          hideTrigger
          open={Boolean(editing)}
          onOpenChange={(open) => {
            if (!open) setEditing(undefined);
          }}
          onSave={(request) => update(editing.id, request)}
        />
      )}
      <DeleteItemDialog
        itemID={deleting?.id}
        title={t(dismissing ? "dismissSuggestionTitle" : "deleteCompetitorTitle")}
        description={t(dismissing ? "dismissSuggestionDescription" : "competitorDeleteDescription")}
        actionLabel={t(dismissing ? "dismiss" : "deleteCompetitor")}
        onClose={() => setDeleting(undefined)}
        pending={mutations.remove.isPending}
        failed={mutations.remove.isError}
        onDelete={mutations.remove.mutateAsync}
        finalFocus={() => (trigger.current?.isConnected ? trigger.current : heading.current)}
      />
    </div>
  );
}
