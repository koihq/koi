import type { Comparison } from "./comparison";
import { koiPrice, type RivalId } from "./tools";

type Alternative = {
  path: string;
  rival: RivalId;
  name: string;
  title: string;
  description: string;
  summary: string;
  updated: string;
  content: () => Promise<Comparison>;
};

/**
 * The comparison pages, one rival each. Each one's text loads on demand, so a page never downloads
 * the others. `updated` changes with every edit to a page's text or facts: it dates the page and
 * the sitemap. `summary` is the rival in one line, for the list of comparisons.
 */
export const alternatives: Alternative[] = [
  {
    path: "/alternatives/profound",
    rival: "profound",
    name: "Koi vs Profound",
    title: "Profound alternative: Koi, open source with a public price",
    description: `Koi is an open source Profound alternative with a public price, from ${koiPrice(15)} a month. Engines, MCP and team features side by side, with where Profound is ahead.`,
    summary: "Enterprise plans at a custom price, with up to ten engines, SSO and unlimited seats.",
    updated: "2026-11-26",
    content: async () => (await import("./profound")).content,
  },
  {
    path: "/alternatives/peec-ai",
    rival: "peec",
    name: "Koi vs Peec AI",
    title: "Peec AI alternative: Koi, open source from €29 a month",
    description: `Koi is an open source Peec AI alternative from ${koiPrice(15)} a month. Prices for the same prompts, engines, MCP and team features, with where Peec AI is ahead.`,
    summary:
      "From $95 a month for 50 prompts on three engines, with unlimited users and sentiment.",
    updated: "2026-11-26",
    content: async () => (await import("./peec")).content,
  },
  {
    path: "/alternatives/otterly",
    rival: "otterly",
    name: "Koi vs Otterly.AI",
    title: "Otterly.AI alternative: Koi, open source, MCP on every plan",
    description: `Koi is an open source Otterly.AI alternative with MCP on every plan, from ${koiPrice(15)} a month. Prices, engines and features, with where Otterly.AI is ahead.`,
    summary: "From €29 a month for 16 prompts before tax, with the MCP server and API from €189.",
    updated: "2026-11-26",
    content: async () => (await import("./otterly")).content,
  },
  {
    path: "/alternatives/elmo",
    rival: "elmo",
    name: "Koi vs Elmo",
    title: "Elmo alternative: Koi, with unlimited brands",
    description: `Koi and Elmo are both open source. Koi Cloud has unlimited brands and four engines from ${koiPrice(15)} a month. Prices, engines and features, with where Elmo is ahead.`,
    summary:
      "Open source like Koi, with more engines and checks a day, and one to five brands on Cloud.",
    updated: "2026-11-26",
    content: async () => (await import("./elmo")).content,
  },
];

/** Every open source project on one page, the format AI answers cite most. */
export const openSourcePage = {
  path: "/open-source-aeo-tools",
  name: "Open source AEO tools",
  title: "Open source AEO and AI visibility tools, compared",
  description:
    "Seven open source AEO tools side by side: Koi, Elmo, GetCito, Limelit Open, OneGlanse, GEO/AEO Tracker, Ansvisor. License, engines and what you run.",
  summary:
    "Koi, Elmo, GetCito, Limelit Open, OneGlanse, GEO/AEO Tracker and Ansvisor: license, engines and what you run.",
  updated: "2026-11-26",
  content: async () => (await import("./open-source")).content,
};

/** The index of every comparison. */
export const comparisonsPage = {
  path: "/alternatives",
  name: "All comparisons",
  title: "Koi compared with other AI visibility tools",
  description:
    "Koi next to Profound, Peec AI, Otterly.AI and Elmo, one tool per page, plus every open source AEO tool. Prices and features checked on each company's own pages.",
  updated: "2026-11-26",
};

/** What the footer's Compare column links to. */
export const comparePages = [...alternatives, openSourcePage, comparisonsPage];
