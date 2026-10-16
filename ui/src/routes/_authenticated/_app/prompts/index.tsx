import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/_app/prompts/")({
  beforeLoad: () => {
    throw redirect({ to: "/", hash: "prompts", replace: true });
  },
});
