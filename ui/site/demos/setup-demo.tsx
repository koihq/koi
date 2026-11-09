import { agentPrompt } from "../copy";
import type { Act } from "../kit/act";
import { Field } from "../kit/controls/Field";
import { Collapse } from "../kit/motion/Collapse";
import { Popover } from "../kit/motion/Popover";
import { Stream } from "../kit/motion/Stream";
import { Check } from "../kit/parts/Check";
import { Stage } from "../kit/stage";
import { EngineLogo, engineList } from "../logos";
import { Frame } from "./parts";
import "./demos.css";

// How the first check really goes. The prompt from the hero is sent, the agent asks for the one
// sign-in, the person approves it, and the agent does the rest in the thread: it reads the repo,
// drafts the questions, asks the engines and reports who was named instead. Plays once.

const TEXT = {
  title: "Agent",
  placeholder: "Ask your agent",
  signInTitle: "Allow your agent to use Koi?",
  signInBody:
    "It can read and change your brand, prompts and checks. It can't change what you pay.",
  allow: "Allow",
  cancel: "Cancel",
  question: "Best AEO tool for developers",
};

const STEPS = [
  "Connected to Koi",
  "Found Koi in this repo",
  "Drafted 26 questions and 6 competitors",
  "Asked the four engines",
];

const REPORT =
  "ChatGPT and Google AI Mode name Profound and Peec AI, not you. ChatGPT cites tryprofound.com. Want a brief with the changes for this repo?";
const WORDS = REPORT.split(" ").length;

const REST = { wide: { x: 700, y: 330 }, compact: { x: 310, y: 300 } };

type S = {
  sent: boolean;
  signIn: boolean;
  step: number;
  engines: number;
  words: number;
};

const initial: S = { sent: false, signIn: false, step: 0, engines: 0, words: 0 };
const poster: S = {
  sent: true,
  signIn: false,
  step: STEPS.length,
  engines: engineList.length,
  words: WORDS,
};

async function script(a: Act<S>) {
  await a.wait(800);
  a.mark("send");
  a.put({ sent: true });
  await a.wait(900);

  a.mark("sign-in");
  a.put({ signIn: true });
  await a.wait(1100);
  await a.move({ at: "allow", x: 0.8, y: 0.6 });
  await a.wait(200);
  await a.click();
  a.put({ signIn: false, step: 1 });
  a.move(REST[a.layout]);
  await a.wait(800);

  a.put({ step: 2 });
  await a.wait(850);
  a.put({ step: 3 });
  await a.wait(850);

  a.mark("engines");
  await a.stagger(
    engineList.map((_, index) => index),
    (index) => a.put({ engines: index + 1 }),
    260,
  );
  await a.wait(500);
  a.put({ step: 4 });
  await a.wait(500);

  a.mark("report");
  for (let word = 1; word <= WORDS; word++) {
    a.put({ words: word });
    await a.wait(40);
  }

  await a.wait(1800);
}

export function SetupDemo() {
  return (
    <Stage
      width={836}
      height={430}
      compact={{ below: 600, width: 358, height: 470 }}
      initial={initial}
      poster={poster}
      script={script}
      play="once"
      theme="dark"
      fidelity="realistic"
      cursor={{ at: REST }}
      className="koi-demo ku"
    >
      {(s, _rect, layout) => (
        <Frame layout={layout} title={TEXT.title} className="ku-window">
          <div className="ku-body">
            <div className="ku-thread">
              <Collapse open={s.sent} reveal="drop" from="below">
                <div className="ku-me">{agentPrompt}</div>
              </Collapse>

              <div className="ku-steps">
                {STEPS.map((step, index) => (
                  <Collapse key={step} open={s.step >= index && (s.signIn || s.step > 0)}>
                    <div className="ku-step">
                      <Check size={16} on={s.step > index} pending={s.step === index} />
                      <span>{step}</span>
                      {index === STEPS.length - 1 && (
                        <span className="ku-engines">
                          {engineList.map((engine, engineIndex) => (
                            <span
                              key={engine}
                              className="ku-engine"
                              data-on={s.engines > engineIndex || undefined}
                            >
                              <EngineLogo engine={engine} size={14} decorative />
                            </span>
                          ))}
                        </span>
                      )}
                    </div>
                  </Collapse>
                ))}
              </div>

              <Collapse open={s.words > 0}>
                <div className="ku-report">
                  <div className="ku-report__q">{TEXT.question}</div>
                  <Stream className="ku-report__a" text={REPORT} count={s.words} />
                </div>
              </Collapse>
            </div>

            <Field
              className="ku-composer"
              value={s.sent ? "" : agentPrompt}
              placeholder={TEXT.placeholder}
            />
          </div>

          <div className="ku-scrim" data-on={s.signIn || undefined} />
          <Popover show={s.signIn} origin="center" className="ku-signin">
            <div className="ku-signin__title">{TEXT.signInTitle}</div>
            <div className="ku-signin__body">{TEXT.signInBody}</div>
            <div className="ku-signin__actions">
              <span className="ku-signin__cancel">{TEXT.cancel}</span>
              <span className="ku-signin__allow" data-target="allow" data-cursor="pointer">
                {TEXT.allow}
              </span>
            </div>
          </Popover>
        </Frame>
      )}
    </Stage>
  );
}
