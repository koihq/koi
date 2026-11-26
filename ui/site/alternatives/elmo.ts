import { type Comparison, compare, koiQuestions, no, pricedAt, unstated, yes } from "./comparison";
import { koiPrice } from "./tools";

export const content: Comparison = {
  heading: "Koi, an Elmo alternative with unlimited brands",
  lead: `Koi and Elmo are both open source under MIT. Koi Cloud has unlimited brands on every plan, and all four of its engines from ${koiPrice(15)} a month.`,
  intro:
    "Elmo Cloud starts at $29 a month for 50 prompts on ChatGPT only, and $99 for 50 prompts on four engines of your choice, checked up to four times a day. Each plan covers one brand, two on Pro and five on Business. Elmo also has more engines, unlimited seats and a REST API. Koi checks once a day and costs less to start on four engines.",
  pick: {
    koi: "if you track several brands, want four engines from the first plan, or want skills that turn an answer into changes in your repo.",
    rival:
      "if you need several checks a day, Claude, Perplexity or Bing AI Mode, a REST API, or a team that shares the dashboard.",
  },
  rows: [
    {
      feature: "Smallest plan",
      koi: yes(`${koiPrice(15)}/mo, 15 prompts`),
      rival: yes("$29/mo, 50 prompts"),
    },
    {
      feature: "Four engines",
      koi: yes(`From ${koiPrice(15)}/mo`),
      rival: yes("From $99/mo"),
    },
    pricedAt(50, yes("$99/mo on 4 engines")),
    pricedAt(150, yes("$299/mo")),
    compare("free", unstated),
    compare("openSource", yes("MIT")),
    compare("brands", yes("1, up to 5 on Business")),
    compare("googleEngines", yes("Any 4, from $99/mo")),
    { feature: "Claude, Perplexity, Bing AI Mode", koi: no(), rival: yes("Perplexity from Pro") },
    compare("daily", yes("Up to 4 times a day")),
    compare("sources", yes()),
    compare("competitors", yes()),
    compare("sentiment", yes()),
    compare("mcp", yes("Every plan")),
    compare("skills", yes("One AEO skill")),
    compare("seats", yes("Unlimited")),
    compare("api", yes("With TypeScript and Python SDKs")),
  ],
  topics: {
    pricing:
      "Starter is $29 a month for 50 prompts on ChatGPT only. Basic is $99 for 50 prompts on any four platforms, Pro $299 for 150 and Business $649 for 350. Each plan covers one brand, two on Pro and five on Business. Annual billing gives two months free. Self-hosting is free, with the provider keys you choose.",
    data: "Five engines through their websites: ChatGPT, Google AI Mode, Google AI Overviews, Bing AI Mode and Gemini. Models through their APIs: Qwen, DeepSeek, Mistral, Claude and Kimi, and from Pro, Perplexity Sonar and the web versions of Claude, Grok and Meta. Starter checks once a day, the other plans up to four times a day.",
    agents:
      "An MCP server for Claude Code, Codex, Cursor and VS Code, a REST API with TypeScript and Python SDKs, and an AEO skill for agents. All of them come with every plan and every self-hosted instance.",
    team: "Unlimited seats on every plan. White-label instances for agencies, with their own domain and SSO, are by request.",
  },
  questions: [
    {
      q: "Is Elmo open source?",
      a: "Yes, under MIT, like Koi. Elmo has been public on GitHub since August 2025. Koi's first release came out in November 2026.",
    },
    {
      q: "How much does Elmo cost?",
      a: "Elmo Cloud lists Starter at $29 a month for 50 prompts on ChatGPT only, Basic at $99 for 50 prompts on four platforms, Pro at $299 for 150 and Business at $649 for 350. Self-hosting is free.",
    },
    {
      q: "Does Elmo check more often than Koi?",
      a: "Yes. Elmo's Basic, Pro and Business plans run each prompt up to four times a day on every platform you choose. Koi runs each prompt once a day.",
    },
    ...koiQuestions,
  ],
};
