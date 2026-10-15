import { useRef, useState } from "react";
import type { VisibilityResult } from "@/api/runs.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { ENGINES } from "../-lib/data";
import { answerResult } from "../-lib/exploration-fixtures";
import { HOME_BRANDS, HOME_LOGOS, type HomeAnswer, type HomeBrand } from "../-lib/home-evidence";
import C from "../-locales/home-signal.en.json";
import M from "../-locales/mvp.en.json";
import { AnswerPanel } from "./answer-panel";
import { HomeBrandText } from "./home-brand-text";

export function AnswerFeed({
  answers,
  brand,
  results,
  stopped = false,
}: {
  answers: HomeAnswer[];
  brand: HomeBrand;
  results?: VisibilityResult[];
  stopped?: boolean;
}) {
  const [selected, setSelected] = useState<{ answer: HomeAnswer; result: VisibilityResult } | null>(
    null,
  );
  const trigger = useRef<HTMLElement | null>(null);

  return (
    <>
      <section className="hc-answer-feed" aria-label={C.latest}>
        {answers.map((answer, index) => {
          const result = results?.[index] ?? answerResult(answer, brand);
          const engine = ENGINES.find((item) => item.id === answer.engine);
          const included = result.outcome === "mentioned" || result.outcome === "cited";
          const completed = result.outcome !== null && result.outcome !== "error";

          return (
            <button
              type="button"
              className="hc-answer-row"
              key={answer.id}
              onClick={(event) => {
                trigger.current = event.currentTarget;
                setSelected({ answer, result });
              }}
            >
              <span className="hc-answer-meta">
                <BrandPill name={engine?.name ?? answer.engine} domain={engine?.domain} />
                <small>{answer.date}</small>
              </span>
              <span className="hc-answer-text">
                <HomeBrandText text={answer.prompt} />
              </span>
              <span className="hc-answer-outcome">
                <span className={included ? "hc-mentioned" : completed ? "hc-absent" : undefined}>
                  {result.outcome ? M[result.outcome] : stopped ? M.notCollected : M.pending}
                </span>
                {completed && (
                  <>
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
                  </>
                )}
              </span>
            </button>
          );
        })}
      </section>
      <AnswerPanel
        answer={selected?.answer}
        result={selected?.result}
        brandName={HOME_BRANDS[brand].name}
        stopped={stopped}
        onClose={() => setSelected(null)}
        finalFocus={() =>
          trigger.current?.isConnected
            ? trigger.current
            : document.getElementById("hc-page-heading")
        }
      />
    </>
  );
}
