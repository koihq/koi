import { createFileRoute, Link } from "@tanstack/react-router";
import { useLayoutEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { TooltipProvider } from "@/components/ui/tooltip";
import {
  BrandHeader,
  C,
  Choice,
  EvidenceDialog,
  Filters,
  Notice,
  StateGate,
} from "../-components/common";
import { TradeRepublicPanel } from "../-directions/-panel";
import { PANELS, type Panel } from "../-directions/-workspace";
import type { DemoState } from "../-lib/data";
import { MockProvider } from "../-lib/state";
import "../-styles.css";

export const Route = createFileRoute("/dev/explorations/trade-republic/panels")({
  validateSearch: (search: Record<string, unknown>): { state: DemoState } => ({
    state: search.state === "loading" || search.state === "empty" ? search.state : "populated",
  }),
  component: PanelCatalog,
});

const PANEL_IDS = Object.keys(PANELS) as Panel[];
const WIDE_PANELS = ["trend", "mentions", "prompts", "competitors"];

function PanelCatalog() {
  const { state } = Route.useSearch();

  useLayoutEffect(() => {
    const root = document.documentElement;
    const previous = root.getAttribute("data-exploration-theme");
    const dark = root.classList.contains("dark");
    const scheme = root.style.colorScheme;
    const title = document.title;
    root.setAttribute("data-exploration-theme", "trade-republic");
    root.classList.add("dark");
    root.style.colorScheme = "dark";
    document.title = `${C.panelCatalog} · Koi`;

    return () => {
      if (previous) root.setAttribute("data-exploration-theme", previous);
      else root.removeAttribute("data-exploration-theme");
      root.classList.toggle("dark", dark);
      root.style.colorScheme = scheme;
      document.title = title;
    };
  }, []);

  return (
    <MockProvider direction="trade-republic" initialState={state}>
      <TooltipProvider delay={250}>
        <CatalogContent />
        <EvidenceDialog />
        <Notice />
      </TooltipProvider>
    </MockProvider>
  );
}

function CatalogContent() {
  const { state } = Route.useSearch();
  const navigate = Route.useNavigate();
  const [selectedPrompt, setSelectedPrompt] = useState<string | null>(null);

  return (
    <div className="exploration exp-tr tr-catalog">
      <a href="#panel-catalog-main" className="exp-skip">
        {C.skip}
      </a>
      <header className="tr-catalog-header">
        <BrandHeader />
        <Button
          variant="outline"
          render={
            <Link
              to="/dev/explorations/$direction/$section"
              params={{ direction: "trade-republic", section: "overview" }}
              search={{ state: "populated" }}
            />
          }
        >
          {C.backToWorkspace}
        </Button>
      </header>
      <main id="panel-catalog-main">
        <div className="tr-catalog-intro">
          <h1>{C.panelCatalog}</h1>
          <p>{C.panelCatalogLead}</p>
        </div>
        <div className="tr-catalog-tools">
          <Filters scan />
          <Choice
            label={C.state}
            value={state}
            options={(["populated", "loading", "empty"] as const).map((value) => ({
              value,
              label: C[value],
            }))}
            onChange={(value) => void navigate({ search: { state: value as DemoState } })}
          />
        </div>
        <nav className="tr-catalog-index" aria-label={C.jumpToPanel}>
          {PANEL_IDS.map((panel, index) => (
            <a key={panel} href={`#panel-${panel}`}>
              <span>{String(index + 1).padStart(2, "0")}</span>
              {PANELS[panel]}
            </a>
          ))}
        </nav>
        <StateGate>
          <div className="tr-catalog-grid">
            {PANEL_IDS.map((panel, index) => (
              <section
                key={panel}
                id={`panel-${panel}`}
                className={`tr-panel tr-catalog-panel ${WIDE_PANELS.includes(panel) ? "is-wide" : ""}`}
                data-panel={panel}
                aria-labelledby={`panel-title-${panel}`}
              >
                <header className="tr-catalog-panel-heading">
                  <span>{String(index + 1).padStart(2, "0")}</span>
                  <div>
                    <h2 id={`panel-title-${panel}`}>{PANELS[panel]}</h2>
                    <p>{C.panelDescriptions[panel]}</p>
                  </div>
                  <a href={`#panel-${panel}`} aria-label={`${C.panelLink}: ${PANELS[panel]}`}>
                    {C.panelLink}
                  </a>
                </header>
                <div className="tr-panel-body">
                  <TradeRepublicPanel
                    panel={panel}
                    selectedPrompt={selectedPrompt}
                    onSelectPrompt={setSelectedPrompt}
                  />
                </div>
              </section>
            ))}
          </div>
        </StateGate>
      </main>
    </div>
  );
}
