import { createFileRoute, Outlet } from "@tanstack/react-router";

import { RouteLoadError } from "@/components/blocks/route-load-error";
import { loadAccountBrand } from "@/routes/-route-loaders";

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: async ({ context, preload }) =>
    loadAccountBrand(context.api, context.queryClient, preload),
  errorComponent: RouteLoadError,
  component: Outlet,
});
