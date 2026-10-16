import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/_app/mentions/")({
  beforeLoad: () => {
    throw redirect({ to: "/", replace: true });
  },
});
