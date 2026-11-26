import { type Comparison, compare, koiQuestions, no, pricedAt, unstated, yes } from "./comparison";
import { koiPrice } from "./tools";

export const content: Comparison = {
  heading: "Koi, an open source Peec AI alternative",
  lead: `Koi is an open source Peec AI alternative that starts smaller: ${koiPrice(15)} a month for 15 prompts on four engines, tax included.`,
  intro: `Peec AI starts at $95 a month for 50 prompts on three engines you choose, with unlimited users. At 50 prompts Koi costs ${koiPrice(50)} a month with all four of its engines and unlimited brands. Peec AI does more for a marketing team: sentiment, Perplexity, shopping, crawler data and an agent of its own. Koi hands each answer to the coding agent you already use.`,
  pick: {
    koi: "if you work alone or in a small team, ship with Claude Code, Codex or Cursor, and ChatGPT, Gemini and Google's two AI surfaces are the engines you need.",
    rival:
      "if a marketing team shares the dashboard and needs sentiment, Perplexity, several countries or reports built in.",
  },
  rows: [
    {
      feature: "Smallest plan",
      koi: yes(`${koiPrice(15)}/mo, 15 prompts`),
      rival: yes("$95/mo, 50 prompts"),
    },
    pricedAt(50, yes("$95/mo")),
    pricedAt(150, yes("$245/mo")),
    compare("free", yes("Free trial")),
    compare("openSource", no()),
    compare("brands", yes("1 to 5, unlimited on Enterprise")),
    compare("googleEngines", yes("Any 3, more from $30/mo each")),
    { feature: "Perplexity, Bing AI Mode, Naver, ERNIE", koi: no(), rival: yes() },
    compare("daily", yes("Daily")),
    compare("sources", yes()),
    compare("competitors", yes()),
    compare("countries", yes("1 per project, 3 from Pro")),
    compare("sentiment", yes("With position, share of voice")),
    { feature: "Shopping and ads in answers", koi: no(), rival: yes() },
    compare("traffic", yes()),
    compare("mcp", yes("Every plan")),
    compare("skills", unstated),
    {
      feature: "Built-in AI agent",
      koi: no("Works with yours"),
      rival: yes("500 credits on Starter"),
    },
    compare("seats", yes("Unlimited, with roles")),
    compare("exports", yes()),
    compare("api", yes("From $245/mo")),
    compare("sso", yes("Enterprise")),
    compare("soc2", yes("Type II")),
  ],
  topics: {
    pricing:
      "Starter is $95 a month for 50 prompts, Pro $245 for 150 and Advanced $495 for 350, each with three engines you choose. Starter has one project, Pro two and Advanced five. Another engine costs $30 a month on Starter. Annual billing takes off 15%. Enterprise, priced by sales, has unlimited projects. The pricing page does not say whether tax is included.",
    data: "Three engines out of eight: ChatGPT, Perplexity, Google AI Overviews, Google AI Mode, Gemini, Naver AI Briefing, ERNIE and Bing AI Mode. Enterprise tracks up to 26 models. Prompts run daily. Peec AI shows visibility, position, sentiment and share of voice, plus shopping, ads and AI crawler data.",
    agents:
      "An MCP server on every plan and a REST API from Pro. Peec AI also has its own agent, with credits and skills, for reports and suggested actions inside the app.",
    team: "Unlimited users with roles on every plan, and a SOC 2 Type II report. SSO comes with Enterprise.",
  },
  questions: [
    {
      q: "How much does Peec AI cost?",
      a: "Its pricing page lists Starter at $95 a month for 50 prompts, Pro at $245 for 150 and Advanced at $495 for 350. Each includes three engines, checked daily. Annual billing takes off 15%.",
    },
    {
      q: "Is Koi cheaper than Peec AI?",
      a: `At 50 prompts, Koi is ${koiPrice(50)} a month for four engines, tax included, and Peec AI is $95 for three. Peec AI includes unlimited users, sentiment and CSV exports, which Koi does not have.`,
    },
    {
      q: "Does Peec AI have an MCP server?",
      a: "Yes, on every plan, and so does Koi. Peec AI also has a REST API from the Pro plan. Koi does not.",
    },
    ...koiQuestions,
  ],
};
