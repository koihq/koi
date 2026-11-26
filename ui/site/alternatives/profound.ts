import { type Comparison, compare, koiQuestions, no, unstated, yes } from "./comparison";
import { koiPrice } from "./tools";

export const content: Comparison = {
  heading: "Koi, an open source Profound alternative",
  lead: `Koi is an open source Profound alternative you can start on your own: ${koiPrice(15)} a month for 15 prompts, or on your own server with your own Cloro account.`,
  intro:
    "Profound sells Enterprise plans at a custom price, after a demo, with up to ten engines, unlimited seats and SSO. Koi tracks four engines: ChatGPT, Gemini, Google AI Mode and Google AI Overviews. It keeps every answer with its sources and turns any answer into a brief that Claude Code, Codex or Cursor can act on in your repo.",
  pick: {
    koi: "if you build with a coding agent and want to start small: a few prompts, four engines, as many brands as you need, and changes your agent makes in your repo.",
    rival:
      "if a marketing team at a large company needs more engines, prompt volumes, crawler analytics, seats and SSO, and is ready to talk to sales.",
  },
  rows: [
    {
      feature: "Public price",
      koi: yes(`${koiPrice(15)}/mo for 15 prompts`),
      rival: no("Custom, after a demo"),
    },
    compare("free", unstated),
    compare("openSource", no()),
    compare("brands", unstated),
    compare("googleEngines", yes()),
    {
      feature: "Perplexity, Claude, DeepSeek",
      koi: no(),
      rival: yes("Up to 10 engines in all"),
    },
    compare("daily", yes("Daily")),
    compare("sources", yes()),
    compare("competitors", yes()),
    compare("countries", yes("Custom regions")),
    compare("sentiment", yes()),
    { feature: "Prompt volumes", koi: no(), rival: yes() },
    compare("traffic", yes()),
    compare("mcp", yes()),
    compare("skills", unstated),
    {
      feature: "Agents that write content",
      koi: no("Your own agent does"),
      rival: yes("AI Marketer, on credits"),
    },
    compare("seats", yes("Unlimited")),
    compare("sso", yes("SSO and SAML")),
    compare("soc2", yes()),
    compare("exports", yes("CSV and JSON")),
    compare("api", yes()),
  ],
  topics: {
    pricing:
      "One plan, Enterprise, at a custom price. Prompts, engines, regions and AI Marketer credits are set for each customer, and you get a number after a demo.",
    data: "Up to ten engines: ChatGPT, Perplexity, Google AI Mode, Gemini, DeepSeek, Claude, Google AI Overviews, Exa Search, Muse Spark and Qwen. Prompts run daily, in the regions and languages you agree on. Profound also shows prompt volumes, ChatGPT Shopping and AI crawler traffic.",
    agents:
      "An MCP server for Claude, ChatGPT and Cursor, to ask about your visibility, add prompts and run Profound's own agents. AI Marketer researches and writes content inside Profound, on credits.",
    team: "Unlimited seats, SSO and SAML, SOC 2, a dedicated Slack channel and a dedicated specialist with a 24-hour SLA.",
  },
  questions: [
    {
      q: "How much does Profound cost?",
      a: "Profound's pricing page lists one plan, Enterprise, at a custom price. You get a number after a demo.",
    },
    {
      q: "Does Koi track as many engines as Profound?",
      a: "No. Koi tracks ChatGPT, Gemini, Google AI Mode and Google AI Overviews. Profound lists up to ten, including Perplexity, Claude and DeepSeek.",
    },
    {
      q: "Do Koi and Profound work with Claude Code or Cursor?",
      a: "Both have an MCP server. Koi also has skills that set it up for your project and turn an answer into changes in your repo. Profound's MCP lets you ask about your data, add prompts and run Profound's own agents.",
    },
    ...koiQuestions,
  ],
};
