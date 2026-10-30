import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, redirect, useRouter } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Brand, CreateBrand } from "@/api/brands.api";
import { BrandForm } from "@/components/blocks/brand-form";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { DataTable, type DataTableColumn } from "@/components/blocks/data-table";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { billingOptions, brandListOptions } from "@/lib/queries";
import { brandsKey } from "@/lib/query-keys";

export const Route = createFileRoute("/_authenticated/_settings/settings/brand")({
  beforeLoad: async ({ context }) => {
    if (!context.brand) throw redirect({ to: "/setup" });

    if (context.capabilities.cloudBilling) {
      const billing = await context.queryClient.fetchQuery(
        billingOptions(context.api, context.me.organization.id),
      );
      if (billing.promptLimit === 0) throw redirect({ to: "/settings/billing" });
    }

    return { brand: context.brand };
  },
  component: BrandSettings,
});

type BrandDialog = { kind: "create" } | { kind: "edit"; brand: Brand };

function BrandSettings() {
  const { brand } = Route.useRouteContext();
  const { t } = useTranslation("brand");
  const api = useApi();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [dialog, setDialog] = useState<BrandDialog | null>(null);
  const brands = useQuery(brandListOptions(api, brand.organizationId, { page, pageSize: 10 }));
  const items = brands.data?.items ?? [];
  const columns: DataTableColumn<Brand>[] = [
    {
      id: "name",
      header: t("fields.name.label"),
      cell: (item) => (
        <span className="inline-flex items-center gap-2">
          <BrandIcon name={item.name} domain={item.domain} lookupLogo={false} className="size-6" />
          <span>{item.name}</span>
          {item.id === brand.id && (
            <span className="text-muted-foreground">{t("settings.active")}</span>
          )}
        </span>
      ),
    },
    {
      id: "domain",
      header: t("fields.domain.label"),
      cell: (item) => item.domain || item.githubRepo,
    },
    { id: "country", header: t("fields.country.label"), cell: (item) => item.country },
    {
      id: "actions",
      header: t("settings.actions"),
      headClassName: "text-end",
      cellClassName: "text-end",
      cell: (item) => (
        <Button variant="ghost" size="sm" onClick={() => setDialog({ kind: "edit", brand: item })}>
          {t("settings.edit")}
        </Button>
      ),
    },
  ];

  async function saveBrand(body: CreateBrand) {
    if (!dialog) return;

    if (dialog.kind === "create") {
      // /me falls back to the newest brand without a saved preference.
      await api.brands.activate(brand.id);
      await api.brands.create(body);
      setPage(1);
    } else {
      await api.brands.update(dialog.brand.id, body);
    }

    await queryClient.invalidateQueries({ queryKey: brandsKey(brand.organizationId) });
    await router.invalidate();
    setDialog(null);
  }

  return (
    <div className="w-full pb-24">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-[30px] font-semibold tracking-[-0.035em]">{t("settings.title")}</h1>
          <p className="mt-2 text-sm text-muted-foreground">{t("settings.description")}</p>
        </div>
        <Button onClick={() => setDialog({ kind: "create" })}>{t("settings.create")}</Button>
      </div>

      <div className="mt-10">
        {brands.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {t("settings.loadError")}{" "}
            <Button variant="link" onClick={() => void brands.refetch()}>
              {t("retry")}
            </Button>
          </p>
        ) : brands.isPending ? (
          <p className="text-sm text-muted-foreground">{t("settings.loading")}</p>
        ) : (
          <>
            <DataTable
              ariaLabel={t("settings.title")}
              data={items}
              columns={columns}
              getRowId={(item) => item.id}
            />
            {brands.data.totalPages > 1 && (
              <nav
                aria-label={t("settings.pagination")}
                className="mt-5 flex items-center justify-between gap-4 text-xs text-muted-foreground"
              >
                <span>{t("settings.page", { current: page, total: brands.data.totalPages })}</span>
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={!brands.data.hasPrev}
                    onClick={() => setPage(page - 1)}
                  >
                    {t("settings.previous")}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={!brands.data.hasNext}
                    onClick={() => setPage(page + 1)}
                  >
                    {t("settings.next")}
                  </Button>
                </div>
              </nav>
            )}
          </>
        )}
      </div>

      <Dialog
        open={dialog !== null}
        onOpenChange={(open) => {
          if (!open) setDialog(null);
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {t(dialog?.kind === "edit" ? "settings.editTitle" : "settings.createTitle")}
            </DialogTitle>
            <DialogDescription>{t("settings.dialogDescription")}</DialogDescription>
          </DialogHeader>
          {dialog && (
            <BrandForm
              key={dialog.kind === "edit" ? dialog.brand.id : "create"}
              brand={dialog.kind === "edit" ? dialog.brand : undefined}
              submitLabel={t(dialog.kind === "edit" ? "settings.save" : "settings.create")}
              onSubmit={saveBrand}
            />
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
