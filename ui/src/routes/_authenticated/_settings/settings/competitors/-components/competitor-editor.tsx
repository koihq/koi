import { useState } from "react";
import { useTranslation } from "react-i18next";

import type { Competitor, CreateCompetitor } from "@/api/competitors.api";
import type { ApiListResponse } from "@/api/shared/types";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { ListPagination } from "@/components/blocks/list-pagination";
import { Button } from "@/components/ui/button";
import { CompetitorForm } from "@/routes/_authenticated/_settings/settings/competitors/-components/competitor-form";

export function CompetitorEditor({
  competitors,
  unconfirmedItems,
  onPageChange,
  onCreate,
  onUpdate,
  onDelete,
  onConfirm,
  onConfirmAll,
  pending,
}: {
  competitors: ApiListResponse<Competitor>;
  unconfirmedItems: number;
  onPageChange: (page: number) => void;
  onCreate: (request: CreateCompetitor) => Promise<unknown>;
  onUpdate: (id: string, request: CreateCompetitor) => Promise<unknown>;
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
    <section aria-labelledby="competitors-heading">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 id="competitors-heading" className="text-lg font-medium tracking-tight">
            {t("lists.competitors.title")}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("lists.competitors.description")}</p>
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
        <CompetitorForm
          pending={pending.create}
          onSave={async (request) => {
            await onCreate(request);
          }}
        />
      </div>
      {competitors.items.length === 0 ? (
        <p className="mt-6 text-sm text-muted-foreground">{t("lists.competitors.empty")}</p>
      ) : (
        <ul className="mt-6 space-y-5">
          {competitors.items.map((competitor) => (
            <li key={competitor.id}>
              {editingID === competitor.id ? (
                <CompetitorForm
                  initial={competitor}
                  pending={pending.update}
                  onCancel={() => setEditingID(undefined)}
                  onSave={async (request) => {
                    await onUpdate(competitor.id, request);
                    setEditingID(undefined);
                  }}
                />
              ) : (
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div className="flex min-w-0 gap-3">
                    <BrandIcon name={competitor.name} domain={competitor.domain} />
                    <div className="min-w-0">
                      <p className="font-medium">{competitor.name}</p>
                      <p className="text-sm text-muted-foreground">{competitor.domain}</p>
                      {competitor.aliases.length > 0 ? (
                        <p className="mt-1 text-xs text-muted-foreground">
                          {t("lists.competitors.aliasesValue", {
                            aliases: competitor.aliases.join(", "),
                          })}
                        </p>
                      ) : null}
                      <p className="mt-1 text-xs text-muted-foreground">
                        {competitor.source === "manual"
                          ? t("lists.source.manual")
                          : competitor.source === "drafted"
                            ? t("lists.source.drafted")
                            : t("lists.source.observed")}
                        {" · "}
                        {competitor.confirmed ? t("lists.confirmed") : t("lists.unconfirmed")}
                      </p>
                    </div>
                  </div>
                  <div className="flex shrink-0 flex-wrap gap-1">
                    {!competitor.confirmed ? (
                      <Button
                        type="button"
                        size="xs"
                        variant="outline"
                        disabled={pending.confirm}
                        onClick={() => onConfirm(competitor.id)}
                      >
                        {t("lists.confirm")}
                      </Button>
                    ) : null}
                    <Button
                      type="button"
                      size="xs"
                      variant="ghost"
                      onClick={() => setEditingID(competitor.id)}
                    >
                      {t("lists.edit")}
                    </Button>
                    <Button
                      type="button"
                      size="xs"
                      variant="ghost"
                      disabled={pending.delete}
                      onClick={() => {
                        if (window.confirm(t("lists.competitors.deleteConfirm")))
                          onDelete(competitor.id);
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
        currentPage={competitors.currentPage}
        totalPages={competitors.totalPages}
        onPageChange={onPageChange}
      />
    </section>
  );
}
