import { useState } from "react";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ENGINES } from "../-lib/data";
import {
  HOME_BRANDS,
  HOME_LOGOS,
  type HomeAnswer,
  type HomeBrand,
  homeEvidence,
  mentionRate,
} from "../-lib/home-evidence";
import C from "../-locales/home-signal.en.json";
import { Icon } from "./common";
import { HomeBrandText } from "./home-brand-text";

export function HomeSignal({
  brand,
  topic,
  engines,
  days,
  goal,
  onDaysChange,
  searchOpen,
  onSearchOpenChange,
}: {
  brand: HomeBrand;
  topic: string;
  engines: string[];
  days: string;
  goal: string;
  onDaysChange: (days: string) => void;
  searchOpen: boolean;
  onSearchOpenChange: (open: boolean) => void;
}) {
  const [selected, setSelected] = useState<HomeAnswer | null>(null);
  const [search, setSearch] = useState("");
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
  const recentMisses = [...missing].reverse().slice(0, 3);
  const engineName = (id: string) => ENGINES.find((engine) => engine.id === id)?.name ?? id;
  const searchResults = [...homeEvidence(brand)]
    .reverse()
    .filter(
      (answer, index, answers) =>
        answers.findIndex(
          (item) => item.prompt === answer.prompt && item.engine === answer.engine,
        ) === index,
    )
    .filter((answer) =>
      `${answer.prompt} ${engineName(answer.engine)}`
        .toLowerCase()
        .includes(search.toLowerCase().trim()),
    )
    .slice(0, 6);

  function downloadBrief() {
    const text = `# ${definition.name}: visibility brief\n\nDesign study — synthetic evidence, not live research.\n\nGoal: ${goal}\nScope: ${topic}, ${days} days; ${engines.map(engineName).join(", ")}\n\nMentioned in ${rate.toFixed(1)}% of ${current.length} sample answers.\n\n## Comparison\n${ranking.map((item) => `- ${item.name}: ${item.rate.toFixed(1)}%`).join("\n")}\n\n## Questions to investigate\n${recentMisses.map((answer) => `- ${answer.prompt} (${engineName(answer.engine)}, ${answer.date})`).join("\n")}\n\nCheck real answers and sources before proposing changes.\n`;
    const url = URL.createObjectURL(new Blob([text], { type: "text/markdown" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = `${brand}-demo-visibility-brief.md`;
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

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
        <section className="hc-answer-feed" aria-labelledby="hc-answers-title">
          <h2 id="hc-answers-title" className="sr-only">
            {C.latest}
          </h2>
          {latest.map((answer) => (
            <button
              type="button"
              className="hc-answer-row"
              key={answer.id}
              onClick={() => setSelected(answer)}
            >
              <span className="hc-answer-meta">
                <BrandPill
                  name={engineName(answer.engine)}
                  domain={ENGINES.find((engine) => engine.id === answer.engine)?.domain}
                />
                <small>{answer.date}</small>
              </span>
              <span className="hc-answer-text">
                <HomeBrandText text={answer.prompt} />
              </span>
              <span className="hc-answer-outcome">
                <span
                  className={
                    answer.mentions.includes(definition.name) ? "hc-mentioned" : "hc-absent"
                  }
                >
                  {answer.mentions.includes(definition.name) ? C.mentioned : C.absent}
                </span>
                <span aria-hidden="true">{C.separator}</span>
                {answer.mentions.length ? (
                  <span className="hc-mentioned-brands">
                    {answer.mentions.map((name) => (
                      <BrandPill key={name} name={name} {...HOME_LOGOS[name]} iconOnly />
                    ))}
                  </span>
                ) : (
                  C.noBrands
                )}
              </span>
            </button>
          ))}
        </section>
        <section className="hc-agent-brief" aria-labelledby="hc-agent-title">
          <div className="hc-section-heading">
            <h2 id="hc-agent-title">
              <Icon name="brief" size={13} />
              {C.agent}
            </h2>
          </div>
          <p>{C.agentLead}</p>
          <Button variant="ghost" onClick={downloadBrief}>
            {C.download}
          </Button>
        </section>
      </aside>
      <Dialog open={searchOpen} onOpenChange={onSearchOpenChange}>
        <DialogContent className="hc-command-search">
          <DialogHeader>
            <DialogTitle>{C.searchTitle}</DialogTitle>
            <DialogDescription>{C.searchDescription}</DialogDescription>
          </DialogHeader>
          <Input
            aria-label={C.searchTitle}
            placeholder={C.searchPlaceholder}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
          <div className="hc-search-results">
            {searchResults.length ? (
              searchResults.map((answer) => (
                <button
                  key={answer.id}
                  type="button"
                  onClick={() => {
                    onSearchOpenChange(false);
                    setSelected(answer);
                  }}
                >
                  <span className="hc-search-result-text">
                    <HomeBrandText text={answer.prompt} />
                    <span className="hc-search-result-brand">
                      <BrandPill
                        name={engineName(answer.engine)}
                        domain={ENGINES.find((engine) => engine.id === answer.engine)?.domain}
                      />
                    </span>
                  </span>
                  <Icon name="next" size={14} />
                </button>
              ))
            ) : (
              <p>{C.noResults}</p>
            )}
          </div>
        </DialogContent>
      </Dialog>
      <Dialog
        open={selected !== null}
        onOpenChange={(open) => {
          if (!open) setSelected(null);
        }}
      >
        <DialogContent className="hc-answer-panel">
          <DialogHeader>
            <DialogTitle>{selected && <HomeBrandText text={selected.prompt} />}</DialogTitle>
            <DialogDescription>
              {selected && (
                <span className="hc-answer-meta">
                  <BrandPill
                    name={engineName(selected.engine)}
                    domain={ENGINES.find((engine) => engine.id === selected.engine)?.domain}
                  />
                  <span>{`${selected.date} · Synthetic answer`}</span>
                </span>
              )}
            </DialogDescription>
          </DialogHeader>
          <p className="hc-answer-detail">
            <HomeBrandText
              text={
                selected?.mentions.length
                  ? `This simulated answer recommends investigating ${selected.mentions.join(" and ")}.`
                  : "This simulated answer does not mention any tracked brand."
              }
            />
          </p>
          <p className="hc-caption">{C.detailNote}</p>
        </DialogContent>
      </Dialog>
    </div>
  );
}
