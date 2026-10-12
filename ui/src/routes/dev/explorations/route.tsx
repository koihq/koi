import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/dev/explorations")({
  beforeLoad: () => {
    if (!import.meta.env.DEV) throw redirect({ to: "/" });
  },
  component: Outlet,
});
