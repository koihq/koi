import {
  Account,
  BrandHeader,
  BriefTeaser,
  C,
  CompetitorList,
  Dock,
  ExploreLink,
  Filters,
  Icon,
  MetricStrip,
  StateGate,
  TrendChart,
} from "../-components/common";
import { ContextNote, FullSection, SectionHeading, SignalFeed } from "../-components/screens";
import type { Section } from "../-lib/data";

export default function Fey({ section }: { section: Section }) {
  return (
    <div className="exp-fey">
      <header className="fey-header">
        <BrandHeader greeting />
        <div>
          <span>{C.lastUpdated}</span>
          <Account />
        </div>
      </header>
      <main id="exploration-main" className="fey-main">
        <div className="fey-toolbar">
          <span>{section === "overview" ? C.asOf : C[section]}</span>
          <Filters />
        </div>
        <StateGate>
          {section === "overview" ? (
            <FeyOverview />
          ) : (
            <>
              <SectionHeading section={section} />
              <div className={`fey-section-grid ${section === "mentions" ? "is-audit" : ""}`}>
                <section className="fey-section-content">
                  <FullSection section={section} />
                </section>
                {section !== "mentions" ? <ContextNote section={section} /> : null}
              </div>
            </>
          )}
        </StateGate>
      </main>
      <Dock section={section} fey />
    </div>
  );
}

function FeyOverview() {
  return (
    <div className="fey-overview-grid">
      <section className="fey-landscape">
        <span className="exp-overline">{C.today}</span>
        <h1>{C.verdict}</h1>
        <MetricStrip compact />
        <TrendChart />
        <div className="fey-ranking">
          <div className="fey-section-label">
            <h2>{C.ranking}</h2>
            <ExploreLink section="competitors" className="exp-small-link">
              {C.viewAll}
              <Icon name="next" size={14} />
            </ExploreLink>
          </div>
          <CompetitorList />
        </div>
        <ExploreLink section="mentions" className="exp-small-link fey-audit-link">
          {C.readEvidence}
          <Icon name="next" size={14} />
        </ExploreLink>
      </section>
      <div className="fey-editorial">
        <section className="fey-recap">
          <div className="exp-feed-meta">
            <span>
              <Icon name="overview" size={14} />
              {C.recap}
            </span>
            <span>{C.sample}</span>
          </div>
          <h2>{C.recapLead}</h2>
          <ExploreLink section="mentions" className="exp-text-link">
            {C.readEvidence}
            <Icon name="next" size={14} />
          </ExploreLink>
        </section>
        <section className="fey-agent">
          <BriefTeaser />
        </section>
        <SignalFeed />
      </div>
    </div>
  );
}
