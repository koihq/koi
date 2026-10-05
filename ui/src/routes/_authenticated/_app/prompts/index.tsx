import { createFileRoute } from "@tanstack/react-router";
import { createStandardSchemaV1, parseAsStringLiteral } from "nuqs";
import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import type { Prompt } from "@/api/prompts.api";
import { DataTable } from "@/components/blocks/data-table";
import { Button } from "@/components/ui/button";
import { tableSearchParams } from "@/hooks/use-table-state";
import { GenerationControls } from "@/components/blocks/generation-controls";
import { DeletePromptDialog } from "@/routes/_authenticated/_app/prompts/-components/delete-prompt-dialog";
import { PromptDialog } from "@/routes/_authenticated/_app/prompts/-components/prompt-dialog";
import { usePromptColumns } from "@/routes/_authenticated/_app/prompts/-hooks/use-prompt-columns";
import { usePromptList } from "@/routes/_authenticated/_app/prompts/-hooks/use-prompt-list";
import { usePromptMutations } from "@/routes/_authenticated/_app/prompts/-hooks/use-prompt-mutations";

export const Route = createFileRoute("/_authenticated/_app/prompts/")({
  validateSearch: createStandardSchemaV1(
    { ...tableSearchParams, draftStart: parseAsStringLiteral(["failed"] as const) },
    { partialOutput: true },
  ),
  component: PromptPage,
});

function PromptPage() {
  const { brand } = Route.useRouteContext();
  const { draftStart } = Route.useSearch();
  const { t } = useTranslation("management");
  const list = usePromptList(brand.id);
  const mutations = usePromptMutations(brand.id);
  const [deleting, setDeleting] = useState<Prompt>();
  const deleteTrigger = useRef<HTMLElement | null>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const { mutateAsync: updateAsync } = mutations.update;
  const { mutate: confirmMutation } = mutations.confirm;
  const { reset: resetDelete } = mutations.remove;
  const chooseDelete = useCallback(
    (item: Prompt) => {
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
    async (id: string, request: { text: string }) => {
      await updateAsync({ id, request });
    },
    [updateAsync],
  );
  const confirm = useCallback((id: string) => confirmMutation(id), [confirmMutation]);
  const columns = usePromptColumns({
    pending,
    onUpdate: update,
    onDelete: chooseDelete,
    onConfirm: confirm,
  });
  const add = (
    <PromptDialog
      label={t("addPrompt")}
      pending={mutations.create.isPending}
      onSave={(request) => mutations.create.mutateAsync(request).then(() => undefined)}
    />
  );

  return (
    <section aria-labelledby="prompts-heading" className="mx-auto max-w-6xl">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2
            ref={heading}
            tabIndex={-1}
            id="prompts-heading"
            className="text-2xl font-medium tracking-tight"
          >
            {t("prompts")}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("promptDescription")}</p>
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
      <GenerationControls
        brandID={brand.id}
        mode="prompts"
        setupStartFailed={draftStart === "failed"}
      />
      <DataTable
        response={list.prompts.data}
        columns={columns}
        page={list.table.page}
        loading={
          list.prompts.isPending ||
          list.prompts.isPlaceholderData ||
          list.table.q !== list.table.debouncedQuery
        }
        failed={list.prompts.isError}
        onRetry={() => void list.prompts.refetch()}
        q={list.table.q}
        status={list.table.status}
        pageSize={list.table.pageSize}
        searchPlaceholder={t("searchPrompts")}
        addAction={add}
        onSearchChange={list.table.setSearch}
        onStatusChange={list.table.setStatus}
        onPageSizeChange={list.table.setPageSize}
        onPageChange={list.table.setPage}
        onClearFilters={list.table.clearFilters}
        emptyCollection={
          <div className="space-y-3">
            <p>{t("promptEmpty")}</p>
            <PromptDialog
              label={t("addFirstPrompt")}
              pending={mutations.create.isPending}
              onSave={(request) => mutations.create.mutateAsync(request).then(() => undefined)}
            />
          </div>
        }
      />
      <DeletePromptDialog
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
