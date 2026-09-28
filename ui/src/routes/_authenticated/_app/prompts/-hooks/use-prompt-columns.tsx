import type { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import type { CreatePrompt, Prompt } from "@/api/prompts.api";
import { Button } from "@/components/ui/button";
import { PromptDialog } from "@/routes/_authenticated/_app/prompts/-components/prompt-dialog";

export function usePromptColumns({
  pending,
  onUpdate,
  onDelete,
  onConfirm,
}: {
  pending: boolean;
  onUpdate: (id: string, request: CreatePrompt) => Promise<void>;
  onDelete: (prompt: Prompt) => void;
  onConfirm: (id: string) => void;
}): ColumnDef<Prompt>[] {
  const { t } = useTranslation("management");
  return useMemo(
    () => [
      {
        accessorKey: "text",
        header: t("prompt"),
        cell: ({ row }) => (
          <p className="max-w-2xl whitespace-normal text-sm">{row.original.text}</p>
        ),
      },
      {
        id: "confirmed",
        accessorKey: "confirmed",
        header: t("sourceStatus"),
        cell: ({ row }) => (
          <span className="text-muted-foreground">
            {row.original.source === "manual" ? t("manual") : t("drafted")}
            {t("statusSeparator")}
            {row.original.confirmed ? t("confirmed") : t("unconfirmed")}
          </span>
        ),
      },
      {
        id: "actions",
        header: t("actions"),
        cell: ({ row }) => (
          <div className="flex justify-end gap-1">
            {!row.original.confirmed ? (
              <Button
                type="button"
                variant="outline"
                size="xs"
                disabled={pending}
                onClick={() => onConfirm(row.original.id)}
              >
                {t("confirm")}
              </Button>
            ) : null}
            <PromptDialog
              prompt={row.original}
              label={t("edit")}
              variant="ghost"
              pending={pending}
              onSave={(request) => onUpdate(row.original.id, request)}
            />
            <Button
              type="button"
              variant="ghost"
              size="xs"
              disabled={pending}
              onClick={() => onDelete(row.original)}
            >
              {t("delete")}
            </Button>
          </div>
        ),
      },
    ],
    [onConfirm, onDelete, onUpdate, pending, t],
  );
}
