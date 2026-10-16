import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/_settings/settings/prompts/")({
  beforeLoad: () => {
    throw redirect({ to: "/", hash: "prompts" });
  },
});
