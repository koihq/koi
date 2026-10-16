import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Competitor, UpdateCompetitor } from "@/api/competitors.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { DeleteItemDialog } from "@/components/blocks/delete-item-dialog";
import { GenerationControls } from "@/components/blocks/generation-controls";
import { TrackingReview } from "@/components/blocks/tracking-review";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { validateTableSearch } from "@/hooks/use-table-state";
import { CompetitorDialog } from "@/routes/_authenticated/_app/competitors/-components/competitor-dialog";
import { useCompetitorList } from "@/routes/_authenticated/_app/competitors/-hooks/use-competitor-list";
import { useCompetitorMutations } from "@/routes/_authenticated/_app/competitors/-hooks/use-competitor-mutations";

export const Route = createFileRoute("/_authenticated/_app/competitors/")({
  validateSearch: validateTableSearch,
  component: CompetitorPage,
});

function CompetitorPage() {
  const { brand } = Route.useRouteContext();
  const { t } = useTranslation(["management", "home"]);
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

  return (
    <div className="pb-24">
      <Link
        to="/"
        className="text-xs font-normal text-muted-foreground mb-6 inline-block hover:text-foreground"
      >
        {t("home:dashboard.backToVisibility")}
      </Link>
      <section
        className="min-w-0 rounded-lg bg-surface p-4 sm:p-7"
        aria-labelledby="competitors-heading"
      >
        <div className="mb-4 flex items-center justify-between gap-4">
          <h1 id="competitors-heading" ref={heading} tabIndex={-1} className="text-base font-bold">
            {t("competitors")}
          </h1>
          <CompetitorDialog
            brandID={brand.id}
            label={t("addCompetitor")}
            pending={mutations.create.isPending}
            triggerVariant="ghost"
            onSave={(request) => mutations.create.mutateAsync(request).then(() => undefined)}
          />
        </div>
        <div className="[&:not(:empty)]:mb-3">
          <TrackingReview brandID={brand.id} kind="competitors" disabled={pending} />
        </div>
        {mutations.confirm.isError && (
          <p role="alert" className="mb-3 text-sm text-destructive">
            {t("mutationError")}
          </p>
        )}
        <div className="mb-4 flex flex-wrap items-center gap-3 [&>select]:w-[88px]">
          <Input
            value={list.table.q}
            onChange={(event) => list.table.setSearch(event.target.value)}
            aria-label={t("searchCompetitors")}
            placeholder={t("searchCompetitors")}
            className="w-full max-w-80"
          />
          <select
            aria-label={t("statusLabel")}
            value={list.table.status}
            onChange={(event) =>
              list.table.setStatus(event.target.value as typeof list.table.status)
            }
            className="min-h-[34px] rounded-md border border-border bg-background px-3 text-[13px] text-foreground"
          >
            <option value="all">{t("all")}</option>
            <option value="confirmed">{t("confirmed")}</option>
            <option value="unconfirmed">{t("unconfirmed")}</option>
          </select>
          <div className="[&>div]:m-0 [&_output:empty]:hidden">
            <GenerationControls brandID={brand.id} mode="competitors" />
          </div>
        </div>
        {list.competitors.isError ? (
          <p role="alert" className="py-8 text-sm text-destructive">
            {t("loadError")}{" "}
            <Button variant="link" onClick={() => void list.competitors.refetch()}>
              {t("retry")}
            </Button>
          </p>
        ) : list.competitors.isPending || list.competitors.isPlaceholderData ? (
          <p className="py-8 text-sm text-muted-foreground">{t("loading")}</p>
        ) : items.length ? (
          <ul className="grid gap-1">
            {items.map((item) => (
              <li
                key={item.id}
                className="flex min-h-[68px] items-center justify-between gap-4 rounded-md p-2 hover:bg-muted focus-within:bg-muted"
              >
                <div className="min-w-0">
                  <BrandPill name={item.name} domain={item.domain} />
                  <p className="text-xs font-normal text-muted-foreground mt-2 break-all">
                    {item.domain || item.githubRepo || t("competitorIdentityNeeded")}
                  </p>
                  {item.source === "observed" && !item.confirmed && (
                    <p className="text-xs font-normal text-muted-foreground mt-1">
                      {t("observed")}
                    </p>
                  )}
                </div>
                <div className="flex items-center gap-3">
                  {!item.domain && !item.githubRepo && (
                    <Button variant="ghost" size="sm" onClick={() => edit(item)}>
                      {t("addIdentity")}
                    </Button>
                  )}
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      aria-label={t("home:dashboard.competitorActions", { name: item.name })}
                      className="grid size-8 place-items-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
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
                      <DropdownMenuItem
                        variant="destructive"
                        disabled={pending}
                        onClick={() => remove(item)}
                      >
                        {t(item.source === "observed" && !item.confirmed ? "dismiss" : "delete")}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <p className="py-8 text-sm text-muted-foreground">
            {list.table.q || list.table.status !== "all"
              ? t("noStatusResults")
              : t("competitorEmpty")}
          </p>
        )}
        <div className="mt-6 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
          <span className="whitespace-nowrap">
            {t("home:dashboard.page", {
              current: list.table.page,
              total: Math.max(1, response?.totalPages ?? 0),
            })}
          </span>
          <select
            aria-label={t("pageSizeLabel")}
            value={list.table.pageSize}
            onChange={(event) => list.table.setPageSize(Number(event.target.value) as 10 | 20 | 50)}
            className="min-h-[34px] rounded-md border border-border bg-background px-3 text-[13px] text-foreground"
          >
            <option value={10}>{"10"}</option>
            <option value={20}>{"20"}</option>
            <option value={50}>{"50"}</option>
          </select>
          <div className="flex gap-2 sm:ms-auto">
            <Button
              variant="ghost"
              size="sm"
              disabled={!response?.hasPrev}
              onClick={() => list.table.setPage(list.table.page - 1)}
            >
              {t("home:dashboard.previous")}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              disabled={!response?.hasNext}
              onClick={() => list.table.setPage(list.table.page + 1)}
            >
              {t("home:dashboard.next")}
            </Button>
          </div>
        </div>
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
