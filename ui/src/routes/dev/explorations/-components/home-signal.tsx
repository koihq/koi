import { Link } from "@tanstack/react-router";
import { BrandPill } from "@/components/blocks/brand-pill";
import { ENGINES } from "../-lib/data";
import {
  HOME_BRANDS,
  HOME_LOGOS,
  type HomeBrand,
  homeEvidence,
  mentionRate,
} from "../-lib/home-evidence";
import C from "../-locales/home-signal.en.json";
import M from "../-locales/mvp.en.json";
import { AnswerFeed } from "./answer-feed";
import { Icon } from "./common";
import { HomeBrandText } from "./home-brand-text";

export function HomeSignal({
  brand,
  topic,
  engines,
  days,
  onDaysChange,
}: {
  brand: HomeBrand;
  topic: string;
  engines: string[];
  days: string;
  onDaysChange: (days: string) => void;
}) {
  const definition = HOME_BRANDS[brand];
  const period = Number(days);
  const scoped = homeEvidence(brand).filter(
    (answer) => answer.topic === topic && engines.includes(answer.engine),
  );
  const current = scoped.filter((answer) => answer.day >= 90 - period);
  const rate = mentionRate(current, definition.name);
  const missing = current.filter((answer) => !answer.mentions.includes(definition.name));
  const ranking = [definition.name, ...definition.rivals]
    .map((name) => ({ name, rate: mentionRate(current, name) }))
    .sort((a, b) => b.rate - a.rate);
  const leader = ranking[0];
  const bucketCount = period;
  const points = Array.from({ length: bucketCount }, (_, index) => {
    const start = 90 - period + Math.floor((index * period) / bucketCount);
    const end = 90 - period + Math.floor(((index + 1) * period) / bucketCount);
    return mentionRate(
      current.filter((answer) => answer.day >= start && answer.day < end),
      definition.name,
    );
  });
  const path = points
    .map(
      (value, index) =>
        `${index === 0 ? "M" : "L"} ${(index / (points.length - 1)) * 600} ${180 - value * 1.6}`,
    )
    .join(" ");
  const alternativePath = Array.from({ length: bucketCount }, (_, index) => {
    const value = mentionRate(
      current.filter((answer) => answer.day === 90 - period + index),
      definition.rivals[0],
    );
    return `${index === 0 ? "M" : "L"} ${(index / (bucketCount - 1)) * 600} ${180 - value * 1.6}`;
  }).join(" ");
  const latest = [...current].reverse().slice(0, 4);

  return (
    <div className="hc-evidence">
      <div className="hc-evidence-main">
        <section className="hc-visibility" aria-labelledby="hc-visibility-title">
          <div className="hc-section-heading">
            <span>{C.sampleDate}</span>
          </div>
          <h2 id="hc-visibility-title">
            <HomeBrandText text={`${definition.name} in AI answers`} />
          </h2>
          <div className="hc-rate">
            {rate.toFixed(1)}
            <span>{C.percent}</span>
          </div>
          <p>
            {`Mentioned in ${current.filter((answer) => answer.mentions.includes(definition.name)).length} of ${current.length} sample answers`}
          </p>
          <svg
            className="hc-chart"
            viewBox="0 0 600 200"
            role="img"
            aria-label={`${definition.name} mention rate over ${days} days, from ${points[0].toFixed(1)}% to ${points.at(-1)?.toFixed(1)}%. Each point groups answers within the selected period.`}
          >
            {[20, 100, 180].map((y) => (
              <path key={y} d={`M 0 ${y} H 600`} className="hc-chart-grid" />
            ))}
            <path d={path} className="hc-chart-line" />
            <path d={alternativePath} className="hc-chart-alternative" />
          </svg>
          <div className="hc-chart-dates">
            <span>{current[0]?.date}</span>
            <span>{current.at(-1)?.date}</span>
          </div>
          <div className="hc-chart-toolbar">
            <div className="hc-chart-legend">
              {[definition.name, definition.rivals[0]].map((name) => (
                <span key={name}>
                  <BrandPill name={name} {...HOME_LOGOS[name]} />
                </span>
              ))}
            </div>
            <fieldset className="hc-periods" aria-label={C.period}>
              {["7", "30", "90"].map((value) => (
                <button
                  type="button"
                  key={value}
                  aria-pressed={days === value}
                  onClick={() => onDaysChange(value)}
                >{`${value}D`}</button>
              ))}
            </fieldset>
          </div>
        </section>
        <section className="hc-comparison" aria-labelledby="hc-comparison-title">
          <div className="hc-section-heading">
            <h2 id="hc-comparison-title">{C.comparison}</h2>
            <span>{C.rate}</span>
          </div>
          {ranking.map((item, index) => (
            <div className="hc-ranking-row" key={item.name}>
              <span className="hc-rank">{index + 1}</span>
              <span className="hc-ranking-brand">
                <BrandPill name={item.name} {...HOME_LOGOS[item.name]} />
                {item.name === definition.name && <small>{C.yourBrand}</small>}
              </span>
              <span className="hc-ranking-track">
                <span style={{ width: `${item.rate}%` }} />
              </span>
              <strong>{`${item.rate.toFixed(1)}%`}</strong>
            </div>
          ))}
          <p className="hc-caption">{C.multiple}</p>
        </section>
        <section className="hc-engine-comparison" aria-labelledby="hc-engine-title">
          <div className="hc-section-heading">
            <h2 id="hc-engine-title">{C.engineComparison}</h2>
          </div>
          {ENGINES.filter((engine) => engines.includes(engine.id)).map((engine) => {
            const engineRate = mentionRate(
              current.filter((answer) => answer.engine === engine.id),
              definition.name,
            );
            return (
              <div className="hc-engine-row" key={engine.id}>
                <BrandPill name={engine.name} domain={engine.domain} />
                <span className="hc-ranking-track">
                  <span style={{ width: `${engineRate}%` }} />
                </span>
                <strong>{`${engineRate.toFixed(1)}%`}</strong>
              </div>
            );
          })}
          <p className="hc-caption">{C.sampleCoverage}</p>
        </section>
      </div>
      <aside className="hc-feed" aria-label={C.recapLabel}>
        <section className="hc-recap">
          <div className="hc-section-heading">
            <h2>
              <Icon name="quote" size={13} />
              {C.recap}
            </h2>
            <span>{C.collection}</span>
          </div>
          <p>
            <strong>
              <HomeBrandText
                text={
                  leader.name === definition.name ? C.leading : `${leader.name} appears more often.`
                }
              />
            </strong>
            <HomeBrandText
              text={` ${definition.name} is absent from ${missing.length} of ${current.length} answers. Start with the questions where alternatives appear.`}
            />
          </p>
        </section>
        <AnswerFeed answers={latest} brand={brand} />
        <Link
          from="/dev/explorations/home-context"
          to="/dev/explorations/home-context"
          search={(previous) => ({ ...previous, view: "mentions", brand })}
          className="hc-text-link"
        >
          {M.viewMentions}
        </Link>
      </aside>
    </div>
  );
}
