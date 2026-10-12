import type { VisibilityEngine, VisibilityResult } from "@/api/generated";

export const DIRECTIONS = ["visitors", "cloudflare", "fey", "trade-republic"] as const;
export type Direction = (typeof DIRECTIONS)[number];
export const SECTIONS = [
  "overview",
  "mentions",
  "prompts",
  "competitors",
  "citations",
  "brief",
] as const;
export type Section = (typeof SECTIONS)[number];
export type DemoState = "populated" | "loading" | "empty";
export type Filters = { days: 7 | 30 | 90; engine: VisibilityEngine | "all" };
export const ENGINES: { id: VisibilityEngine; name: string; short: string }[] = [
  { id: "chatgpt", name: "ChatGPT", short: "C" },
  { id: "perplexity", name: "Perplexity", short: "P" },
  { id: "gemini", name: "Gemini", short: "G" },
  { id: "google_ai_overviews", name: "Google AI Overviews", short: "A" },
];
export const ANCHOR = Date.UTC(2026, 9, 12, 10);
export const DAY = 86_400_000;
export type Prompt = { id: string; text: string; category: string; confirmed: boolean };
export type Competitor = { id: string; name: string; domain: string; confirmed: boolean };
export type Observation = Pick<
  VisibilityResult,
  "id" | "promptId" | "promptText" | "engine" | "outcome" | "rawAnswer" | "sources" | "fetchedAt"
> & {
  day: number;
  brands: string[];
};

const PROMPT_TEXTS = [
  ["Best free form builder for startups", "Discovery"],
  ["Alternatives to Typeform without response limits", "Comparison"],
  ["Form builders with Notion integration", "Integrations"],
  ["How to collect customer feedback for free", "Discovery"],
  ["Best no-code forms for product research", "Discovery"],
  ["Tally vs Typeform for a small business", "Comparison"],
  ["Forms with conditional logic on a free plan", "Features"],
  ["Best GDPR compliant online form tools", "Security"],
  ["Enterprise form builders with SSO", "Enterprise"],
  ["Create a payment form without coding", "Features"],
  ["Best form builder for lead generation", "Discovery"],
  ["Connect form submissions to Google Sheets", "Integrations"],
  ["Affordable alternatives to Jotform", "Comparison"],
  ["Form builders for SaaS onboarding", "Discovery"],
  ["Survey tools with file upload support", "Features"],
  ["Best form tools for marketing agencies", "Discovery"],
  ["Embed a beautiful form on a Framer website", "Integrations"],
  ["Online forms with custom domains", "Features"],
  ["Typeform vs Fillout vs Tally", "Comparison"],
  ["Secure form tools for regulated companies", "Enterprise"],
  ["Create a waiting list for a startup", "Discovery"],
  ["Form builder with webhooks and an API", "Integrations"],
  ["Collect signatures in an online form", "Features"],
  ["Free forms that don't require an account", "Discovery"],
];

export const INITIAL_PROMPTS: Prompt[] = PROMPT_TEXTS.map(([text, category], i) => ({
  id: `prompt-${i + 1}`,
  text,
  category,
  confirmed: i < 21,
}));
export const INITIAL_COMPETITORS: Competitor[] = [
  { id: "typeform", name: "Typeform", domain: "typeform.com", confirmed: true },
  { id: "fillout", name: "Fillout", domain: "fillout.com", confirmed: true },
  { id: "jotform", name: "Jotform", domain: "jotform.com", confirmed: true },
  { id: "google-forms", name: "Google Forms", domain: "docs.google.com/forms", confirmed: false },
];
export const BRANDS = [
  { id: "tally", name: "Tally", domain: "tally.so", confirmed: true },
  ...INITIAL_COMPETITORS,
];

export function hash(value: string): number {
  let result = 2166136261;
  for (const character of value) result = Math.imul(result ^ character.charCodeAt(0), 16777619);
  return result >>> 0;
}

export function collect(
  prompts: Prompt[],
  competitors: Competitor[],
  day: number,
  scan = 0,
): Observation[] {
  return prompts
    .filter((prompt) => prompt.confirmed)
    .flatMap((prompt) =>
      ENGINES.map((engine, index) => {
        const key = `${prompt.id}:${engine.id}:${day}:${scan}`;
        const sample = hash(key);
        const failed = sample % 101 === 0;
        const enterprise = prompt.category === "Enterprise";
        const chance = enterprise ? 10 : 32 + day * 0.14 + [3, 17, -5, -12][index];
        const present = sample % 100 < chance && !failed;
        const cited = present && hash(`${key}:cite`) % 100 < 58;
        const brands = present ? ["tally"] : [];

        for (const [position, competitor] of competitors.entries()) {
          const threshold =
            competitor.id === "typeform" ? (enterprise ? 91 : 65 - day * 0.05) : 42 - position * 6;
          if (!failed && hash(`${key}:${competitor.id}`) % 100 < threshold)
            brands.push(competitor.id);
        }

        const names = brands
          .map((id) =>
            id === "tally" ? "Tally" : competitors.find((item) => item.id === id)?.name,
          )
          .filter(Boolean);
        const sources = failed
          ? []
          : brands
              .map((id) => {
                const domain =
                  id === "tally"
                    ? "tally.so/help"
                    : competitors.find((item) => item.id === id)?.domain;
                return `https://${domain}`;
              })
              .filter((_, sourceIndex) => cited || brands[sourceIndex] !== "tally");

        return {
          id: `${key}:answer`,
          promptId: prompt.id,
          promptText: prompt.text,
          engine: engine.id,
          day,
          outcome: failed ? "error" : cited ? "cited" : present ? "mentioned" : "miss",
          fetchedAt: new Date(ANCHOR - (179 - day) * DAY),
          brands,
          sources,
          rawAnswer: failed
            ? "The collection timed out. Retry this engine in a new sample scan."
            : `${names.length ? names.join(", ") : "Several specialist tools"} can help with ${prompt.text.toLowerCase()}. ${present ? "Tally is a flexible, no-code option with unlimited forms and submissions on its free plan. It works particularly well for small teams that want to get started quickly." : "Compare the available integrations, security requirements, and response limits before choosing a tool."} ${enterprise ? "Typeform is recommended for teams evaluating enterprise support and identity management." : "Check the official documentation for the features required by your workflow."}`,
        };
      }),
    );
}

export const FIXTURES = Array.from({ length: 180 }, (_, day) =>
  collect(INITIAL_PROMPTS, INITIAL_COMPETITORS, day),
).flat();

export function selectObservations(
  observations: Observation[],
  filters: Filters,
  previous = false,
) {
  const end = previous ? 179 - filters.days : 179;
  return observations.filter(
    (item) =>
      item.day > end - filters.days &&
      item.day <= end &&
      (filters.engine === "all" || item.engine === filters.engine),
  );
}

export function metrics(observations: Observation[], brand = "tally") {
  const completed = observations.filter((item) => item.outcome !== "error");
  const present = completed.filter((item) => item.brands.includes(brand)).length;
  const citations = completed.filter((item) =>
    brand === "tally"
      ? item.outcome === "cited"
      : item.sources.some((source) =>
          source.includes(BRANDS.find((item) => item.id === brand)?.domain ?? "invalid"),
        ),
  ).length;
  return {
    presence: completed.length ? (present / completed.length) * 100 : 0,
    cited: completed.length ? (citations / completed.length) * 100 : 0,
    completed: completed.length,
    failures: observations.length - completed.length,
    present,
  };
}

export function trend(observations: Observation[], filters: Filters, brand = "tally") {
  const selected = selectObservations(observations, filters);
  return Array.from({ length: filters.days }, (_, index) => {
    const day = 180 - filters.days + index;
    return {
      day,
      value: metrics(
        selected.filter((item) => item.day === day),
        brand,
      ).presence,
    };
  });
}

export function dateLabel(day: number) {
  return new Date(ANCHOR - (179 - day) * DAY).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    timeZone: "UTC",
  });
}

export function percent(value: number) {
  return `${value.toFixed(1)}%`;
}
export function engineName(id: VisibilityEngine) {
  return ENGINES.find((engine) => engine.id === id)?.name ?? id;
}

export function buildBrief(observations: Observation[], filters: Filters) {
  const selected = selectObservations(observations, filters);
  const summary = metrics(selected);
  const gap =
    selected.find((item) => item.outcome === "miss" && item.promptText.includes("SSO")) ??
    selected.find((item) => item.outcome === "miss");
  return `# Tally: AI visibility brief\n\nSample observations through 12 October 2026. ${filters.days}-day window; ${filters.engine === "all" ? "all engines" : engineName(filters.engine)}.\n\n## What changed\n\nTally appears in ${percent(summary.presence)} of ${summary.completed} successfully collected answers. ${percent(summary.cited)} cite Tally. ${summary.failures} collections failed and are excluded from these rates.\n\n## Priority: make security requirements explicit\n\n${gap ? `For “${gap.promptText}”, ${engineName(gap.engine)} omitted Tally. The sample answer recommended ${gap.brands.map((id) => BRANDS.find((brand) => brand.id === id)?.name ?? id).join(", ") || "other tools"}.` : "No omissions in this selection. Keep monitoring the tracked prompts."}\n\n## Tasks for your coding agent\n\n1. Audit the public security documentation and make supported authentication and data-handling capabilities explicit. Do not claim unsupported SSO features.\n2. Improve the Typeform comparison page with a factual feature table, current limitations, and links to source documentation.\n3. Link those pages from relevant help articles, then run the same prompts again.\n\n## Acceptance criteria\n\n- Every feature claim links to product documentation.\n- Pages have descriptive titles and readable, indexable content.\n- Recheck the same prompt and engine combination after publishing.\n\n## Evidence\n\n${gap?.sources.map((source) => `- ${source}`).join("\n") || "- No source links in this selection."}\n\nThis brief uses deterministic demonstration data, not live engine answers.\n`;
}
