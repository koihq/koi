import type { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import type { Competitor } from "@/api/competitors.api";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { Button } from "@/components/ui/button";

export function useCompetitorColumns({
  pending,
  onDelete,
  onConfirm,
}: {
  pending: boolean;
  onDelete: (competitor: Competitor) => void;
  onConfirm: (id: string) => void;
}): ColumnDef<Competitor>[] {
  const { t } = useTranslation("management");

  return useMemo(
    () => [
      {
        accessorKey: "name",
        header: t("competitor"),
        cell: ({ row }) => (
          <div className="flex items-center gap-3">
            <BrandIcon name={row.original.name} domain={row.original.domain} />
            <div>
              <p className="text-sm font-medium">{row.original.name}</p>
              <p className="text-muted-foreground">
                {row.original.domain || row.original.githubRepo}
              </p>
            </div>
          </div>
        ),
      },
      {
        id: "confirmed",
        accessorKey: "confirmed",
        header: t("sourceStatus"),
        cell: ({ row }) => (
          <span className="text-muted-foreground">
            {row.original.source === "manual"
              ? t("manual")
              : row.original.source === "drafted"
                ? t("drafted")
                : t("observed")}
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
    [onConfirm, onDelete, pending, t],
  );
}
