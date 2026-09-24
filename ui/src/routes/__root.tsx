import { TanStackDevtools } from "@tanstack/react-devtools";
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { NuqsAdapter } from "nuqs/adapters/tanstack-router";

import type { Api } from "@/api/apiProvider";
import "../styles.css";

export const Route = createRootRouteWithContext<{ api: Api }>()({
  component: RootComponent,
});

function parkDevtoolsTrigger() {
  try {
    const key = "tanstack_devtools_settings";
    const prev = JSON.parse(localStorage.getItem(key) ?? "{}") as {
      triggerMode?: string;
      position?: string;
    };
    if (prev.triggerMode === "fixed" && prev.position === "bottom-right") {
      return;
    }
    localStorage.setItem(
      key,
      JSON.stringify({ ...prev, triggerMode: "fixed", position: "bottom-right" }),
    );
  } catch {
    // quota / private mode
  }
}

function RootComponent() {
  if (import.meta.env.DEV) {
    parkDevtoolsTrigger();
  }

  return (
    <>
      <NuqsAdapter>
        <Outlet />
      </NuqsAdapter>
      {import.meta.env.DEV ? (
        <TanStackDevtools
          config={{
            position: "bottom-right",
            triggerMode: "fixed",
          }}
          plugins={[
            {
              name: "TanStack Router",
              render: <TanStackRouterDevtoolsPanel />,
            },
          ]}
        />
      ) : null}
    </>
  );
}
