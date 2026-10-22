import { createFileRoute, redirect, useRouter } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import type { CreateBrand } from "@/api/brands.api";
import { BrandForm } from "@/components/blocks/brand-form";

export const Route = createFileRoute("/_authenticated/_settings/settings/brand")({
  beforeLoad: async ({ context }) => {
    if ((await context.api.capabilities.get()).cloudBilling) {
      const billing = await context.api.billing.subscription();
      if (billing.promptLimit === 0) throw redirect({ to: "/settings/billing" });
    }
  },
  component: BrandSettings,
});

function BrandSettings() {
  const { brand } = Route.useRouteContext();
  const { t } = useTranslation("brand");
  const api = useApi();
  const router = useRouter();
  const [saved, setSaved] = useState(false);

  async function updateBrand(body: CreateBrand) {
    setSaved(false);
    await api.brands.update(brand.id, body);
    await router.invalidate();
    setSaved(true);
  }

  return (
    <div className="max-w-2xl pb-24">
      <h1 className="text-[30px] font-semibold tracking-[-0.035em]">{t("settings.title")}</h1>
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
