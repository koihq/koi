import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import type { Competitor, CreateCompetitor } from "@/api/competitors.api";
import { DataTable } from "@/components/blocks/data-table";
import { Button } from "@/components/ui/button";
import { validateTableSearch } from "@/hooks/use-table-state";
import { CompetitorDialog } from "@/routes/_authenticated/_app/competitors/-components/competitor-dialog";
import { DeleteCompetitorDialog } from "@/routes/_authenticated/_app/competitors/-components/delete-competitor-dialog";
import { useCompetitorColumns } from "@/routes/_authenticated/_app/competitors/-hooks/use-competitor-columns";
import { useCompetitorList } from "@/routes/_authenticated/_app/competitors/-hooks/use-competitor-list";
import { useCompetitorMutations } from "@/routes/_authenticated/_app/competitors/-hooks/use-competitor-mutations";

export const Route = createFileRoute("/_authenticated/_app/competitors/")({
  validateSearch: validateTableSearch,
  component: CompetitorPage,
});

function CompetitorPage() {
  const { brand } = Route.useRouteContext();
  const { t } = useTranslation("management");
  const list = useCompetitorList(brand.id);
  const mutations = useCompetitorMutations(brand.id);
  const [deleting, setDeleting] = useState<Competitor>();
  const deleteTrigger = useRef<HTMLElement | null>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const { mutateAsync: updateAsync } = mutations.update;
  const { mutate: confirmMutation } = mutations.confirm;
  const { reset: resetDelete } = mutations.remove;
  const chooseDelete = useCallback(
    (item: Competitor) => {
      deleteTrigger.current = document.activeElement as HTMLElement;
      resetDelete();
      setDeleting(item);
    },
    [resetDelete],
  );
  const pending =
    mutations.create.isPending ||
    mutations.update.isPending ||
    mutations.remove.isPending ||
    mutations.confirm.isPending;
  const update = useCallback(
    async (id: string, request: CreateCompetitor) => {
      await updateAsync({ id, request });
    },
    [updateAsync],
  );
  const confirm = useCallback((id: string) => confirmMutation(id), [confirmMutation]);
  const columns = useCompetitorColumns({
    pending,
    onUpdate: update,
    onDelete: chooseDelete,
    onConfirm: confirm,
  });

  return (
    <section aria-labelledby="competitors-heading" className="mx-auto max-w-6xl">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2
            ref={heading}
            tabIndex={-1}
            id="competitors-heading"
            className="text-2xl font-medium tracking-tight"
          >
            {t("competitors")}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("competitorDescription")}</p>
        </div>
        {(list.unconfirmed.data?.totalItems ?? 0) > 0 ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={mutations.confirmAll.isPending}
            onClick={() => mutations.confirmAll.mutate()}
          >
            {t("confirmAllCount", { count: list.unconfirmed.data?.totalItems ?? 0 })}
          </Button>
        ) : null}
      </div>
      {mutations.confirm.isError || mutations.confirmAll.isError ? (
        <p role="alert" className="mb-4 text-sm text-destructive">
          {t("mutationError")}
        </p>
      ) : null}
      {list.unconfirmed.isError ? (
        <p role="alert" className="mb-4 text-sm text-destructive">
          {t("unconfirmedError")}
        </p>
      ) : null}
      <DataTable
        response={list.competitors.data}
        columns={columns}
        page={list.table.page}
        loading={
          list.competitors.isPending ||
          list.competitors.isPlaceholderData ||
          list.table.q !== list.table.debouncedQuery
        }
        failed={list.competitors.isError}
        onRetry={() => void list.competitors.refetch()}
        q={list.table.q}
        status={list.table.status}
        pageSize={list.table.pageSize}
        searchPlaceholder={t("searchCompetitors")}
        addAction={
          <CompetitorDialog
            label={t("addCompetitor")}
            pending={mutations.create.isPending}
            onSave={(request) => mutations.create.mutateAsync(request).then(() => undefined)}
          />
        }
        onSearchChange={list.table.setSearch}
        onStatusChange={list.table.setStatus}
        onPageSizeChange={list.table.setPageSize}
        onPageChange={list.table.setPage}
        onClearFilters={list.table.clearFilters}
        emptyCollection={
          <div className="space-y-3">
            <p>{t("competitorEmpty")}</p>
            <CompetitorDialog
              label={t("addFirstCompetitor")}
              pending={mutations.create.isPending}
              onSave={(request) => mutations.create.mutateAsync(request).then(() => undefined)}
            />
          </div>
        }
      />
      <DeleteCompetitorDialog
        item={deleting}
        onClose={() => setDeleting(undefined)}
        pending={mutations.remove.isPending}
        failed={mutations.remove.isError}
        onDelete={mutations.remove.mutateAsync}
        finalFocus={() =>
          deleteTrigger.current?.isConnected ? deleteTrigger.current : heading.current
        }
      />
    </section>
  );
}
