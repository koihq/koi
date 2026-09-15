import { createFileRoute, useRouteContext, useRouter } from "@tanstack/react-router";

import { useApi } from "@/api/apiProvider";
import { Button } from "@/components/ui/button";

export const Route = createFileRoute("/_authenticated/")({
  component: Home,
});

function Home() {
  const { me } = useRouteContext({ from: "/_authenticated" });
  const api = useApi();
  const router = useRouter();

  return (
    <main className="min-h-svh px-6 py-10 sm:px-16 sm:py-16">
      <div className="flex items-baseline justify-between gap-4">
        <p className="text-sm tracking-tight">Koi</p>
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
          Sign out
        </Button>
      </div>
      <div className="mt-24 max-w-lg">
        <h1 className="text-2xl font-medium tracking-tight">{me.email}</h1>
        <p className="mt-2 text-muted-foreground">{me.organization.name}</p>
      </div>
    </main>
  );
}
