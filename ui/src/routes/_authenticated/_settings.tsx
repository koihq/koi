import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { AppHeader } from "@/components/blocks/app-header";
import { RouteLoadError } from "@/components/blocks/route-load-error";

export const Route = createFileRoute("/_authenticated/_settings")({
  beforeLoad: ({ context }) => {
    if (!context.brand) throw redirect({ to: "/setup" });

    return { brand: context.brand };
  },
  errorComponent: RouteLoadError,
  component: SettingsLayout,
});

function SettingsLayout() {
  const { t } = useTranslation("home");
  const { brand } = Route.useRouteContext();

  return (
    <div className="font-sans text-sm font-medium leading-normal min-h-svh bg-background text-foreground">
      <a
        href="#settings-content"
        className="sr-only focus:not-sr-only focus:absolute focus:start-4 focus:top-4 focus:z-50 focus:rounded-sm focus:bg-popover focus:px-3 focus:py-2 focus:text-foreground"
      >
        {t("skip")}
      </a>
      <AppHeader brand={brand} />
      <main
        id="settings-content"
        tabIndex={-1}
        className="mx-auto w-full max-w-[1320px] px-4 pb-10 sm:px-7"
      >
        <Outlet key={brand.id} />
      </main>
    </div>
  );
}
