import { createRouter, RouterProvider } from "@tanstack/react-router";
import { createRoot } from "react-dom/client";

import { ApiProvider, createApi } from "@/api/apiProvider";
import "@/config/i18n";
import { routeTree } from "./routeTree.gen";

const api = createApi();

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
    <ApiProvider api={api}>
      <RouterProvider router={router} />
    </ApiProvider>,
  );
}
