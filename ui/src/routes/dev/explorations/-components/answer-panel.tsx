import type { VisibilityResult } from "@/api/runs.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { safeSourceURL } from "@/lib/source-url";
import { ENGINES } from "../-lib/data";
import { HOME_LOGOS, type HomeAnswer } from "../-lib/home-evidence";
import C from "../-locales/home-signal.en.json";
import M from "../-locales/mvp.en.json";
import V from "../-locales/prompt-visibility.en.json";
import { HomeBrandText } from "./home-brand-text";

export function AnswerContent({
  result,
  brandName,
  stopped,
}: {
  result: VisibilityResult;
  brandName: string;
  stopped: boolean;
}) {
  return (
    <div className="hc-answer-content">
      <blockquote className="hc-original-question">
        <HomeBrandText text={result.promptText} />
      </blockquote>
      <AnswerBody result={result} brandName={brandName} stopped={stopped} />
      <SourceList result={result} />
    </div>
  );
}

export function AnswerBody({
  result,
  brandName,
  stopped,
}: {
  result: VisibilityResult;
  brandName: string;
  stopped: boolean;
}) {
  return (
    <>
      <div className="hc-answer-outcome">
        <BrandPill name={brandName} {...HOME_LOGOS[brandName]} />
        <span>{result.outcome ? M[result.outcome] : stopped ? M.notCollected : M.pending}</span>
      </div>
      <p className="hc-answer-detail">
        {result.rawAnswer ? (
          <HomeBrandText text={result.rawAnswer} />
        ) : result.outcome === "error" ? (
          M.answerFailed
        ) : result.outcome === null && !stopped ? (
          M.answerPending
        ) : (
          M.noAnswer
        )}
      </p>
      {result.appearances.length > 0 && (
        <section className="hc-detail-section" aria-label={M.appearances}>
          <h3>{M.appearances}</h3>
          {result.appearances.map((appearance) => (
            <div key={appearance.competitorId ?? appearance.name}>
              <BrandPill name={appearance.name} {...HOME_LOGOS[appearance.name]} />
              {appearance.evidence.map((evidence) => (
                <p key={`${evidence.kind}:${evidence.value}`}>{evidence.value}</p>
              ))}
            </div>
          ))}
        </section>
      )}
      <p className="hc-caption">{C.detailNote}</p>
    </>
  );
}

export function SourceList({
  result,
  sample = false,
  emptyMessage = M.noSources,
}: {
  result: VisibilityResult;
  sample?: boolean;
  emptyMessage?: string;
}) {
  const title = sample && result.sources.length ? V.sampleSources : M.sources;

  return (
    <section className="hc-detail-section" aria-label={title}>
      <h3>{title}</h3>
      {result.sources.length ? (
        result.sources.map((source) => {
          const url = safeSourceURL(source);

          return url ? (
            <a key={source} href={url} target="_blank" rel="noopener noreferrer">
              {source}
            </a>
          ) : (
            <p key={source}>{source}</p>
          );
        })
      ) : (
        <p>{emptyMessage}</p>
      )}
    </section>
  );
}

export function AnswerPanel({
  answer,
  result,
  brandName,
  stopped,
  onClose,
  finalFocus,
}: {
  answer?: HomeAnswer;
  result?: VisibilityResult;
  brandName: string;
  stopped: boolean;
  onClose: () => void;
  finalFocus: () => HTMLElement | null;
}) {
  const engine = ENGINES.find((item) => item.id === answer?.engine);

  return (
    <Dialog
      open={Boolean(result)}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent className="hc-answer-panel" finalFocus={finalFocus}>
        <DialogHeader>
          <DialogTitle>{result && <HomeBrandText text={result.promptText} />}</DialogTitle>
          <DialogDescription>
            <span className="hc-answer-meta">
              {engine && <BrandPill name={engine.name} domain={engine.domain} />}
              <span>
                {answer?.date}
                {C.separator}
                {M.synthetic}
              </span>
            </span>
          </DialogDescription>
        </DialogHeader>
        {result && <AnswerContent result={result} brandName={brandName} stopped={stopped} />}
      </DialogContent>
    </Dialog>
  );
}
