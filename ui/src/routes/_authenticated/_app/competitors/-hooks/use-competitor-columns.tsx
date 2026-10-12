import type { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import type { Competitor, UpdateCompetitor } from "@/api/competitors.api";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { Button } from "@/components/ui/button";
import { CompetitorDialog } from "@/routes/_authenticated/_app/competitors/-components/competitor-dialog";

export function useCompetitorColumns({
  brandID,
  pending,
  onUpdate,
  onDelete,
  onConfirm,
}: {
  brandID: string;
  pending: boolean;
  onUpdate: (id: string, request: UpdateCompetitor) => Promise<void>;
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
                {row.original.domain || row.original.githubRepo || t("competitorIdentityNeeded")}
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
          <div className="flex flex-wrap justify-end gap-1">
            {!row.original.confirmed && (row.original.domain || row.original.githubRepo) ? (
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
            <CompetitorDialog
              brandID={brandID}
              competitor={row.original}
              label={t(!row.original.domain && !row.original.githubRepo ? "addIdentity" : "edit")}
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
              {t(
                row.original.source === "observed" && !row.original.confirmed
                  ? "dismiss"
                  : "delete",
              )}
            </Button>
          </div>
        ),
      },
    ],
    [brandID, onConfirm, onDelete, onUpdate, pending, t],
  );
}
