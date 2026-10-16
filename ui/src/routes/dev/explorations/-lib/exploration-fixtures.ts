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

export const RECORDED_PROMPT = "best zsh plugin for inline predictive command autosuggestions";
const RECORDED_RESPONSES: Record<
  number,
  Partial<
    Record<
      VisibilityEngine,
      { outcome: "cited" | "miss" | "error"; rawAnswer: string; sources: string[] }
    >
  >
> = {
  84: {
    chatgpt: {
      outcome: "cited",
      rawAnswer:
        "If you mean inline “ghost text” suggestions as you type, my pick is zsh-autosuggestions.\n\nBest default: zsh-autosuggestions — mature, fast, ~36k GitHub stars, and specifically designed for Fish-like inline suggestions based on your history/completion system.\n\nMore powerful completion UI: zsh-autocomplete — automatically shows actual completion candidates while typing, rather than just one ghost-text prediction.\n\nInteresting newer option: Deja — tries to improve on traditional history-prefix matching with fuzzy matching, directory awareness, command-sequence prediction, and frecency.",
      sources: [
        "https://github.com/zsh-users/zsh-autosuggestions",
        "https://github.com/marlonrichert/zsh-autocomplete/blob/main/README.md",
        "https://github.com/Giammarco-Ferranti/deja",
        "https://github.com/zsh-users/zsh-autosuggestions/blob/master/INSTALL.md",
      ],
    },
    gemini: {
      outcome: "miss",
      rawAnswer:
        "When it comes to inline predictive command autosuggestions (the “ghost text” style popularized by Fish shell), there is a clear industry standard, alongside a couple of popular alternatives depending on what kind of prediction you want.\n\nThe Undisputed Gold Standard: zsh-autosuggestions\n\nIt draws Fish-like ghost text inline as you type, predicting commands based on your history and completion items.",
      sources: [
        "https://github.com/zsh-users/zsh-autosuggestions",
        "https://catalins.tech/zsh-plugins/#:~:text=It's%20a%20handy%20plugin%20that%20helps%20you,and%20restart%20Zsh%20as%20follows%3A%20source%20~%2F.zshrc.",
        "https://www.reddit.com/r/linux4noobs/comments/1o9rbem/i_remember_there_was_a_very_useful_autocomplete/#:~:text=*%20zsh%2Dautosuggestions%20felt%20too%20dumb%2C%20so%20I,characters%20when%20using%20zsh%20autocomplete%20and%20autosuggestions.",
        "https://www.reddit.com/r/commandline/comments/1sv5ivz/zshsage_a_smarter_autosuggestions_that_learn_from/#:~:text=So%20I%20built%20zsh%2Dsage.%20*%20The%20autosuggestion,sure%2C%20faint%20grey%20when%20it's%20a%20stretch.",
        "https://www.reddit.com/r/commandline/comments/1sv5ivz/zshsage_a_smarter_autosuggestions_that_learn_from/#:~:text=Long%20prefix%20like%20git%20commit%20%2Dm%20%22f,totally%20different%20UX%20and%20works%20alongside%20sage.",
        "https://www.reddit.com/r/commandline/comments/1sv5ivz/zshsage_a_smarter_autosuggestions_that_learn_from/#:~:text=*%20%5BOC%5D%20Deja%20v0.4.2%20%2D%20smarter%20zsh,about%20attendance%3F%2031%20upvotes%20%C2%B7%2022%20comments.",
        "https://tiagodanin.com/post/the-best-autocomplete-plugin-for-zsh/#:~:text=With%20Homebrew%2C%20install%20the%20package%20and%20source,needs%20to%20find%20them%20already%20in%20place.",
      ],
    },
    google_ai_overviews: { outcome: "miss", rawAnswer: "", sources: [] },
    perplexity: {
      outcome: "miss",
      rawAnswer:
        "zsh-autosuggestions is a solid default: it displays muted inline suggestions as you type, based on your command history and completions. Press → or End to accept one.",
      sources: [
        "https://github.com/zsh-users/zsh-autosuggestions",
        "https://github.com/zsh-users/zsh-autosuggestions/blob/master/README.md",
        "https://www.kali.org/tools/zsh-autosuggestions/",
        "https://best-of-web.builder.io/library/zsh-users/zsh-autosuggestions",
        "https://troglodyteconsulting.com/terminal/autosuggestions/",
        "https://github.com/numbnet/zsh-autosuggestions",
        "https://www.reddit.com/r/zsh/comments/13tv3if/question_what_are_the_best_plugins_for_zsh/",
        "https://danielkorn.io/post/zsh-autosuggestions/",
        "https://www.reddit.com/r/zsh/comments/1v8rq7s/an_alternative_to_zsh_autosuggestion_and/",
        "https://github.com/gstax/zsh-autosuggestions",
      ],
    },
  },
  85: {
    chatgpt: {
      outcome: "cited",
      rawAnswer:
        "For inline predictive command autosuggestions in Zsh, my default recommendation is zsh-autosuggestions. But if you want smarter predictions rather than just matching your command history, there’s a newer alternative worth considering.\n\nTop picks\n\nDisplays a muted gray continuation of your command as you type, based on shell history or completion results. Lightweight, mature, and easy to configure.\n\nUses directory awareness, fuzzy matching, and command-sequence prediction to improve on simple history-prefix matching. It requires a separate background daemon.\n\nBest if you want a richer, continuously updating completion interface with selectable candidates, rather than only inline ghost text.",
      sources: [
        "https://github.com/zsh-users/zsh-autosuggestions",
        "https://github.com/zsh-users/zsh-autos",
        "https://github.com/Giammarco-Ferranti/deja",
        "https://ds31x.tistory.com/81",
        "https://formulae.brew.sh/formula/zsh-autocomplete",
        "https://github.com/bd-naveent/zsh-autosuggestions",
        "https://github.com/marlonrichert/zsh-autocomplete",
        "https://github.com/zsh-users/zsh-history-substring-search/issues/17",
        "https://stackoverflow.com/questions/13613698/zsh-history-completion-menu",
        "https://formulae.brew.sh/formula/zsh-autosuggestions",
        "https://explore.market.dev/ecosystems/zsh/projects/zsh-you-should-use",
        "https://formulae.brew.sh/formula/zsh-you-should-use",
        "https://github.com/ohmyzsh/ohmyzsh/wiki/plugins",
        "https://github.locker/zsh-users/zsh-autosuggestions",
        "https://linuxhaxor.net/code/setup-configure-autocomplete-zsh.html",
        "https://qgithub.com/MichaelAquilina/zsh-you-should-use",
        "https://sourceforge.net/projects/zsh-autosuggestions.mirror/",
        "https://www.reddit.com/r/zsh/comments/v08lsx",
      ],
    },
    gemini: { outcome: "error", rawAnswer: "", sources: [] },
    google_ai_overviews: { outcome: "error", rawAnswer: "", sources: [] },
    perplexity: { outcome: "error", rawAnswer: "", sources: [] },
  },
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
  const recorded =
    brand === "deja" && position === 0 && answer.prompt === RECORDED_PROMPT
      ? RECORDED_RESPONSES[answer.day]?.[answer.engine as VisibilityEngine]
      : undefined;
  const sourceExample = brand === "deja" && answer.day === 89 && position === 1;
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
  const recordedRunAt = !recorded
    ? undefined
    : answer.day === 84
      ? "2026-10-08T13:06:09.739765+02:00"
      : answer.day === 85
        ? "2026-10-09T17:57:55.300907+02:00"
        : undefined;

  return {
    id: answer.id,
    runId: recorded ? `recorded:deja:${answer.date}` : `${brand}:${answer.day}`,
    promptId: `${brand}:prompt:${position}`,
    promptText: answer.prompt,
    promptPosition: position,
    engine: ENGINES.find((engine) => engine.id === answer.engine)?.id ?? "chatgpt",
    collector: recorded ? "cloro" : "stub",
    fetchedAt: recordedRunAt ? new Date(recordedRunAt) : new Date(`${answer.date}T12:00:00Z`),
    outcome:
      recorded?.outcome ??
      (sourceExample && answer.engine === "chatgpt"
        ? "cited"
        : answer.mentions.includes(definition.name)
          ? "mentioned"
          : "miss"),
    rawAnswer:
      recorded?.rawAnswer ??
      demoAnswer ??
      (answer.mentions.length
        ? `This simulated answer recommends investigating ${answer.mentions.join(" and ")}.`
        : "This simulated answer does not mention any tracked brand."),
    sources: recorded?.sources ?? sources,
    errorCode: recorded?.outcome === "error" ? "collection_failed" : null,
    appearances: recorded
      ? recorded.rawAnswer.includes("zsh-autosuggestions")
        ? [
            {
              competitorId: "deja:competitor:0",
              name: "zsh-autosuggestions",
              confirmed: true,
              evidence: [],
            },
          ]
        : []
      : answer.mentions
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
