import { createFileRoute, redirect, useRouter } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Brand, CreateBrand } from "@/api/brands.api";
import { ApiError } from "@/api/errors";
import { BrandForm } from "@/components/blocks/brand-form";
import { KoiMark } from "@/components/blocks/koi-mark";
export const Route = createFileRoute("/_authenticated/setup/")({
  beforeLoad: ({ context }) => {
    if (context.brand) {
      throw redirect({ to: "/prompts" });
    }
  },
  component: Setup,
});

function Setup() {
  const { t } = useTranslation("brand");
  const api = useApi();
  const router = useRouter();

  async function createBrand(body: CreateBrand) {
    let brand: Brand | undefined;
    try {
      brand = await api.brands.create(body);
    } catch (error) {
      if (!(error instanceof ApiError) || error.code !== "brand_limit_reached") {
        throw error;
      }
    }
    let draftStartFailed = false;
    if (brand) {
      try {
        if ((await api.capabilities.get()).aiDrafting) {
          await api.generations.create(brand.id, "initial", `setup-${brand.id}`);
        }
      } catch {
        draftStartFailed = true;
      }
    }
    await router.invalidate();
    await router.navigate({
      to: "/prompts",
      search: draftStartFailed ? { draftStart: "failed" } : {},
    });
  }

  return (
    <main className="flex min-h-svh flex-col items-center justify-center px-6 py-12">
      <div className="w-full max-w-md">
        <KoiMark className="mb-8" />
        <h1 className="text-2xl font-medium tracking-tight">{t("setup.title")}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t("setup.description")}</p>
        <div className="mt-8">
          <BrandForm submitLabel={t("setup.submit")} onSubmit={createBrand} />
        </div>
      </div>
    </main>
  );
}
