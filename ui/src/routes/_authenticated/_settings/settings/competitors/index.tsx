import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/_settings/settings/competitors/")({
  beforeLoad: () => {
    throw redirect({ to: "/competitors" });
  },
});
