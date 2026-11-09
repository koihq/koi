import { createFileRoute, Outlet } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { AnalyticsProvider, useAnalytics } from "@/analytics/analyticsProvider";
import { ConsentBanner } from "@/components/blocks/consent-banner";
import { RouteLoadError } from "@/components/blocks/route-load-error";
import { loadAccountBrand } from "@/routes/-route-loaders";

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: async ({ context, preload }) =>
    loadAccountBrand(context.api, context.queryClient, preload),
  errorComponent: RouteLoadError,
  component: AuthenticatedLayout,
});

function AuthenticatedLayout() {
  const { me, capabilities } = Route.useRouteContext();

  return (
    <AnalyticsProvider config={capabilities.analytics} user={me}>
      <Outlet />
      <AnalyticsConsent />
    </AnalyticsProvider>
  );
}

function AnalyticsConsent() {
  const analytics = useAnalytics();
  const { t } = useTranslation("common");

  if (!analytics.enabled) return null;

  return (
    <ConsentBanner
      message={t("consent.message")}
      accept={t("consent.accept")}
      reject={t("consent.reject")}
      onChoose={analytics.setConsent}
    />
  );
}
