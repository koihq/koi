import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import { NuqsAdapter } from "nuqs/adapters/tanstack-router";
import type { Api } from "@/api/apiProvider";
import "../styles.css";

export const Route = createRootRouteWithContext<{ api: Api; queryClient: QueryClient }>()({
  component: RootComponent,
});

function RootComponent() {
  return (
    <NuqsAdapter>
      <Outlet />
    </NuqsAdapter>
  );
}
