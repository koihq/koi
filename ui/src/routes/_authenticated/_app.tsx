import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useApi } from "@/api/apiProvider";
import type { Brand } from "@/api/brands.api";
import { AppHeader } from "@/components/blocks/app-header";
import { useCollection } from "@/hooks/use-collection";
import { useTracking } from "@/hooks/use-tracking";
import { billingOptions, ensureInitialGeneration, generationOptions } from "@/lib/queries";

export const Route = createFileRoute("/_authenticated/_app")({
  beforeLoad: async ({ context, preload }) => {
    if (!context.brand) throw redirect({ to: "/setup" });
    let draftStartFailed = false;
    if (context.capabilities.cloudBilling) {
      const billing = await context.queryClient.fetchQuery(
        billingOptions(context.api, context.me.organization.id),
      );
      if (billing.promptLimit === 0) throw redirect({ to: "/settings/billing" });
      // A First Look brand drafts from setup or from its agent, which may still be at work.
      // A hover preload must not start a draft.
      if (context.capabilities.aiDrafting && !billing.firstLook && !preload) {
        try {
          await ensureInitialGeneration(context.api, context.queryClient, context.brand.id);
        } catch {
          draftStartFailed = true;
        }
      }
    }

    return { brand: context.brand, draftStartFailed };
  },
  component: AppLayout,
});

function CloudHeader({ brand }: { brand: Brand }) {
  const api = useApi();
  const { me } = Route.useRouteContext();
  const billing = useQuery(billingOptions(api, me.organization.id));

  return billing.data?.firstLook ? <FirstLookHeader brand={brand} /> : <AppHeader brand={brand} />;
}

/** The app opens up when the free check ends. An agent still setting up keeps everything. */
function FirstLookHeader({ brand }: { brand: Brand }) {
  const api = useApi();
  const tracking = useTracking(brand.id);
  const { collecting, latest } = useCollection(brand.id);
  const draft = useQuery(generationOptions(api, brand.id)).data?.items[0];

  const unavailable = tracking.data?.blockedReason === "first_look_unavailable";
  const drafting = Boolean(draft?.startTracking) && draft?.state !== "failed";
  const waitingForRun = !latest.data?.items[0] && drafting && !unavailable;

  return <AppHeader brand={brand} focused={collecting || waitingForRun} />;
}

function AppLayout() {
  const { t } = useTranslation("home");
  const { brand, capabilities } = Route.useRouteContext();

  return (
    <div className="font-sans text-sm leading-normal min-h-svh bg-background text-foreground">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:start-4 focus:top-4 focus:z-50 focus:rounded-chip focus:bg-popover focus:px-3 focus:py-2 focus:text-foreground"
      >
        {t("skip")}
      </a>
      {capabilities.cloudBilling ? <CloudHeader brand={brand} /> : <AppHeader brand={brand} />}
      <main
        id="main-content"
        tabIndex={-1}
        className="mx-auto w-full max-w-[1376px] px-4 pb-10 sm:px-7"
      >
        <Outlet key={brand.id} />
      </main>
    </div>
  );
}
