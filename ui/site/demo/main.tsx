// The real Koi app, unchanged, on recorded data. Same entry as `src/main.tsx`, with the API answered
// by the replay and every route under /demo, so links move inside the frame and never the page.
import "@/config/zod";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouter, RouterProvider } from "@tanstack/react-router";
import { createRoot } from "react-dom/client";
import { ApiProvider, createApi } from "@/api/apiProvider";
import "@/config/i18n";
import "./demo.css";
import { routeTree } from "@/routeTree.gen";
import fixture from "./api.json.gz?url";
import { installReplay } from "./replay";

async function start() {
  await installReplay(fixture);

  const api = createApi();
  const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity } } });
  const router = createRouter({
    routeTree,
    basepath: "/demo",
    context: { api, queryClient },
    defaultPreload: "intent",
  });
  const root = document.getElementById("app");

  if (root) {
    createRoot(root).render(
      <QueryClientProvider client={queryClient}>
        <ApiProvider api={api}>
          <RouterProvider router={router} />
        </ApiProvider>
      </QueryClientProvider>,
    );
  }
}

void start();
