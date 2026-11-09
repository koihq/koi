import type { Act } from "../kit/act";
import { States } from "../kit/motion/States";
import { Check } from "../kit/parts/Check";
import { Stage } from "../kit/stage";
import { EngineLogo } from "../logos";
import { OutcomeDot } from "./parts";
import "./demos.css";

// What happens after the brief. The agent works through its edits one by one, and the next daily
// check is the payoff: the same question on the same engine now cites the brand. A bento tile shows
// one component, so the agent's work is a bare card: the setup tile above owns the window chrome.
// The card sits whole inside the tile, so every edge it shows is its own.

const TEXT = {
  brief: "AI answer evidence",
  question: "Open source alternative to Profound",
  meta: "Perplexity, not in answer, Profound and Peec AI cited",
  next: "Next daily check",
  miss: "Not in answer",
  cited: "Koi cited",
};

const STEPS = [
  "Read tryprofound.com and peec.ai",
  "Added docs/compare/profound.md",
  "Answered self-hosting on /pricing",
  "Opened a pull request",
];

type S = { step: number; working: boolean; next: boolean; cited: boolean };

const initial: S = { step: 0, working: false, next: false, cited: false };
const poster: S = { step: STEPS.length, working: false, next: true, cited: true };

async function script(a: Act<S>) {
  await a.wait(700);
  a.put({ working: true });

  for (let step = 1; step <= STEPS.length; step++) {
    await a.wait(step === 1 ? 700 : 850);
    a.put({ step });
  }

  a.put({ working: false });
  await a.wait(700);
  a.mark("next");
  a.put({ next: true });
  await a.wait(1100);
  a.mark("cited");
  a.put({ cited: true });
  await a.wait(1800);
}

export function FixDemo() {
  return (
    <Stage
      width={836}
      height={266}
      compact={{ below: 600, width: 358, height: 316 }}
      initial={initial}
      poster={poster}
      script={script}
      play="once"
      theme="dark"
      fidelity="realistic"
      className="koi-demo kf"
    >
      {(s) => (
        <>
          <div className="kf-window">
            <div className="kf-body">
              <div className="kf-brief">
                <div className="kf-brief__label">{TEXT.brief}</div>
                <div className="kf-brief__q">{TEXT.question}</div>
                <div className="kf-brief__meta">{TEXT.meta}</div>
              </div>

              <div className="kf-steps">
                {STEPS.map((step, index) => (
                  <div key={step} className="kf-step" data-on={s.step >= index || undefined}>
                    <Check size={16} on={s.step > index} pending={s.working && s.step === index} />
                    {step}
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="kf-next" data-on={s.next || undefined}>
            <div className="kf-next__label">{TEXT.next}</div>
            <div className="kf-next__row">
              <EngineLogo engine="perplexity" size={18} decorative />
              <OutcomeDot outcome={s.cited ? "cited" : "miss"} size={14} />
              <States
                value={s.cited ? "cited" : "miss"}
                align="start"
                states={{ miss: TEXT.miss, cited: TEXT.cited }}
              />
            </div>
          </div>
        </>
      )}
    </Stage>
  );
}
