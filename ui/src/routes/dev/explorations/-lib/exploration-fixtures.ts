import type { Brand } from "@/api/brands.api";
import type { Competitor } from "@/api/competitors.api";
import type { VisibilityEngine } from "@/api/generated";
import type { Prompt } from "@/api/prompts.api";
import type { VisibilityResult } from "@/api/runs.api";
import type { ApiListResponse } from "@/api/shared/types";
import { ENGINES } from "./data";
import {
  HOME_BRANDS,
  HOME_LOGOS,
  type HomeAnswer,
  type HomeBrand,
  homeEvidence,
} from "./home-evidence";

export const VIEWS = ["visibility", "mentions", "prompts", "competitors", "settings"] as const;
export type ExplorationView = (typeof VIEWS)[number];
export const STATES = [
  "empty",
  "review",
  "loading",
  "error",
  "partial",
  "running",
  "optional-off",
  "save-error",
  "draft-failed",
  "draft-empty",
  "draft-queued",
  "draft-running",
] as const;
export type ExplorationState = (typeof STATES)[number];

export type ExplorationSearch = {
  view?: ExplorationView;
  brand?: HomeBrand;
  state?: ExplorationState;
  days?: 7 | 30 | 90;
  engines?: VisibilityEngine[];
  promptId?: string;
  compareId?: string;
  q?: string;
  status?: "all" | "confirmed" | "unconfirmed" | "past";
  sort?: "original" | "prompt" | "rate" | "change";
  descending?: boolean;
  page?: number;
  pageSize?: 10 | 20 | 50;
  responseEngine?: VisibilityEngine;
  runId?: string;
};

export function parseExplorationSearch(search: Record<string, unknown>): ExplorationSearch {
  const engine = (value: unknown) => ENGINES.find((item) => item.id === value)?.id;
  const id = (value: unknown) =>
    typeof value === "string" && value.length <= 200 ? value : undefined;
  const days = Number(search.days);
  const pageSize = Number(search.pageSize);
  const page = Number(search.page);
  const engines = ENGINES.filter(
    (item) => Array.isArray(search.engines) && search.engines.includes(item.id),
  ).map((item) => item.id);

  return {
    view: VIEWS.find((value) => value === search.view),
    brand: (Object.keys(HOME_BRANDS) as HomeBrand[]).find((value) => value === search.brand),
    state: STATES.find((value) => value === search.state),
    days: days === 7 || days === 30 || days === 90 ? days : undefined,
    engines: engines.length ? engines : undefined,
    promptId: id(search.promptId),
    compareId: id(search.compareId),
    q: typeof search.q === "string" ? search.q.slice(0, 500) : undefined,
    status: (["all", "confirmed", "unconfirmed", "past"] as const).find(
      (value) => value === search.status,
    ),
    sort: (["original", "prompt", "rate", "change"] as const).find(
      (value) => value === search.sort,
    ),
    descending: search.descending === true || search.descending === "true" ? true : undefined,
    page: Number.isSafeInteger(page) && page > 0 ? page : undefined,
    pageSize: pageSize === 10 || pageSize === 20 || pageSize === 50 ? pageSize : undefined,
    responseEngine: engine(search.responseEngine),
    runId: id(search.runId),
  };
}

export function pageOf<T>(items: T[], page: number, pageSize: number): ApiListResponse<T> {
  const totalPages = Math.ceil(items.length / pageSize);
  const currentPage = Math.max(1, Math.min(page, totalPages || 1));

  return {
    items: items.slice((currentPage - 1) * pageSize, currentPage * pageSize),
    totalItems: items.length,
    totalPages,
    currentPage,
    hasNext: currentPage < totalPages,
    hasPrev: currentPage > 1,
  };
}

export type PreviewBrand = {
  settings: Brand;
  prompts: Prompt[];
  competitors: Competitor[];
  reviewed: { prompts: boolean; competitors: boolean };
};

const REPOSITORIES: Record<string, string> = {
  Deja: "https://github.com/Giammarco-Ferranti/deja",
  "zsh-autosuggestions": "https://github.com/zsh-users/zsh-autosuggestions",
};

export const SUGGESTIONS: Record<HomeBrand, { name: string; domain: string }> = {
  deja: { name: "fish", domain: "fishshell.com" },
  koi: { name: "Otterly", domain: "otterly.ai" },
  mintlify: { name: "Docusaurus", domain: "docusaurus.io" },
  beehiiv: { name: "Ghost", domain: "ghost.org" },
  tally: { name: "Jotform", domain: "jotform.com" },
  posthog: { name: "Plausible", domain: "plausible.io" },
};

export function createPreviewBrand(brand: HomeBrand, state?: ExplorationState): PreviewBrand {
  const definition = HOME_BRANDS[brand];
  const prompts = Object.values(definition.topics).flat();
  const empty = state === "empty";
  const reviewed = state !== "review" && !empty;

  return {
    settings: {
      id: brand,
      userId: "preview",
      name: definition.name,
      domain: HOME_LOGOS[definition.name]?.domain ?? "",
      githubRepo: REPOSITORIES[definition.name] ?? "",
      country: "US",
      description: "",
    },
    prompts: empty
      ? []
      : [
          ...prompts.map((text, index) => ({
            id: `${brand}:prompt:${index}`,
            brandId: brand,
            text,
            source: "manual" as const,
            confirmed: true,
          })),
          ...(brand === "beehiiv"
            ? [
                "How should I compare newsletter platforms for a local publication?",
                "Which newsletter tool supports a private member community?",
                "Can I import subscribers and keep their existing paid plans?",
                "What should a creator measure after sending their first ten issues?",
              ].map((text, index) => ({
                id: `${brand}:draft:${index}`,
                brandId: brand,
                text,
                source: "drafted" as const,
                confirmed: false,
              }))
            : []),
        ],
    competitors: empty
      ? []
      : [
          ...definition.rivals.map(
            (name, index): Competitor => ({
              id: `${brand}:competitor:${index}`,
              brandId: brand,
              name,
              domain: HOME_LOGOS[name]?.domain ?? "",
              githubRepo: REPOSITORIES[name] ?? "",
              aliases: [],
              additionalUrls: [],
              source: "manual",
              confirmed: true,
            }),
          ),
          {
            id: `${brand}:observed`,
            brandId: brand,
            name: "Unresolved alternative",
            domain: "",
            githubRepo: "",
            aliases: [],
            additionalUrls: [],
            source: "observed",
            confirmed: false,
          },
        ],
    reviewed: { prompts: reviewed, competitors: reviewed },
  };
}

export function finishReview(data: PreviewBrand, kind: "prompts" | "competitors"): PreviewBrand {
  return {
    ...data,
    reviewed: { ...data.reviewed, [kind]: true },
    [kind]: data[kind].map((item) => ({
      ...item,
      confirmed: item.source === "observed" ? item.confirmed : true,
    })),
  };
}

export function confirmItem(
  data: PreviewBrand,
  kind: "prompts" | "competitors",
  id: string,
): PreviewBrand {
  return {
    ...data,
    [kind]: data[kind].map((item) =>
      item.id === id && (!("domain" in item) || item.domain || item.githubRepo)
        ? { ...item, confirmed: true }
        : item,
    ),
  };
}

// Adapt the existing evidence without changing its names, dates, or ordering.
// The latest Deja example includes clearly labeled, illustrative sources for the reader study.
export function answerResult(answer: HomeAnswer, brand: HomeBrand): VisibilityResult {
  const definition = HOME_BRANDS[brand];
  const prompts = Object.values(definition.topics).flat();
  const position = prompts.indexOf(answer.prompt);
  const sourceExample = brand === "deja" && answer.day === 89 && position === 0;
  const sources = sourceExample
    ? answer.engine === "chatgpt"
      ? [
          "https://github.com/Giammarco-Ferranti/deja",
          "https://github.com/zsh-users/zsh-autosuggestions",
        ]
      : answer.engine === "perplexity"
        ? ["https://github.com/zsh-users/zsh-autosuggestions"]
        : []
    : [];
  const demoAnswer =
    sourceExample && answer.engine === "chatgpt"
      ? "Deja is an option for zsh users who want suggestions drawn from command history. It ranks fuzzy matches using frequency, recency, the current directory, and learned command sequences.\n\nUnlike a generative assistant, it does not create commands or execute a suggestion automatically. That makes it worth considering when you want help recalling commands you have already used.\n\nzsh-autosuggestions remains a familiar alternative for inline suggestions. Compare each tool's setup and behavior in your own shell workflow before switching."
      : null;

  return {
    id: answer.id,
    runId: `${brand}:${answer.day}`,
    promptId: `${brand}:prompt:${position}`,
    promptText: answer.prompt,
    promptPosition: position,
    engine: ENGINES.find((engine) => engine.id === answer.engine)?.id ?? "chatgpt",
    collector: "stub",
    fetchedAt: new Date(`${answer.date}T12:00:00Z`),
    outcome:
      sourceExample && answer.engine === "chatgpt"
        ? "cited"
        : answer.mentions.includes(definition.name)
          ? "mentioned"
          : "miss",
    rawAnswer:
      demoAnswer ??
      (answer.mentions.length
        ? `This simulated answer recommends investigating ${answer.mentions.join(" and ")}.`
        : "This simulated answer does not mention any tracked brand."),
    sources,
    errorCode: null,
    appearances: answer.mentions
      .filter((name) => name !== definition.name)
      .map((name) => ({
        competitorId: `${brand}:competitor:${definition.rivals.indexOf(name)}`,
        name,
        confirmed: true,
        evidence: [],
      })),
  };
}

export function collectionAnswers(
  brand: HomeBrand,
  day: number,
  state?: ExplorationState,
): HomeAnswer[] {
  if (state === "empty" || state === "review") return [];

  return homeEvidence(brand)
    .filter((answer) => answer.day === day)
    .reverse();
}

export function resultForState(
  answer: HomeAnswer,
  brand: HomeBrand,
  index: number,
  state?: ExplorationState,
): VisibilityResult {
  const result = answerResult(answer, brand);
  if (state !== "running" && state !== "partial") return result;

  if (index % 3 === 0)
    return {
      ...result,
      outcome: null,
      rawAnswer: "",
      fetchedAt: null,
      appearances: [],
      sources: [],
    };
  if (index % 3 === 1)
    return {
      ...result,
      outcome: "error",
      rawAnswer: "",
      sources: [],
      errorCode: "collection_failed",
      appearances: [],
    };

  return result;
}
