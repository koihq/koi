import chatgpt from "./assets/logos/chatgpt.svg";
import chatgptDark from "./assets/logos/chatgpt-dark.svg";
import claude from "./assets/logos/claude.svg";
import cursor from "./assets/logos/cursor.svg";
import gemini from "./assets/logos/gemini.svg";
import github from "./assets/logos/github.svg";
import githubDark from "./assets/logos/github-dark.svg";
import google from "./assets/logos/google.svg";
import perplexity from "./assets/logos/perplexity.svg";
import { engineNames } from "./copy";

export type Engine = keyof typeof engineNames;

export const engineList: Engine[] = ["chatgpt", "perplexity", "gemini", "google"];

const engineArt: Record<Engine, { light: string; dark?: string }> = {
  chatgpt: { light: chatgpt, dark: chatgptDark },
  perplexity: { light: perplexity },
  gemini: { light: gemini },
  google: { light: google },
};

function Art({
  light,
  dark,
  size,
  label,
}: {
  light: string;
  dark?: string;
  size: number;
  label: string;
}) {
  if (!dark) {
    return (
      <img
        src={light}
        alt={label}
        width={size}
        height={size}
        draggable={false}
        className="site-logo"
      />
    );
  }

  return (
    <span
      className="site-logo"
      {...(label ? { role: "img", "aria-label": label } : { "aria-hidden": true })}
    >
      <img src={light} alt="" width={size} height={size} draggable={false} data-theme-art="light" />
      <img src={dark} alt="" width={size} height={size} draggable={false} data-theme-art="dark" />
    </span>
  );
}

/** A real engine mark. The name is the accessible label, and decorative repeats pass `decorative`. */
export function EngineLogo({
  engine,
  size = 16,
  decorative = false,
}: {
  engine: Engine;
  size?: number;
  decorative?: boolean;
}) {
  const art = engineArt[engine];

  return (
    <Art
      light={art.light}
      dark={art.dark}
      size={size}
      label={decorative ? "" : engineNames[engine]}
    />
  );
}

export function GithubLogo({ size = 16 }: { size?: number }) {
  return <Art light={github} dark={githubDark} size={size} label="" />;
}

const agentArt = [
  { name: "Claude Code", light: claude },
  { name: "Codex", light: chatgpt, dark: chatgptDark },
  { name: "Cursor", light: cursor, invert: true },
];

/** The agents the prompt works with, overlapped like avatars. The button beside it names them. */
export function AgentStack({ size = 16 }: { size?: number }) {
  return (
    <span aria-hidden="true" className="flex -space-x-2">
      {agentArt.map((agent) => (
        <span key={agent.name} className="site-agent-mark">
          {agent.invert ? (
            <img
              src={agent.light}
              alt=""
              width={size}
              height={size}
              draggable={false}
              className="dark:invert"
            />
          ) : (
            <Art light={agent.light} dark={agent.dark} size={size} label="" />
          )}
        </span>
      ))}
    </span>
  );
}
