import type { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { Competitor } from "@/api/competitors.api";

import type { CreatePrompt, Prompt } from "@/api/prompts.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { PromptPreviewLink } from "@/components/blocks/prompt-preview-link";
import { Button } from "@/components/ui/button";
import { engines, mockChangePoints, mockRate, type VisibilityScope } from "@/lib/mock-visibility";
import { PromptDialog } from "@/routes/_authenticated/_app/prompts/-components/prompt-dialog";

export function usePromptColumns({
  brandID,
  competitors,
  scope,
  pending,
  onUpdate,
  onDelete,
  onConfirm,
}: {
  brandID: string;
  competitors: Competitor[];
  scope: VisibilityScope;
  pending: boolean;
  onUpdate: (id: string, request: CreatePrompt) => Promise<void>;
  onDelete: (prompt: Prompt) => void;
  onConfirm: (id: string) => void;
}): ColumnDef<Prompt>[] {
  const { t } = useTranslation(["management", "home"]);

  return useMemo(
    () => [
      {
        accessorKey: "text",
        header: t("prompt"),
        cell: ({ row }) => (
          <PromptPreviewLink
            prompt={row.original}
            brandID={brandID}
            competitors={competitors}
            scope={scope}
          />
        ),
      },
      {
        id: "visibility",
        header: t("home:dashboard.title"),
        cell: ({ row }) => {
          if (!row.original.confirmed) {
            return (
              <span className="text-xs text-muted-foreground">
                {t("home:dashboard.awaitingReview")}
              </span>
            );
          }

          const rate = mockRate(brandID, [row.original], scope);
          const change = mockChangePoints(brandID, [row.original], scope);

          return (
            <div className="min-w-28 space-y-1">
              <span className="text-sm font-medium tabular-nums">{`${rate?.toFixed(1)}%`}</span>
              <span
                role="img"
                className="flex gap-0.5"
                aria-label={engines
                  .map(
                    (engine) =>
                      `${engine.name} ${mockRate(brandID, [row.original], { ...scope, engines: [engine.id] })?.toFixed(0)}%`,
                  )
                  .join(", ")}
              >
                {engines.map((engine) => (
                  <span
                    key={engine.id}
                    aria-hidden="true"
                    className="h-1.5 w-5 rounded-full bg-foreground/25"
                    style={{
                      opacity:
                        (mockRate(brandID, [row.original], { ...scope, engines: [engine.id] }) ??
                          0) / 100,
                    }}
                  />
                ))}
              </span>
              {change !== null && (
                <span
                  className={`block text-xs tabular-nums ${change >= 0 ? "text-positive" : "text-negative"}`}
                >
                  {t("home:dashboard.change", {
                    points: `${change >= 0 ? "+" : ""}${change.toFixed(1)}`,
                  })}
                </span>
              )}
            </div>
          );
        },
      },
      {
        id: "competitors",
        header: t("home:dashboard.topCompetitors"),
        cell: ({ row }) => {
          if (!row.original.confirmed) return <span className="text-muted-foreground">{"—"}</span>;

          const top = competitors
            .filter((item) => item.confirmed)
            .map((item) => ({ item, rate: mockRate(brandID, [row.original], scope, item) ?? 0 }))
            .sort((left, right) => right.rate - left.rate)
            .slice(0, 3);

          return top.length ? (
            <span className="flex gap-1">
              {top.map(({ item }) => (
                <BrandPill key={item.id} name={item.name} domain={item.domain} iconOnly />
              ))}
            </span>
          ) : (
            <span className="text-muted-foreground">{"—"}</span>
          );
        },
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
    [brandID, competitors, scope, onConfirm, onDelete, onUpdate, pending, t],
  );
}
