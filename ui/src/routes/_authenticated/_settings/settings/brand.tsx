import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import type { CreateBrand } from "@/api/brands.api";
import { BrandForm } from "@/components/blocks/brand-form";

export const Route = createFileRoute("/_authenticated/_settings/settings/brand")({
  component: BrandSettings,
});

function BrandSettings() {
  const { brand } = Route.useRouteContext();
  const { t } = useTranslation("brand");
  const api = useApi();
  const [saved, setSaved] = useState(false);

  async function updateBrand(body: CreateBrand) {
    setSaved(false);
    await api.brands.update(brand.id, body);
    setSaved(true);
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h2 className="text-2xl font-medium tracking-tight">{t("settings.title")}</h2>
      <p className="mt-2 text-sm text-muted-foreground">{t("settings.description")}</p>
      <div className="mt-10">
        <BrandForm brand={brand} submitLabel={t("settings.save")} onSubmit={updateBrand} />
        <output className="mt-4 block min-h-5 text-xs text-muted-foreground">
          {saved ? t("settings.saved") : null}
        </output>
      </div>
    </div>
  );
}
