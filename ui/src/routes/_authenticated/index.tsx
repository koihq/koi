import { createFileRoute, useRouteContext, useRouter } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import { KoiMark } from "@/components/blocks/koi-mark";
import { Button } from "@/components/ui/button";

export const Route = createFileRoute("/_authenticated/")({
  component: Home,
});

function Home() {
  const { me } = useRouteContext({ from: "/_authenticated" });
  const api = useApi();
  const router = useRouter();
  const { t } = useTranslation("home");

  return (
    <main className="min-h-svh px-6 py-10 sm:px-16 sm:py-16">
      <div className="flex items-center justify-between gap-4">
        <KoiMark />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={async () => {
            await api.auth.logout();
            await router.invalidate();
            await router.navigate({ to: "/sign-in" });
          }}
        >
          {t("signOut")}
        </Button>
      </div>
      <div className="mt-24 max-w-lg">
        <h1 className="text-2xl font-medium tracking-tight">{me.email}</h1>
        <p className="mt-2 text-muted-foreground">{me.organization.name}</p>
      </div>
    </main>
  );
}
