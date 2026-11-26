import type { Question } from "../schema";
import { koiPrice, type ToolId } from "./tools";

export type Project = {
  tool: ToolId;
  license: string;
  engines: string;
  hosted: string;
  /** Finishes "Choose <name> …". */
  pick: string;
  summary: string;
  /** How it asks the engines, and what that needs from you. */
  collects: string;
  runs: string;
  agents: string;
};

export type OpenSourceContent = {
  heading: string;
  answer: string;
  /** Koi first, then the others. */
  projects: Project[];
  questions: Question[];
};

export const content: OpenSourceContent = {
  heading: "Open source AEO tools, compared",
  answer:
    "Seven open source projects track how AI engines mention and cite your brand: Koi, Elmo, GetCito, Limelit Open, OneGlanse, GEO/AEO Tracker and Ansvisor. The code is free to run. You pay the service that asks the engines, or use your own accounts. Elmo tracks the most engines, Limelit Open is the lightest to run, and five of the seven also sell a hosted version.",
  projects: [
    {
      tool: "koi",
      license: "MIT",
      engines: "4",
      hosted: `From ${koiPrice(15)}/mo`,
      pick: "if you build with a coding agent and want it to act on the answers.",
      summary:
        "Koi asks ChatGPT, Gemini, Google AI Mode and Google AI Overviews your buyers' questions once a day, keeps every answer with its sources, and turns any answer into a brief for the coding agent you already use. Koi Cloud runs the same code.",
      collects:
        "Through Cloro, which opens each engine in a real browser, logged out, in your brand's country. You need your own Cloro account.",
      runs: "An installer that starts Koi and Postgres with Docker Compose.",
      agents: "An MCP server and three skills: setup, brief and fix.",
    },
    {
      tool: "elmo",
      license: "MIT",
      engines: "5 sites, more by API",
      hosted: "From $29/mo",
      pick: "if you want the most engines, several checks a day, and a team on one dashboard.",
      summary:
        "Elmo is the oldest project here, public since August 2025. It tracks more engines than the others, through their websites and their APIs, and its hosted plans check up to four times a day.",
      collects:
        "ChatGPT, Google AI Mode, Google AI Overviews, Bing AI Mode and Gemini through their websites, and models such as Claude and DeepSeek through their APIs, with the provider keys you choose.",
      runs: "A Docker Compose stack managed by a CLI, with a bundled database or your own Postgres.",
      agents:
        "An MCP server, a REST API with TypeScript and Python SDKs, and an AEO skill for agents.",
    },
    {
      tool: "getcito",
      license: "MIT",
      engines: "2 self-hosted, more hosted",
      hosted: "From $599/mo",
      pick: "if you want audits, content drafts and publishing in the same tool.",
      summary:
        "GetCito tracks mentions, sentiment, citations and competitors, then turns what it finds into tasks, briefs and drafts it can publish through your integrations. Public since November 2025.",
      collects:
        "Cloro by default for the AI websites, and a Gemini key for suggestions. Its pricing page lists ChatGPT and Google AI Overviews for self-hosting.",
      runs: "A Next.js and Express app on Supabase, with Node.js 24.",
      agents: "An MCP server, a REST API, webhooks and a Looker Studio connector.",
    },
    {
      tool: "limelit",
      license: "Apache 2.0",
      engines: "7",
      hosted: "Limelit Cloud",
      pick: "if you want the smallest thing to run, on one machine.",
      summary:
        "Limelit Open is the open core of Limelit, still in pre-release. It is the lightest to run: one Go binary with SQLite, no Docker or Postgres. Sentiment, prompt drafting and several brands are in Limelit Cloud only.",
      collects:
        "ChatGPT, Claude, Perplexity and Gemini through their APIs with web search on, and Google AI Overviews, AI Mode and Bing Copilot through scrapers. You pay each provider directly.",
      runs: "One Go binary that creates its own SQLite database.",
      agents: "An MCP server with fourteen read tools, and Claude plugins with skills.",
    },
    {
      tool: "oneglanse",
      license: "MIT",
      engines: "5",
      hosted: "None",
      pick: "if you want answers from your own accounts on the real websites.",
      summary:
        "OneGlanse runs your prompts on the AI websites with your own signed-in accounts, in a modified Firefox, instead of APIs or a scraping service. A model you connect then scores each answer.",
      collects:
        "Your own accounts on ChatGPT, Perplexity, Gemini, Claude and Google AI Overviews, plus an OpenAI or Anthropic key for the scoring. On a server it needs a residential proxy.",
      runs: "Node.js, pnpm and Docker, on your computer or your own server.",
      agents: "The README does not mention an MCP server.",
    },
    {
      tool: "geoAeoTracker",
      license: "MIT",
      engines: "4",
      hosted: "None",
      pick: "if you want a quick check without a database, or a CLI for CI.",
      summary:
        "GEO/AEO Tracker keeps results in your browser, or in your own Supabase project. Besides tracking, it audits a page for AI search and drafts competitor battlecards.",
      collects:
        "Bright Data's AI Scraper API for ChatGPT, Perplexity, Gemini and Google AI Mode, billed per call.",
      runs: "A Next.js app, on your computer or deployed to Vercel.",
      agents: "An MCP server and a command line tool.",
    },
    {
      tool: "ansvisor",
      license: "MIT",
      engines: "6, more coming",
      hosted: "Ansvisor Cloud",
      pick: "if you want agents inside the tool to draft content for your team to approve.",
      summary:
        "Ansvisor goes past tracking: its agents find opportunities and draft content fixes, which you approve before anything goes live. It also connects AI visibility with traffic and search data.",
      collects:
        "Cloro or Anakin for ChatGPT, Google AI Overviews, Google AI Mode, Gemini, Claude and Grok, plus an OpenAI, Gemini or Anthropic key.",
      runs: "A Next.js and Express app on Supabase, or Docker Compose.",
      agents: "An MCP server that can also create content drafts and actions.",
    },
  ],
  questions: [
    {
      q: "What is the best open source AEO tool?",
      a: "It depends on what you need. Elmo tracks the most engines and checks most often. Limelit Open is a single binary. OneGlanse uses your own accounts on the real websites. Koi turns answers into work for your coding agent.",
    },
    {
      q: "Are open source AEO tools free?",
      a: "The code is free. Each one needs a way to ask the engines: a provider you pay per answer or per call, or, for OneGlanse, your own accounts on each website plus a model key.",
    },
    {
      q: "Which open source AEO tools have an MCP server?",
      a: "Koi, Elmo, GetCito, Limelit Open, GEO/AEO Tracker and Ansvisor. The OneGlanse README does not mention one.",
    },
    {
      q: "Which ones have a hosted version?",
      a: "Koi, Elmo, GetCito, Limelit and Ansvisor each sell one. OneGlanse and GEO/AEO Tracker have none: you run them yourself.",
    },
  ],
};
