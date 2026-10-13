import { createFileRoute, notFound, redirect } from "@tanstack/react-router";
import { lazy, Suspense, useLayoutEffect } from "react";
import { TooltipProvider } from "@/components/ui/tooltip";
import { C, EvidenceDialog, Notice, PreviewControls } from "../-components/common";
import { type DemoState, DIRECTIONS, type Direction, SECTIONS, type Section } from "../-lib/data";
import { MockProvider } from "../-lib/state";
import "../-styles.css";

const Visitors = lazy(() => import("../-directions/visitors"));
const Cloudflare = lazy(() => import("../-directions/cloudflare"));
const Fey = lazy(() => import("../-directions/fey"));
const TradeRepublic = lazy(() => import("../-directions/trade-republic"));

export const Route = createFileRoute("/dev/explorations/$direction/$section")({
  validateSearch: (search: Record<string, unknown>): { state: DemoState; workspace?: string } => ({
    state: search.state === "loading" || search.state === "empty" ? search.state : "populated",
    workspace: typeof search.workspace === "string" ? search.workspace : undefined,
  }),
  beforeLoad: ({ params }) => {
    if (params.direction === "trade-republic" && params.section !== "overview")
      throw redirect({
        to: "/dev/explorations/$direction/$section",
        params: { direction: "trade-republic", section: "overview" },
        search: { state: "populated" },
      });
    if (
      !DIRECTIONS.includes(params.direction as Direction) ||
      !SECTIONS.includes(params.section as Section)
    )
      throw notFound();
  },
  component: Exploration,
});

function Exploration() {
  const params = Route.useParams();
  const { state, workspace } = Route.useSearch();
  const direction = params.direction as Direction;
  const section = params.section as Section;

  useLayoutEffect(() => {
    const root = document.documentElement;
    const previous = root.getAttribute("data-exploration-theme");
    const dark = root.classList.contains("dark");
    const scheme = root.style.colorScheme;
    const title = document.title;
    root.setAttribute("data-exploration-theme", direction);
    root.classList.toggle("dark", direction === "fey" || direction === "trade-republic");
    root.style.colorScheme =
      direction === "fey" || direction === "trade-republic" ? "dark" : "light";
    document.title = `${C[direction]} · Koi design exploration`;

    return () => {
      if (previous) root.setAttribute("data-exploration-theme", previous);
      else root.removeAttribute("data-exploration-theme");
      root.classList.toggle("dark", dark);
      root.style.colorScheme = scheme;
      document.title = title;
    };
  }, [direction]);

  return (
    <MockProvider key={direction} direction={direction} initialState={state}>
      <TooltipProvider delay={250}>
        <div className="exploration">
          <a href="#exploration-main" className="exp-skip">
            {C.skip}
          </a>
          <PreviewControls />
          <Suspense fallback={<output className="exp-initial-loading">{C.loading}</output>}>
            {direction === "visitors" ? (
              <Visitors section={section} />
            ) : direction === "cloudflare" ? (
              <Cloudflare section={section} />
            ) : direction === "fey" ? (
              <Fey section={section} />
            ) : (
              <TradeRepublic section={section} workspaceId={workspace} />
            )}
          </Suspense>
          <EvidenceDialog />
          <Notice />
        </div>
      </TooltipProvider>
    </MockProvider>
  );
}
