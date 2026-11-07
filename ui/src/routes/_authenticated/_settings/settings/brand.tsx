import { Add01Icon, PencilEdit02Icon, Tick02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, redirect, useRouter } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Brand, CreateBrand } from "@/api/brands.api";
import { BrandForm } from "@/components/blocks/brand-form";
import { BrandLabel } from "@/components/blocks/brand-label";
import { DataTable, type DataTableColumn } from "@/components/blocks/data-table";
import { PageNav } from "@/components/blocks/page-nav";
import { RowAction, RowActions } from "@/components/blocks/row-actions";
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
  const { brand, capabilities, me } = Route.useRouteContext();
  const { t } = useTranslation("brand");
  const api = useApi();
  const billing = useQuery({
    ...billingOptions(api, me.organization.id),
    enabled: capabilities.cloudBilling,
  });
  const router = useRouter();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [dialog, setDialog] = useState<BrandDialog | null>(null);
  const brands = useQuery(brandListOptions(api, brand.organizationId, { page, pageSize: 10 }));
  const items = brands.data?.items ?? [];
  const activation = useMutation({
    mutationFn: (id: string) => api.brands.activate(id),
    onSuccess: async () => {
      await router.invalidate();
      await router.navigate({ to: "/" });
    },
  });
  const headClass = "h-auto px-3 py-1.5 text-xs text-muted-foreground";
  const cellClass = "px-3 py-3.5 text-body";
  const columns: DataTableColumn<Brand>[] = [
    {
      id: "name",
      header: t("fields.name.label"),
      headClassName: `${headClass} w-[40%]`,
      cellClassName: `${cellClass} whitespace-normal [overflow-wrap:anywhere]`,
      cell: (item) => (
        <span className="flex flex-wrap items-center gap-2">
          <BrandLabel name={item.name} domain={item.domain} lookupLogo={false} />
          {item.id === brand.id && (
            <span className="rounded-chip bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
              {t("settings.active")}
            </span>
          )}
        </span>
      ),
    },
    {
      id: "domain",
      header: t("fields.domain.label"),
      headClassName: `${headClass} w-[32%]`,
      cellClassName: `${cellClass} text-muted-foreground`,
      cell: (item) => (
        <span className="block truncate" title={item.domain || item.githubRepo}>
          {item.domain || item.githubRepo}
        </span>
      ),
    },
    {
      id: "country",
      header: t("fields.country.label"),
      headClassName: headClass,
      cellClassName: `${cellClass} text-muted-foreground`,
      cell: (item) => item.country,
    },
    {
      id: "actions",
      header: <span className="sr-only">{t("settings.actions")}</span>,
      headClassName: `${headClass} w-28 text-end`,
      cellClassName: `${cellClass} text-end`,
      cell: renderActions,
    },
  ];

  function renderActions(item: Brand) {
    return (
      <RowActions>
        {item.id !== brand.id && (
          <RowAction
            label={t("settings.switch")}
            icon={Tick02Icon}
            disabled={activation.isPending}
            onClick={() => activation.mutate(item.id)}
          />
        )}
        <RowAction
          label={t("settings.edit")}
          icon={PencilEdit02Icon}
          onClick={() => setDialog({ kind: "edit", brand: item })}
        />
      </RowActions>
    );
  }

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
    <section
      className="min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9"
      aria-labelledby="brands-heading"
    >
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <h1 id="brands-heading" className="text-section font-semibold">
          {t("settings.title")}
        </h1>
        {/* The free check covers one brand; the trial lifts the limit. */}
        {!billing.data?.firstLook && (
          <Button
            variant="icon-outline"
            size="circle-sm"
            aria-label={t("settings.create")}
            title={t("settings.create")}
            className="text-muted-foreground"
            onClick={() => setDialog({ kind: "create" })}
          >
            <HugeiconsIcon icon={Add01Icon} aria-hidden="true" />
          </Button>
        )}
      </div>
      {activation.isError && (
        <p role="alert" className="mb-4 text-sm text-destructive">
          {t("settings.switchError")}
        </p>
      )}

      <div>
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
            <div className="@container">
              <div className="-mx-3 hidden @2xl:block">
                <DataTable
                  interactive
                  ariaLabel={t("settings.title")}
                  className="table-fixed"
                  data={items}
                  columns={columns}
                  getRowId={(item) => item.id}
                />
              </div>
              <ul className="list-rows -mx-2 grid gap-0.5 @2xl:hidden">
                {items.map((item) => (
                  <li
                    key={item.id}
                    className="group flex min-w-0 items-center gap-3 rounded-control px-2 py-3 hover:bg-signal-pill"
                  >
                    <div className="grid min-w-0 flex-1 gap-1">
                      <span className="flex flex-wrap items-center gap-2">
                        <BrandLabel name={item.name} domain={item.domain} lookupLogo={false} />
                        {item.id === brand.id && (
                          <span className="rounded-chip bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                            {t("settings.active")}
                          </span>
                        )}
                      </span>
                      <span className="truncate text-xs text-muted-foreground">
                        {[item.domain || item.githubRepo, item.country].filter(Boolean).join(" · ")}
                      </span>
                    </div>
                    {renderActions(item)}
                  </li>
                ))}
              </ul>
            </div>
            <PageNav
              page={page}
              pageSize={10}
              total={brands.data.totalItems}
              hasPrev={brands.data.hasPrev}
              hasNext={brands.data.hasNext}
              onPage={setPage}
            />
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
    </section>
  );
}
