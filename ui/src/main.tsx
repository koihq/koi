import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouter, RouterProvider } from "@tanstack/react-router";
import { createRoot } from "react-dom/client";

import { ApiProvider, createApi } from "@/api/apiProvider";
import { initializeTheme } from "@/hooks/use-theme";
import "@/config/i18n";
import { routeTree } from "./routeTree.gen";

initializeTheme();

const api = createApi();
const queryClient = new QueryClient();

const router = createRouter({
  routeTree,
  context: { api },
  defaultPreload: "intent",
  scrollRestoration: true,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

const rootElement = document.getElementById("app");

if (rootElement && !rootElement.innerHTML) {
  createRoot(rootElement).render(
    <QueryClientProvider client={queryClient}>
      <ApiProvider api={api}>
        <RouterProvider router={router} />
      </ApiProvider>
    </QueryClientProvider>,
  );
}
