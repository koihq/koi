import { useState } from "react";
import { useTranslation } from "react-i18next";

import type { CreatePrompt, Prompt } from "@/api/prompts.api";
import type { ApiListResponse } from "@/api/shared/types";
import { ListPagination } from "@/components/blocks/list-pagination";
import { Button } from "@/components/ui/button";
import { PromptForm } from "@/routes/_authenticated/_settings/settings/prompts/-components/prompt-form";

export function PromptEditor({
  prompts,
  unconfirmedItems,
  onPageChange,
  onCreate,
  onUpdate,
  onDelete,
  onConfirm,
  onConfirmAll,
  pending,
}: {
  prompts: ApiListResponse<Prompt>;
  unconfirmedItems: number;
  onPageChange: (page: number) => void;
  onCreate: (request: CreatePrompt) => Promise<unknown>;
  onUpdate: (id: string, request: CreatePrompt) => Promise<unknown>;
  onDelete: (id: string) => void;
  onConfirm: (id: string) => void;
  onConfirmAll: () => void;
  pending: {
    create: boolean;
    update: boolean;
    delete: boolean;
    confirm: boolean;
    confirmAll: boolean;
  };
}) {
  const { t } = useTranslation("brand");
  const [editingID, setEditingID] = useState<string>();

  return (
    <section aria-labelledby="prompts-heading">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 id="prompts-heading" className="text-lg font-medium tracking-tight">
            {t("lists.prompts.title")}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("lists.prompts.description")}</p>
        </div>
        {unconfirmedItems > 0 ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={pending.confirmAll}
            onClick={onConfirmAll}
          >
            {t("lists.confirmAll")}
          </Button>
        ) : null}
      </div>
      <div className="mt-6">
        <PromptForm
          pending={pending.create}
          onSave={async (request) => {
            await onCreate(request);
          }}
        />
      </div>
      {prompts.items.length === 0 ? (
        <p className="mt-6 text-sm text-muted-foreground">{t("lists.prompts.empty")}</p>
      ) : (
        <ul className="mt-6 space-y-5">
          {prompts.items.map((prompt) => (
            <li key={prompt.id}>
              {editingID === prompt.id ? (
                <PromptForm
                  initialText={prompt.text}
                  pending={pending.update}
                  onCancel={() => setEditingID(undefined)}
                  onSave={async (request) => {
                    await onUpdate(prompt.id, request);
                    setEditingID(undefined);
                  }}
                />
              ) : (
                <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <p>{prompt.text}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {prompt.source === "manual"
                        ? t("lists.source.manual")
                        : t("lists.source.drafted")}
                      {" · "}
                      {prompt.confirmed ? t("lists.confirmed") : t("lists.unconfirmed")}
                    </p>
                  </div>
                  <div className="flex shrink-0 flex-wrap gap-1">
                    {!prompt.confirmed ? (
                      <Button
                        type="button"
                        size="xs"
                        variant="outline"
                        disabled={pending.confirm}
                        onClick={() => onConfirm(prompt.id)}
                      >
                        {t("lists.confirm")}
                      </Button>
                    ) : null}
                    <Button
                      type="button"
                      size="xs"
                      variant="ghost"
                      onClick={() => setEditingID(prompt.id)}
                    >
                      {t("lists.edit")}
                    </Button>
                    <Button
                      type="button"
                      size="xs"
                      variant="ghost"
                      disabled={pending.delete}
                      onClick={() => {
                        if (window.confirm(t("lists.prompts.deleteConfirm"))) onDelete(prompt.id);
                      }}
                    >
                      {t("lists.delete")}
                    </Button>
                  </div>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
      <ListPagination
        currentPage={prompts.currentPage}
        totalPages={prompts.totalPages}
        onPageChange={onPageChange}
      />
    </section>
  );
}
