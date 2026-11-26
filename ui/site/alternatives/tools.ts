import catalog from "../../../pkg/pricing/catalog.json";

export type Tool = {
  name: string;
  domain: string;
  /** The ISO date its facts were last read on the pages in `sources`. */
  checked: string;
  sources: { label: string; url: string }[];
};

export function koiPrice(prompts: number) {
  const option = catalog.options.find((candidate) => candidate.prompts === prompts);

  if (!option) throw new Error(`No Koi plan with ${prompts} prompts`);

  return `€${option.euros}`;
}

/**
 * Where each page's facts come from. Read them on the vendor's own pages, date them, and keep the
 * vendor's currency.
 */
export const tools = {
  koi: {
    name: "Koi",
    domain: "getkoi.co",
    checked: "2026-11-26",
    sources: [
      { label: "pricing", url: "https://getkoi.co/#pricing" },
      { label: "source code", url: "https://github.com/koihq/koi" },
    ],
  },
  profound: {
    name: "Profound",
    domain: "tryprofound.com",
    checked: "2026-11-26",
    sources: [
      { label: "pricing", url: "https://www.tryprofound.com/pricing" },
      { label: "MCP server", url: "https://www.tryprofound.com/features/mcp" },
    ],
  },
  peec: {
    name: "Peec AI",
    domain: "peec.ai",
    checked: "2026-11-26",
    sources: [{ label: "pricing", url: "https://peec.ai/pricing" }],
  },
  otterly: {
    name: "Otterly.AI",
    domain: "otterly.ai",
    checked: "2026-11-26",
    sources: [{ label: "pricing", url: "https://otterly.ai/pricing" }],
  },
  elmo: {
    name: "Elmo",
    domain: "elmohq.com",
    checked: "2026-11-26",
    sources: [
      { label: "pricing", url: "https://www.elmohq.com/pricing" },
      { label: "source code", url: "https://github.com/elmohq/elmo" },
    ],
  },
  limelit: {
    name: "Limelit Open",
    domain: "limelit.co",
    checked: "2026-11-26",
    sources: [{ label: "source code", url: "https://github.com/limelit-co/open" }],
  },
  oneglanse: {
    name: "OneGlanse",
    domain: "oneglanse.com",
    checked: "2026-11-26",
    sources: [{ label: "source code", url: "https://github.com/oneglanse/oneglanse" }],
  },
  geoAeoTracker: {
    name: "GEO/AEO Tracker",
    domain: "github.com/danishashko/geo-aeo-tracker",
    checked: "2026-11-26",
    sources: [{ label: "source code", url: "https://github.com/danishashko/geo-aeo-tracker" }],
  },
  getcito: {
    name: "GetCito",
    domain: "getcito.com",
    checked: "2026-11-26",
    sources: [
      {
        label: "source code",
        url: "https://github.com/ai-search-guru/getcito-worlds-first-open-source-aio-aeo-or-geo-tool",
      },
      { label: "pricing", url: "https://getcito.com/pricing" },
    ],
  },
  ansvisor: {
    name: "Ansvisor",
    domain: "ansvisor.com",
    checked: "2026-11-26",
    sources: [{ label: "source code", url: "https://github.com/ansvisor/ansvisor" }],
  },
} satisfies Record<string, Tool>;

export type ToolId = keyof typeof tools;

export type RivalId = Exclude<keyof typeof tools, "koi">;
