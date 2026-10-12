import {
  Account,
  BrandHeader,
  BriefTeaser,
  C,
  CompetitorList,
  Dock,
  EngineList,
  ExploreLink,
  Filters,
  Icon,
  MetricStrip,
  StateGate,
  TrendChart,
} from "../-components/common";
import { FullSection, SectionHeading } from "../-components/screens";
import { type Section, trend } from "../-lib/data";
import { useMock } from "../-lib/state";

export default function Visitors({ section }: { section: Section }) {
  return (
    <div className="exp-visitors">
      <header className="visitors-header">
        <BrandHeader />
        <Account />
      </header>
      <main
        id="exploration-main"
        className={`visitors-main ${section === "overview" ? "is-overview" : ""}`}
      >
        <div className="visitors-toolbar">
          <Filters />
          <ExploreLink section="brief" className="exp-text-link">
            {C.openBrief}
            <Icon name="next" size={14} />
          </ExploreLink>
        </div>
        <StateGate>
          {section === "overview" ? (
            <VisitorsOverview />
          ) : (
            <div className="visitors-section-page">
              <SectionHeading section={section} />
              <FullSection section={section} />
            </div>
          )}
        </StateGate>
      </main>
      <Dock section={section} />
    </div>
  );
}

function VisitorsOverview() {
  const { summary, filters, observations } = useMock();
  return (
    <>
      <h1 className="sr-only">{C.overview}</h1>
      <div className="visitors-measurements">
        <MetricStrip compact />
        <p className="visitors-verdict">{C.verdictDetail}</p>
      </div>
      <div className="visitors-chart">
        <TrendChart large />
      </div>
      <div className="visitors-bottom">
        <section className="visitors-block">
          <div className="visitors-block-heading">
            <h2>{C.enginesTitle}</h2>
            <ExploreLink section="mentions" className="exp-small-link">
              {C.mentions}
              <Icon name="next" size={14} />
            </ExploreLink>
          </div>
          <EngineList />
        </section>
        <section className="visitors-block">
          <div className="visitors-block-heading">
            <h2>{C.substitutes}</h2>
            <ExploreLink section="competitors" className="exp-small-link">
              {C.viewAll}
              <Icon name="next" size={14} />
            </ExploreLink>
          </div>
          <CompetitorList />
        </section>
        <section className="visitors-block visitors-collection">
          <div className="visitors-block-heading">
            <h2>{C.collection}</h2>
            <span className="exp-small-link">{C.sample}</span>
          </div>
          <strong>{summary.completed.toLocaleString("en-US")}</strong>
          <p>{`${C.answers.toLowerCase()} in the last ${filters.days} days`}</p>
          <div className="visitors-pips" aria-hidden={true}>
            {trend(observations, filters).map((point) => (
              <i key={point.day} style={{ height: `${10 + point.value * 0.5}px` }} />
            ))}
          </div>
          <ExploreLink section="mentions" className="exp-text-link">
            {C.readEvidence}
            <Icon name="next" size={14} />
          </ExploreLink>
        </section>
        <section className="visitors-block visitors-brief">
          <BriefTeaser />
        </section>
      </div>
    </>
  );
}
