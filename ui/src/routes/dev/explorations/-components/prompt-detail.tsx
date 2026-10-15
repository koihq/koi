import { type ReactNode, useLayoutEffect, useRef, useState } from "react";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Button } from "@/components/ui/button";
import { ENGINES } from "../-lib/data";
import type { ExplorationSearch } from "../-lib/exploration-fixtures";
import type { HomeBrand } from "../-lib/home-evidence";
import {
  type EvidenceRecord,
  inScope,
  type PromptRow,
  percent,
  rateFor,
} from "../-lib/prompt-visibility";
import M from "../-locales/mvp.en.json";
import C from "../-locales/prompt-visibility.en.json";
import { AnswerContent } from "./answer-panel";
import { HomeBrandText } from "./home-brand-text";
import { PromptEditor } from "./prompt-editor";

export function PromptDetail({
  row,
  records,
  brand,
  brandName,
  search,
  onSearch,
  onSave,
  back,
  children,
}: {
  row: PromptRow;
  records: EvidenceRecord[];
  brand: HomeBrand;
  brandName: string;
  search: ExplorationSearch;
  onSearch: (patch: Partial<ExplorationSearch>) => void;
  onSave: (text: string, id: string) => void;
  back: ReactNode;
  children: ReactNode;
}) {
  const [editing, setEditing] = useState(false);
  const [limit, setLimit] = useState(20);
  const [mobileReader, setMobileReader] = useState(Boolean(search.answerId));
  const editTrigger = useRef<HTMLButtonElement | null>(null);
  const reader = useRef<HTMLElement | null>(null);
  const lastTrigger = useRef<HTMLButtonElement | null>(null);
  const engine = search.responseEngine ?? search.engine ?? "chatgpt";
  const scope = { days: search.days ?? 30, promptIds: [row.prompt.id] };
  const period = inScope(records, scope);
  const answers = period
    .filter((record) => record.result.engine === engine)
    .sort((a, b) => b.answer.day - a.answer.day || b.result.id.localeCompare(a.result.id));
  const selected = search.answerId
    ? answers.find((record) => record.result.id === search.answerId)
    : answers[0];
  const engineInfo = ENGINES.find((item) => item.id === engine);

  useLayoutEffect(() => {
    if (mobileReader && window.matchMedia("(max-width: 959px)").matches)
      reader.current?.focus({ preventScroll: true });
  }, [mobileReader]);

  return (
    <section className="hc-prompt-detail">
      {back}
      <div className="hc-section-heading hc-prompt-title">
        <h1 id="hc-page-heading" tabIndex={-1}>
          <HomeBrandText text={row.prompt.text} />
        </h1>
        {!row.past && (
          <Button ref={editTrigger} variant="ghost" size="sm" onClick={() => setEditing(true)}>
            {M.editPrompt}
          </Button>
        )}
      </div>
      {row.past && <p className="hc-caption">{C.pastPrompt}</p>}
      {children}
      <section className="hc-responses" aria-labelledby="hc-responses-heading">
        <h2 id="hc-responses-heading">{C.responses}</h2>
        <fieldset className="hc-response-engines" aria-label={M.engine}>
          {ENGINES.map((item) => {
            const rate = rateFor(
              period.filter((record) => record.result.engine === item.id),
              brand,
            ).rate;

            return (
              <button
                type="button"
                key={item.id}
                aria-pressed={engine === item.id}
                onClick={() => {
                  setLimit(20);
                  setMobileReader(false);
                  onSearch({ responseEngine: item.id, answerId: undefined });
                }}
              >
                <BrandPill name={item.name} domain={item.domain} />
                <span>{percent(rate)}</span>
              </button>
            );
          })}
        </fieldset>
        {!answers.length ? (
          <p className="hc-empty">{C.noResponses}</p>
        ) : (
          <div className="hc-response-layout" data-reading={mobileReader || undefined}>
            <div className="hc-response-history">
              <ul aria-label={C.responseHistory}>
                {answers.slice(0, limit).map((record) => (
                  <li key={record.result.id}>
                    <button
                      type="button"
                      aria-pressed={selected?.result.id === record.result.id}
                      onClick={(event) => {
                        lastTrigger.current = event.currentTarget;
                        setMobileReader(true);
                        onSearch({ responseEngine: engine, answerId: record.result.id });
                      }}
                    >
                      <time dateTime={record.answer.date}>{record.answer.date}</time>
                      <span>
                        {record.result.outcome
                          ? M[record.result.outcome]
                          : record.stopped
                            ? M.notCollected
                            : M.pending}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
              {answers.length > limit && (
                <Button variant="ghost" size="sm" onClick={() => setLimit((value) => value + 20)}>
                  {C.older}
                </Button>
              )}
            </div>
            <section
              className="hc-response-reader"
              ref={reader}
              tabIndex={-1}
              aria-label={C.responses}
            >
              <Button
                variant="ghost"
                size="sm"
                className="hc-back-responses"
                onClick={() => {
                  setMobileReader(false);
                  requestAnimationFrame(() => lastTrigger.current?.focus());
                }}
              >
                {C.backResponses}
              </Button>
              {selected ? (
                <>
                  <div className="hc-answer-meta">
                    {engineInfo && <BrandPill name={engineInfo.name} domain={engineInfo.domain} />}
                    <time dateTime={selected.answer.date}>{selected.answer.date}</time>
                    <span>{M.synthetic}</span>
                  </div>
                  <AnswerContent
                    result={selected.result}
                    brandName={brandName}
                    stopped={selected.stopped}
                  />
                </>
              ) : (
                <output>{search.answerId ? C.responseMissing : C.selectResponse}</output>
              )}
            </section>
          </div>
        )}
      </section>
      {editing && (
        <PromptEditor
          prompt={row.prompt}
          onClose={() => setEditing(false)}
          onSave={(text) => onSave(text, row.prompt.id)}
          finalFocus={() => editTrigger.current}
        />
      )}
    </section>
  );
}
