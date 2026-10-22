import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { AppHeader } from "@/components/blocks/app-header";

export const Route = createFileRoute("/_authenticated/_app")({
  beforeLoad: async ({ context }) => {
    if (!context.brand) throw redirect({ to: "/setup" });
    if ((await context.api.capabilities.get()).cloudBilling) {
      const billing = await context.api.billing.subscription();
      if (billing.promptLimit === 0) throw redirect({ to: "/settings/billing" });
    }

    return { brand: context.brand, brands: context.brands };
  },
  component: AppLayout,
});

function AppLayout() {
  const { t } = useTranslation("home");
  const { brand, brands } = Route.useRouteContext();

  return (
    <div className="font-sans text-sm font-medium leading-normal min-h-svh bg-background text-foreground">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:start-4 focus:top-4 focus:z-50 focus:rounded-sm focus:bg-popover focus:px-3 focus:py-2 focus:text-foreground"
      >
        {t("skip")}
      </a>
      <AppHeader brand={brand} brands={brands} />
      <main
        id="main-content"
        tabIndex={-1}
        className="mx-auto w-full max-w-[1376px] px-4 pb-10 sm:px-7"
      >
        <Outlet />
      </main>
    </div>
  );
}
