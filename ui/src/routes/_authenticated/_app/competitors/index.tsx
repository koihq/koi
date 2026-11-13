import { Delete02Icon, PencilEdit02Icon, Tick02Icon } from "@hugeicons/core-free-icons";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Competitor, UpdateCompetitor } from "@/api/competitors.api";
import { ApiErrorMessage } from "@/components/blocks/api-error-message";
import { BackLink } from "@/components/blocks/back-link";
import { BrandLabel } from "@/components/blocks/brand-label";
import { DataTable, type DataTableColumn } from "@/components/blocks/data-table";
import { DeleteItemDialog } from "@/components/blocks/delete-item-dialog";
import { GenerationControls } from "@/components/blocks/generation-controls";
import { PageNav } from "@/components/blocks/page-nav";
import { QueryRefreshIndicator } from "@/components/blocks/query-refresh-indicator";
import { RowAction, RowActions } from "@/components/blocks/row-actions";
import { SearchToggle } from "@/components/blocks/search-toggle";
import { StatusTabs } from "@/components/blocks/status-tabs";
import { Button } from "@/components/ui/button";
import { useCompetitorCounts } from "@/hooks/use-competitor-counts";
import { validateTableSearch } from "@/hooks/use-table-state";
import { useTracking } from "@/hooks/use-tracking";
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
  const counts = useCompetitorCounts(brand.id);
  const tracking = useTracking(brand.id);
  const started = tracking.data?.confirmed !== false;
  const mutations = useCompetitorMutations(brand.id);
  const [editing, setEditing] = useState<Competitor>();
  const [deleting, setDeleting] = useState<Competitor>();
  const trigger = useRef<HTMLElement | null>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const pending = Object.values(mutations).some((mutation) => mutation.isPending);
  const items = useMemo(() => {
    const rates = new Map(
      summary.data?.competitors.map((competitor) => [competitor.id, competitor.metric.rate]),
    );

    // Most visible first, the same order the home page uses.
    return [...(list.competitors.data?.items ?? [])].sort(
      (left, right) => (rates.get(right.id) ?? -1) - (rates.get(left.id) ?? -1),
    );
  }, [list.competitors.data, summary.data]);
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
      <span className="rounded-chip bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
        {t(item.source === "observed" ? "suggested" : started ? "unconfirmed" : "drafted")}
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
      <span
        className={`tabular-nums ${value === null ? "text-muted-foreground" : "font-semibold"}`}
      >
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

  const headClass = "h-auto px-3 py-1.5 text-xs text-muted-foreground";
  const cellClass = "px-3 py-3.5 text-body";
  const columns: DataTableColumn<Competitor>[] = [
    {
      id: "name",
      header: t("name"),
      headClassName: `${headClass} w-[34%]`,
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
      headClassName: `${headClass} w-[30%]`,
      cellClassName: `${cellClass} text-muted-foreground`,
      cell: (item) => {
        const identity = item.domain || item.githubRepo || t("competitorIdentityNeeded");

        return (
          <span className="block truncate" title={identity}>
            {identity}
          </span>
        );
      },
    },
    {
      id: "rate",
      header: t("home:dashboard.visibility"),
      headClassName: `${headClass} w-[13%] text-end`,
      cellClassName: `${cellClass} text-end`,
      cell: renderRate,
    },
    {
      id: "gap",
      header: t("home:dashboard.vsYou"),
      headClassName: `${headClass} w-[13%] text-end`,
      cellClassName: `${cellClass} text-end`,
      cell: renderGap,
    },
    {
      id: "actions",
      header: <span className="sr-only">{t("actions")}</span>,
      headClassName: `${headClass} w-28`,
      cellClassName: `${cellClass} ps-0! pe-1! text-end`,
      cell: (item) => renderActions(item),
    },
  ];

  function renderActions(item: Competitor) {
    const identified = Boolean(item.domain || item.githubRepo);
    const dismissable = item.source === "observed" && !item.confirmed;

    return (
      <RowActions>
        {started && !item.confirmed && identified && (
          <RowAction
            persistent
            label={t("confirm")}
            icon={Tick02Icon}
            disabled={pending}
            onClick={() => mutations.confirm.mutate(item.id)}
          />
        )}
        <RowAction
          persistent={!identified}
          label={t(identified ? "edit" : "addIdentity")}
          icon={PencilEdit02Icon}
          disabled={pending}
          onClick={() => edit(item)}
        />
        <RowAction
          destructive
          label={t(dismissable ? "dismiss" : "delete")}
          icon={Delete02Icon}
          disabled={pending}
          onClick={() => remove(item)}
        />
      </RowActions>
    );
  }

  return (
    <div className="grid gap-6 pb-24">
      <BackLink label={t("home:dashboard.backToVisibility")} />
      <section
        className="group/refresh relative min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9"
        aria-busy={list.competitors.isFetching}
        data-refreshing={list.competitors.isFetching && Boolean(list.competitors.data)}
        aria-labelledby="competitors-heading"
      >
        <QueryRefreshIndicator active={list.competitors.isFetching} label={t("loading")} />
        <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
            <h1
              id="competitors-heading"
              ref={heading}
              tabIndex={-1}
              className="text-section font-semibold"
            >
              {t("competitors")}
            </h1>
            {started && (
              <StatusTabs
                label={t("statusLabel")}
                value={list.table.status}
                options={[
                  { value: "all", label: t("all"), count: counts.all },
                  { value: "confirmed", label: t("confirmed"), count: counts.confirmed },
                  ...(counts.unconfirmed || list.table.status === "unconfirmed"
                    ? [
                        {
                          value: "unconfirmed" as const,
                          label: t("unconfirmed"),
                          count: counts.unconfirmed,
                        },
                      ]
                    : []),
                ]}
                onChange={(status) => list.table.setStatus(status)}
              />
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <SearchToggle
              label={t("searchCompetitors")}
              clearLabel={t("home:dashboard.clearSearch")}
              value={list.table.q}
              onChange={(q) => list.table.setSearch(q)}
            />
            <CompetitorDialog
              brandID={brand.id}
              label={t("addCompetitor")}
              pending={mutations.create.isPending}
              onSave={(request) => mutations.create.mutateAsync(request).then(() => undefined)}
            />
          </div>
        </div>
        <div className="[&:not(:empty)]:mb-4">
          <GenerationControls brandID={brand.id} />
        </div>
        {mutations.confirm.isError && (
          <p role="alert" className="mb-4 text-sm text-destructive">
            <ApiErrorMessage error={mutations.confirm.error} fallback={t("mutationError")} />
          </p>
        )}
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
            <div className="-mx-3 hidden @2xl:block">
              <DataTable
                interactive
                ariaLabel={t("competitors")}
                className="table-fixed"
                data={items}
                columns={columns}
                getRowId={(item) => item.id}
              />
            </div>
            <ul className="list-rows -mx-2 grid gap-0.5 @2xl:hidden">
              {items.map((item) => (
                <li
                  key={item.id}
                  className="group flex min-w-0 items-center gap-3 rounded-control px-2 py-3 hover:bg-signal-pill"
                >
                  <div className="grid min-w-0 flex-1 grid-cols-[minmax(0,1fr)] gap-1">
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
        <PageNav
          page={list.table.page}
          pageSize={list.table.pageSize}
          total={response?.totalItems ?? 0}
          hasPrev={Boolean(response?.hasPrev)}
          hasNext={Boolean(response?.hasNext)}
          onPage={(page) => list.table.setPage(page)}
        />
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
