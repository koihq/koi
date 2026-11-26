import { type Comparison, compare, koiQuestions, no, pricedAt, unstated, yes } from "./comparison";
import { koiPrice } from "./tools";

export const content: Comparison = {
  heading: "Koi, an open source Otterly.AI alternative",
  lead: `Koi is an open source Otterly.AI alternative with the MCP server on every plan, from ${koiPrice(15)} a month for 15 prompts, tax included.`,
  intro: `Otterly.AI also starts at €29 a month, for 16 prompts before tax, with unlimited brands, seats and countries. Its MCP server and API start with Standard, at €189 for 120 prompts. Koi has plans in between, ${koiPrice(50)} for 50 prompts and ${koiPrice(100)} for 100, each with the MCP server and skills that hand an answer to Claude Code, Codex or Cursor.`,
  pick: {
    koi: "if you want your agent to read the results from the first plan, or need between 16 and 120 prompts.",
    rival:
      "if a marketing team needs Perplexity, Copilot or Claude, several countries, sentiment or Looker Studio reports.",
  },
  rows: [
    {
      feature: "Smallest plan",
      koi: yes(`${koiPrice(15)}/mo, 15 prompts`),
      rival: yes("€29/mo, 16 prompts"),
    },
    { feature: "Tax in the price", koi: yes("Included"), rival: no("Added on top") },
    pricedAt(50, no("Next plan: 120 for €189/mo")),
    {
      feature: "120 to 150 prompts",
      koi: yes(`${koiPrice(150)}/mo for 150`),
      rival: yes("€189/mo for 120"),
    },
    compare("free", yes("Free trial")),
    compare("openSource", no()),
    compare("brands", yes("Unlimited on every plan")),
    compare("googleEngines", yes("Any 4 of 7")),
    { feature: "Perplexity, Copilot, Claude", koi: no(), rival: yes("Within the 4") },
    compare("daily", yes("Daily, or weekly on Premium")),
    compare("sources", yes()),
    compare("competitors", yes("Unlimited")),
    compare("countries", yes("Unlimited")),
    compare("sentiment", yes()),
    compare("traffic", yes("From €189/mo")),
    compare("mcp", yes("From €189/mo")),
    compare("skills", unstated),
    compare("seats", yes("Unlimited")),
    { feature: "Reports and export", koi: no(), rival: yes("Every plan") },
    { feature: "Looker Studio connector", koi: no(), rival: yes("From €189/mo") },
    compare("api", yes("From €189/mo")),
    compare("sso", yes("Enterprise")),
  ],
  topics: {
    pricing:
      "Lite is €29 a month for 16 prompts, Standard €189 for 120 and Premium €489 for 403, all before tax. Prompts are paid in credits: one prompt checked once on one engine is one credit, and a Claude answer counts as three. Annual billing takes off 15%. Enterprise starts at 1,000 prompts and is priced by sales.",
    data: "Four of seven engines on each plan: ChatGPT, Google AI Overviews, Perplexity, Microsoft Copilot, Google AI Mode, Gemini and Claude. Prompts run daily, or weekly on Premium. Every plan has unlimited brands, competitors and countries, with sentiment, cited sources and reports.",
    agents:
      "An MCP server and an API from Standard, each with 2,000 requests a month, plus a ChatGPT plugin and a Looker Studio connector. Lite has none of them.",
    team: "Unlimited team members on every plan. SSO comes with Enterprise. The pricing page does not mention SOC 2.",
  },
  questions: [
    {
      q: "How much does Otterly.AI cost?",
      a: "Its pricing page lists Lite at €29 a month for 16 prompts, Standard at €189 for 120 and Premium at €489 for 403, before tax. Annual billing takes off 15%.",
    },
    {
      q: "Does Otterly.AI have an MCP server?",
      a: "Yes, from the Standard plan, with 2,000 requests a month. Koi has one on every plan.",
    },
    {
      q: "Which engines does Otterly.AI track?",
      a: "Four of seven on each plan: ChatGPT, Google AI Overviews, Perplexity, Microsoft Copilot, Google AI Mode, Gemini and Claude. Koi tracks ChatGPT, Gemini, Google AI Mode and Google AI Overviews.",
    },
    ...koiQuestions,
  ],
};
