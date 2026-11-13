import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createBrowserHistory, createRouter, RouterProvider } from "@tanstack/react-router";
import { render } from "vitest-browser-react";

import { ApiProvider, createApi } from "@/api/apiProvider";
import { initializeTheme } from "@/hooks/use-theme";
import { routeTree } from "@/routeTree.gen";

const queryClients = new Set<QueryClient>();

/** Stops polling and cancels the requests of every rendered route, before handlers change. */
export function clearRenderedQueries() {
  for (const client of queryClients) client.clear();
  queryClients.clear();
}

/**
 * Mounts the real app on `path`: route tree, loaders, query client and API client, with a fresh
 * cache and failed queries reported at once instead of retried. `router.state.location` is the URL.
 * Only one route may be rendered at a time, because it owns `window.location`.
 */
export async function renderRoute(
  path: string,
  { theme = "light" }: { theme?: "light" | "dark" } = {},
) {
  try {
    localStorage.setItem("koi:theme:v1", theme);
  } catch {}
  initializeTheme();

  // The app reads search params from `window.location` (nuqs), so the router uses the real
  // history. The setup file restores the test frame's own URL after each test.
  window.history.replaceState(null, "", path);

  const api = createApi();
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  queryClients.add(queryClient);
  const router = createRouter({
    routeTree,
    context: { api, queryClient },
    history: createBrowserHistory(),
  });

  const screen = await render(
    <QueryClientProvider client={queryClient}>
      <ApiProvider api={api}>
        <RouterProvider router={router} />
      </ApiProvider>
    </QueryClientProvider>,
  );

  return { screen, router, queryClient };
}

/** The current router location as `pathname?search#hash`. */
export function currentUrl(router: { state: { location: { href: string } } }) {
  return router.state.location.href;
}
