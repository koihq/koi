import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Account,
  BrandHeader,
  BriefTeaser,
  C,
  CompetitorList,
  Directory,
  EngineList,
  ExploreLink,
  Filters,
  Icon,
  StateGate,
  TrendChart,
} from "../-components/common";
import { ContextNote, FullSection, SectionHeading } from "../-components/screens";
import type { Section } from "../-lib/data";
import { useMock } from "../-lib/state";

const GROUPS: { label: string; sections: Section[] }[] = [
  { label: C.monitor, sections: ["overview", "mentions", "citations"] },
  { label: C.research, sections: ["prompts", "competitors"] },
  { label: C.improve, sections: ["brief"] },
];

function Navigation({
  section,
  onNavigate,
  onSearch,
}: {
  section: Section;
  onNavigate?: () => void;
  onSearch: () => void;
}) {
  return (
    <>
      <button className="cloudflare-search" type="button" onClick={onSearch}>
        <Icon name="search" size={16} />
        <span>{C.searchSections}</span>
        <span>{"⌘ K"}</span>
      </button>
      <nav aria-label={C.navigation}>
        {GROUPS.map((group) => (
          <div key={group.label} className="cloudflare-nav-group">
            <span>{group.label}</span>
            {group.sections.map((item) => (
              <ExploreLink
                key={item}
                section={item}
                className={section === item ? "is-active" : undefined}
                onClick={onNavigate}
              >
                <Icon name={item} size={18} />
                <span>{C[item]}</span>
              </ExploreLink>
            ))}
          </div>
        ))}
      </nav>
      <div className="cloudflare-nav-bottom">
        <span className="exp-signal-dot" />
        <span>{C.sample}</span>
      </div>
    </>
  );
}

export default function Cloudflare({ section }: { section: Section }) {
  const [mobile, setMobile] = useState(false);
  const [directory, setDirectory] = useState(false);
  return (
    <div className="exp-cloudflare">
      <aside className="cloudflare-sidebar">
        <BrandHeader />
        <Navigation section={section} onSearch={() => setDirectory(true)} />
      </aside>
      <div className="cloudflare-body">
        <header className="cloudflare-header">
          <div>
            <Button
              className="cloudflare-mobile-toggle"
              variant="ghost"
              size="icon-lg"
              aria-label={C.navigation}
              onClick={() => setMobile(true)}
            >
              <Icon name="menu" />
            </Button>
            <span>{C.workspace}</span>
            <span className="exp-muted">{C[section]}</span>
          </div>
          <div>
            <span className="cloudflare-sample">{C.sample}</span>
            <Account />
          </div>
        </header>
        <main id="exploration-main" className="cloudflare-main">
          <div className="cloudflare-page-top">
            <SectionHeading section={section} />
            <Filters scan />
          </div>
          <StateGate>
            {section === "overview" ? (
              <CloudflareOverview />
            ) : (
              <div className={`cloudflare-detail ${section === "mentions" ? "is-audit" : ""}`}>
                <FullSection section={section} />
                {section !== "mentions" ? <ContextNote section={section} /> : null}
              </div>
            )}
          </StateGate>
        </main>
      </div>
      <Sheet open={mobile} onOpenChange={setMobile}>
        <SheetContent side="left" className="cloudflare-mobile-nav">
          <SheetHeader>
            <SheetTitle>{C.workspace}</SheetTitle>
            <SheetDescription>{C.navigation}</SheetDescription>
          </SheetHeader>
          <Navigation
            section={section}
            onNavigate={() => setMobile(false)}
            onSearch={() => {
              setMobile(false);
              setDirectory(true);
            }}
          />
        </SheetContent>
      </Sheet>
      <Directory open={directory} onClose={() => setDirectory(false)} />
    </div>
  );
}

function CloudflareOverview() {
  const { summary, previous, prompts } = useMock();
  return (
    <>
      <div className="cloudflare-status-row">
        <section>
          <header>
            <Icon name="mentions" />
            {C.collection}
          </header>
          <div className="cloudflare-status-values">
            <div>
              <span>{C.answers}</span>
              <strong>{summary.completed.toLocaleString("en-US")}</strong>
              <small>{C.collectionReady}</small>
            </div>
            <div>
              <span>{C.tracking}</span>
              <strong>{prompts.filter((prompt) => prompt.confirmed).length}</strong>
              <small>{`${summary.failures} ${C.failures.toLowerCase()}`}</small>
            </div>
          </div>
        </section>
        <section>
          <header>
            <Icon name="overview" />
            {C.enginePerformance}
          </header>
          <div className="cloudflare-status-values">
            <div>
              <span>{C.presence}</span>
              <strong>{`${summary.presence.toFixed(1)}%`}</strong>
              <small className="exp-positive">{`+${(summary.presence - previous.presence).toFixed(1)} pp`}</small>
            </div>
            <div>
              <span>{C.citationRate}</span>
              <strong>{`${summary.cited.toFixed(1)}%`}</strong>
            </div>
          </div>
        </section>
        <section className="cloudflare-trend-summary">
          <header>
            <Icon name="overview" />
            {C.trend}
          </header>
          <TrendChart compare={false} />
        </section>
      </div>
      <div className="cloudflare-overview-grid">
        <div className="cloudflare-overview-left">
          <section className="cloudflare-panel">
            <header>
              <h2>{C.enginesTitle}</h2>
              <ExploreLink section="mentions" className="exp-text-link">
                {C.viewAll}
                <Icon name="next" size={14} />
              </ExploreLink>
            </header>
            <div className="cloudflare-panel-content">
              <EngineList />
            </div>
          </section>
          <section className="cloudflare-panel">
            <header>
              <h2>{C.quickStart}</h2>
            </header>
            <div className="cloudflare-guide">
              <p>{C.guideLead}</p>
              {[
                { title: C.stepOne, lead: C.stepOneLead, section: "mentions" as const },
                { title: C.stepTwo, lead: C.stepTwoLead, section: "competitors" as const },
                { title: C.stepThree, lead: C.stepThreeLead, section: "brief" as const },
              ].map((step, index) => (
                <ExploreLink
                  key={step.title}
                  section={step.section}
                  className="cloudflare-guide-step"
                >
                  <span>{index + 1}</span>
                  <div>
                    <h3>{step.title}</h3>
                    <p>{step.lead}</p>
                  </div>
                  <Icon name="next" size={18} />
                </ExploreLink>
              ))}
            </div>
          </section>
        </div>
        <div className="cloudflare-overview-right">
          <section className="cloudflare-panel">
            <header>
              <h2>{C.substitutes}</h2>
            </header>
            <div className="cloudflare-panel-content">
              <CompetitorList />
            </div>
          </section>
          <section className="cloudflare-action">
            <BriefTeaser />
          </section>
        </div>
      </div>
    </>
  );
}
