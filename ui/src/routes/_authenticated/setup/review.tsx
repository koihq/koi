import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { KoiMark } from "@/components/blocks/koi-mark";
import { SetupReviewEditors } from "@/routes/_authenticated/setup/-components/setup-review-editors";

export const Route = createFileRoute("/_authenticated/setup/review")({
  beforeLoad: ({ context }) => {
    if (!context.brand) throw redirect({ to: "/setup" });
    return { brand: context.brand };
  },
  component: SetupReview,
});

function SetupReview() {
  const { t } = useTranslation("brand");
  const { brand } = Route.useRouteContext();
  const navigate = useNavigate();

  return (
    <main className="min-h-svh px-6 py-12 md:py-16">
      <div className="mx-auto max-w-4xl">
        <KoiMark className="mb-8" />
        <h1 className="text-2xl font-medium tracking-tight">{t("setup.reviewTitle")}</h1>
        <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
          {t("setup.reviewDescription")}
        </p>
        <SetupReviewEditors brandID={brand.id} onFinish={() => navigate({ to: "/mentions" })} />
      </div>
    </main>
  );
}
