import { createRouter as createTanStackRouter } from "@tanstack/react-router";

import { createApi } from "@/api/apiProvider";
import { routeTree } from "./routeTree.gen";

export function getRouter() {
  return createTanStackRouter({
    routeTree,
    context: { api: createApi() },
    scrollRestoration: true,
    defaultPreload: "intent",
    defaultPreloadStaleTime: 0,
  });
}
