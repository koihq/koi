import type { Question } from "../schema";
import { koiPrice } from "./tools";

/**
 * One answer in the table: a mark, and what it costs or how much you get. When the vendor's page
 * does not say, the cell is `unstated`, never a "no" we inferred.
 */
export type Cell = { mark: "yes" | "no" | "unstated"; text?: string };

export const yes = (text?: string): Cell => ({ mark: "yes", text });
export const no = (text?: string): Cell => ({ mark: "no", text });
export const unstated: Cell = { mark: "unstated", text: "Not published" };

export type Row = { feature: string; koi: Cell; rival: Cell };

/** Koi's side of the rows most pages share, so two pages never describe Koi differently. */
const koiRows = {
  free: { feature: "Try it free", koi: yes("First check, no card") },
  openSource: { feature: "Open source, run it yourself", koi: yes("MIT") },
  brands: { feature: "Brands", koi: yes("Unlimited on every plan") },
  googleEngines: {
    feature: "ChatGPT, Gemini, Google AI Mode, AI Overviews",
    koi: yes("All four on every plan"),
  },
  daily: { feature: "Checked every day", koi: yes("Once a day") },
  sources: { feature: "Sources behind each answer", koi: yes() },
  competitors: { feature: "Competitors named instead of you", koi: yes() },
  countries: { feature: "Countries", koi: yes("One per brand") },
  sentiment: { feature: "Sentiment", koi: no() },
  traffic: { feature: "AI crawler and referral traffic", koi: no() },
  mcp: { feature: "MCP server", koi: yes("Every plan") },
  skills: { feature: "Skills for Claude Code, Codex and Cursor", koi: yes("Setup, brief and fix") },
  seats: { feature: "Team seats", koi: no("One person") },
  sso: { feature: "SSO", koi: no() },
  soc2: { feature: "SOC 2 report", koi: no() },
  exports: { feature: "CSV export", koi: no() },
  api: { feature: "REST API", koi: no("Agents use MCP") },
} satisfies Record<string, Omit<Row, "rival">>;

export function compare(id: keyof typeof koiRows, rival: Cell): Row {
  return { ...koiRows[id], rival };
}

/** What the same number of prompts costs on each side. */
export function pricedAt(prompts: number, rival: Cell): Row {
  return { feature: `${prompts} prompts`, koi: yes(`${koiPrice(prompts)}/mo`), rival };
}

export const topics = ["pricing", "data", "agents", "team"] as const;

export type Topic = (typeof topics)[number];

/** Koi's side of each topic. The page puts the rival's beside it. */
export const koiTopics: Record<Topic, string> = {
  pricing: `You pay for the prompts you track: ${koiPrice(15)} a month for 15, ${koiPrice(50)} for 50, ${koiPrice(100)} for 100, tax included. Every plan has unlimited brands, which share its prompts, and all four engines, the MCP server and the skills. The first check is free and needs no card. Daily tracking starts with a 7-day trial. Running Koi on your own server is free, with your own Cloro account to collect the answers.`,
  data: "ChatGPT, Gemini, Google AI Mode and Google AI Overviews. Each prompt runs once a day, in one country per brand. For every answer Koi keeps the full text, whether you are cited, mentioned or missing, who is named instead, and the sources.",
  agents:
    "An MCP server and three skills. Claude Code, Codex or Cursor can set Koi up for your project, read the results, and turn an answer into changes in your repo.",
  team: "One person per account. There are no team seats, SSO or SOC 2 report. Koi is new: its first release came out in November 2026.",
};

/** One page's words. Which rival it is comes from `list.ts`. */
export type Comparison = {
  heading: string;
  /** The answer in one sentence, for a reader or an engine that stops here. */
  lead: string;
  intro: string;
  /** Each finishes "Choose Koi …" and "Choose <rival> …". */
  pick: { koi: string; rival: string };
  rows: Row[];
  /** The rival's side of each topic. */
  topics: Record<Topic, string>;
  questions: Question[];
};

/** Asked about Koi on every page. */
export const koiQuestions: Question[] = [
  {
    q: "Is Koi free?",
    a: `The first check is free and needs no card: 5 prompts, asked once on all four engines. Daily tracking starts at ${koiPrice(15)} a month for 15 prompts, after a 7-day trial. The code is MIT licensed, so you can also run Koi on your own server with your own Cloro account.`,
  },
  {
    q: "Can I track more than one brand with Koi?",
    a: "Yes. Every plan has unlimited brands, and they share the plan's prompts. The free first check covers one brand.",
  },
];
