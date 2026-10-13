import { useRef, useState } from "react";
import { BrandIcon } from "@/components/blocks/brand-icon";
import { Button } from "@/components/ui/button";
import {
  Account,
  BrandHeader,
  C,
  Dock,
  EngineList,
  ExploreLink,
  Filters,
  Icon,
  StateGate,
} from "../-components/common";
import { AnswerInspector, FullSection, SectionHeading } from "../-components/screens";
import { metrics, percent, type Section, selectObservations } from "../-lib/data";
import { useMock } from "../-lib/state";
import { VisibilityChart } from "./-visibility";
import "./-focus.css";

export default function Focus({ section }: { section: Section }) {
  return (
    <div className="exp-focus">
      <header className="focus-header">
        <BrandHeader />
        <div>
          <span>{C.lastUpdated}</span>
          <Account />
        </div>
      </header>
      <main id="exploration-main" className="focus-main">
        <div className="focus-toolbar">
          <span>{C.asOf}</span>
          <Filters scan />
        </div>
        <StateGate>
          {section === "overview" ? (
            <FocusOverview />
          ) : (
            <div className="focus-section">
              <SectionHeading section={section} />
              <FullSection section={section} />
            </div>
          )}
        </StateGate>
      </main>
      <Dock section={section} fey />
    </div>
  );
}

function FocusOverview() {
  const { results, observations, filters, prompts, competitors } = useMock();
  const [selectedPrompt, setSelectedPrompt] = useState<string | null>(null);
  const context = useRef<HTMLElement>(null);
  const prompt = prompts.find((item) => item.id === selectedPrompt);
  const selection = prompt?.id ?? null;
  const answers = selection ? results.filter((answer) => answer.promptId === selection) : results;
  const summary = metrics(answers);
  const previousAnswers = selectObservations(observations, filters, true);
  const previous = metrics(
    selection ? previousAnswers.filter((answer) => answer.promptId === selection) : previousAnswers,
  );
  const delta = summary.presence - previous.presence;
  const questions = prompts
    .filter((item) => item.confirmed)
    .map((item) => ({
      ...item,
      summary: metrics(results.filter((answer) => answer.promptId === item.id)),
    }))
    .filter((item) => item.summary.completed > 0)
    .sort((a, b) => a.summary.presence - b.summary.presence);
  const brands = [
    { id: "tally", name: C.tally, domain: "tally.so" },
    ...competitors.filter((brand) => brand.confirmed),
  ]
    .map((brand) => ({ ...brand, rate: metrics(answers, brand.id).presence }))
    .sort((a, b) => b.rate - a.rate);

  function selectQuestion(id: string | null) {
    setSelectedPrompt(id);
    if (id && window.matchMedia("(max-width: 880px)").matches) {
      context.current?.scrollIntoView({ block: "start" });
    }
  }

  return (
    <>
      <div className="focus-intro">
        <h1>{C.focusTitle}</h1>
        <p>{C.focusLead}</p>
      </div>
      <div className="focus-grid">
        <section className="focus-signal" aria-label={C.visibility}>
          <div className="focus-scope">
            <span>{selection ? C.focusPromptScope : C.focusScope}</span>
            {selection ? (
              <Button variant="ghost" size="sm" onClick={() => selectQuestion(null)}>
                {C.focusClear}
              </Button>
            ) : null}
          </div>
          <dl className="focus-metrics">
            <div className="focus-primary-metric">
              <dt>{C.presence}</dt>
              <dd>{summary.completed ? percent(summary.presence) : "—"}</dd>
              <span className={delta >= 0 ? "exp-positive" : "exp-negative"}>
                <span className="sr-only">{`${C.focusChange}: `}</span>
                {`${delta >= 0 ? "+" : ""}${delta.toFixed(1)} pp`} <small>{C.compared}</small>
              </span>
            </div>
            <div>
              <dt>{C.citationRate}</dt>
              <dd>{summary.completed ? percent(summary.cited) : "—"}</dd>
              <small>{C.focusWindow}</small>
            </div>
            <div>
              <dt>{C.answers}</dt>
              <dd>{summary.completed.toLocaleString("en-US")}</dd>
              <small>{`${summary.failures} ${C.failures.toLowerCase()}`}</small>
            </div>
          </dl>
          <VisibilityChart selectedPrompt={selection} onSelectPrompt={selectQuestion} />
        </section>
        <aside className="focus-context" ref={context} aria-label={C.focusContext}>
          <div className="focus-context-label">
            <Icon name={selection ? "quote" : "overview"} size={16} />
            <span>{C.focusContext}</span>
            <span>{C.sample}</span>
          </div>
          <output className="sr-only" aria-live="polite">
            {prompt ? `${C.focusSelected}: ${prompt.text}` : C.focusOverview}
          </output>
          {selection ? (
            <>
              <span className="exp-overline">{C.focusSelected}</span>
              <AnswerInspector key={selection} promptId={selection} />
              <Button variant="ghost" size="sm" onClick={() => selectQuestion(null)}>
                {C.focusClear}
              </Button>
            </>
          ) : (
            <>
              <h2>{C.focusRecap}</h2>
              <p>{C.focusRecapLead}</p>
              <EngineList logos />
              <div className="focus-opportunity">
                <span className="exp-overline">{C.focusGap}</span>
                {questions[0] ? (
                  <button type="button" onClick={() => selectQuestion(questions[0].id)}>
                    <span>{questions[0].text}</span>
                    <span>
                      {percent(questions[0].summary.presence)}
                      <Icon name="next" size={16} />
                    </span>
                  </button>
                ) : (
                  <p>{C.focusNoQuestions}</p>
                )}
              </div>
            </>
          )}
          <div className="focus-next">
            <span className="exp-overline">
              <Icon name="brief" size={15} />
              {C.focusNext}
            </span>
            <h2>{C.focusNextLead}</h2>
            <ExploreLink section="brief" className="focus-brief-link">
              {C.openBrief}
              <Icon name="next" size={16} />
            </ExploreLink>
          </div>
        </aside>
        <div className="focus-detail">
          <section className="focus-landscape">
            <div className="focus-section-title">
              <h2>{C.focusLandscape}</h2>
              <ExploreLink section="competitors" className="exp-small-link">
                {C.focusAllCompetitors}
                <Icon name="next" size={14} />
              </ExploreLink>
            </div>
            <ol className="focus-brands">
              {brands.map((brand, index) => (
                <li key={brand.id} className={brand.id === "tally" ? "is-brand" : undefined}>
                  <span>{index + 1}</span>
                  <BrandIcon name={brand.name} domain={brand.domain.split("/")[0]} />
                  <span>
                    {brand.name}
                    <small>{brand.domain}</small>
                  </span>
                  <div className="exp-bar" aria-hidden="true">
                    <span style={{ width: `${brand.rate}%` }} />
                  </div>
                  <strong>{summary.completed ? percent(brand.rate) : "—"}</strong>
                </li>
              ))}
            </ol>
          </section>
          <section className="focus-questions">
            <div className="focus-section-title">
              <h2>{C.focusQuestions}</h2>
              <ExploreLink section="prompts" className="exp-small-link">
                {C.focusAllPrompts}
                <Icon name="next" size={14} />
              </ExploreLink>
            </div>
            <p>{C.focusQuestionsLead}</p>
            <div className="focus-question-head">
              <span>{C.promptText}</span>
              <span>{C.focusPromptRate}</span>
            </div>
            {questions.slice(0, 5).map((question) => (
              <button
                key={question.id}
                type="button"
                aria-pressed={selection === question.id}
                className="focus-question"
                onClick={() => selectQuestion(question.id)}
              >
                <span>
                  {question.text}
                  <small>{question.category}</small>
                </span>
                <strong>{percent(question.summary.presence)}</strong>
                <Icon name="next" size={15} />
              </button>
            ))}
            {!questions.length ? <p>{C.focusNoQuestions}</p> : null}
          </section>
        </div>
      </div>
      <footer className="focus-footer">
        <span>{C.sampleEvidence}</span>
        <ExploreLink section="mentions" className="exp-small-link">
          {C.focusReadAnswer}
          <Icon name="next" size={14} />
        </ExploreLink>
      </footer>
    </>
  );
}
