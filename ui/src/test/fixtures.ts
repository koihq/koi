import type {
  BillingCatalog,
  BillingStatus,
  BillingSubscription,
  BrandRes,
  CapabilitiesResponse,
  CompetitorRes,
  CompetitorVisibility,
  Credential,
  GenerationStatus,
  MeRes,
  PromptRes,
  TrackingStatus,
  VisibilityDay,
  VisibilityMetric,
  VisibilityReport,
  VisibilityResult,
  VisibilityRunStatus,
} from "@/api/generated";

/** The JSON a generated model arrives as: dates are ISO strings on the wire. */
export type Wire<T> = T extends Date
  ? string
  : T extends Array<infer Item>
    ? Array<Wire<Item>>
    : T extends object
      ? { [Key in keyof T]: Wire<T[Key]> }
      : T;

export type Page<T> = {
  items: T[];
  totalItems: number;
  totalPages: number;
  currentPage: number;
  hasNext: boolean;
  hasPrev: boolean;
};

export const ORG_ID = "00000000-0000-4000-8000-0000000000a1";
export const NOW = "2026-10-23T12:00:00.000Z";

export function page<T>(items: T[]): Page<T> {
  return {
    items,
    totalItems: items.length,
    totalPages: items.length ? 1 : 0,
    currentPage: 1,
    hasNext: false,
    hasPrev: false,
  };
}

/** One server page of `items`, as the API slices it. */
export function slice<T>(items: T[], currentPage: number, pageSize: number): Page<T> {
  const totalPages = Math.ceil(items.length / pageSize);

  return {
    items: items.slice((currentPage - 1) * pageSize, currentPage * pageSize),
    totalItems: items.length,
    totalPages,
    currentPage,
    hasNext: currentPage < totalPages,
    hasPrev: currentPage > 1,
  };
}

export function brand(overrides: Partial<Wire<BrandRes>> = {}): Wire<BrandRes> {
  return {
    id: "brand",
    organizationId: ORG_ID,
    name: "Koi",
    domain: "koi.example",
    githubRepo: "",
    country: "IT",
    description: "",
    ...overrides,
  };
}

export function me(overrides: Partial<Wire<MeRes>> = {}): Wire<MeRes> {
  return {
    id: "user",
    email: "founder@example.com",
    organization: { id: ORG_ID, name: "Koi", role: "owner" },
    activeBrand: brand(),
    ...overrides,
  };
}

export function capabilities(
  overrides: Partial<Wire<CapabilitiesResponse>> = {},
): Wire<CapabilitiesResponse> {
  return {
    cloudBilling: false,
    liveCollection: true,
    competitorSearch: false,
    aiDrafting: false,
    mcpUrl: "https://koi.example/mcp",
    logoDevPublishableKey: "",
    analytics: null,
    ...overrides,
  };
}

export function billingCatalog(): Wire<BillingCatalog> {
  return {
    currency: "eur",
    interval: "month",
    options: [
      { prompts: 15, amountCents: 2900, currency: "eur", interval: "month" },
      { prompts: 25, amountCents: 4500, currency: "eur", interval: "month" },
      { prompts: 50, amountCents: 7500, currency: "eur", interval: "month" },
    ],
  };
}

export function subscription(
  overrides: Partial<Wire<BillingSubscription>> = {},
): Wire<BillingSubscription> {
  return {
    organizationId: ORG_ID,
    quantity: 25,
    status: "active",
    cancelAtPeriodEnd: false,
    pendingQuantity: null,
    pendingUpgradeQuantity: null,
    pendingUpgradeExpiresAt: null,
    paymentUrl: "",
    ...overrides,
  };
}

export function billingStatus(overrides: Partial<Wire<BillingStatus>> = {}): Wire<BillingStatus> {
  return {
    subscription: subscription(),
    promptLimit: 25,
    usedPrompts: 0,
    trialAvailable: false,
    firstLook: false,
    ...overrides,
  };
}

export function metric(overrides: Partial<VisibilityMetric> = {}): VisibilityMetric {
  return { completed: 4, cited: 1, mentioned: 1, rate: 50, ...overrides };
}

/** Daily points from 2026-10-01, one per rate; `null` is a day without completed answers. */
export function series(rates: (number | null)[]): Wire<VisibilityDay>[] {
  return rates.map((rate, index) => ({
    date: `2026-10-${String(index + 1).padStart(2, "0")}`,
    metric: metric({ rate }),
  }));
}

export function competitorVisibility(
  overrides: Partial<Wire<CompetitorVisibility>> = {},
): Wire<CompetitorVisibility> {
  return {
    id: "rival",
    name: "Rival",
    domain: "rival.example",
    metric: metric({ rate: 25 }),
    series: series([25, 25]),
    ...overrides,
  };
}

export function report(overrides: Partial<Wire<VisibilityReport>> = {}): Wire<VisibilityReport> {
  return {
    metric: metric(),
    previous: metric(),
    series: series([50, 50]),
    engines: {},
    competitors: [],
    ...overrides,
  };
}

export function prompt(overrides: Partial<Wire<PromptRes>> = {}): Wire<PromptRes> {
  return {
    id: "prompt-0",
    brandId: "brand",
    text: "Which analytics tools should a founder consider?",
    source: "manual",
    confirmed: true,
    inactive: false,
    hasResults: true,
    ...overrides,
  };
}

export function competitor(overrides: Partial<Wire<CompetitorRes>> = {}): Wire<CompetitorRes> {
  return {
    id: "rival",
    brandId: "brand",
    name: "Rival",
    domain: "rival.example",
    githubRepo: "",
    source: "manual",
    confirmed: true,
    aliases: [],
    additionalUrls: [],
    ...overrides,
  };
}

export function tracking(overrides: Partial<Wire<TrackingStatus>> = {}): Wire<TrackingStatus> {
  return {
    confirmed: true,
    startedAt: NOW,
    nextRunAt: null,
    blockedReason: null,
    ...overrides,
  };
}

export function run(overrides: Partial<Wire<VisibilityRunStatus>> = {}): Wire<VisibilityRunStatus> {
  const id = overrides.id ?? "run";

  return {
    id,
    brandId: "brand",
    engines: ["chatgpt", "gemini", "google_ai_mode"],
    state: "running",
    totalResults: 3,
    completedResults: 0,
    createdAt: NOW,
    startedAt: NOW,
    finishedAt: null,
    errorCode: null,
    statusUrl: `/api/v1/runs/${id}`,
    ...overrides,
  };
}

export function result(
  overrides: Partial<Wire<VisibilityResult>> & Pick<VisibilityResult, "engine">,
): Wire<VisibilityResult> {
  const outcome = overrides.outcome === undefined ? "mentioned" : overrides.outcome;

  return {
    id: `result-${overrides.engine}`,
    runId: "run",
    promptId: "prompt-0",
    promptText: "Which analytics tools should a founder consider?",
    promptPosition: 0,
    collector: "cloro",
    fetchedAt: outcome ? NOW : null,
    rawAnswer: "",
    sources: [],
    errorCode: outcome === "error" ? "upstream_error" : null,
    appearances: [],
    mentionVerdict: null,
    ...overrides,
    outcome,
  };
}

export function generation(
  overrides: Partial<Wire<GenerationStatus>> = {},
): Wire<GenerationStatus> {
  return {
    id: "generation",
    state: "succeeded",
    attempt: 1,
    promptCount: 5,
    competitorCount: 3,
    createdAt: NOW,
    startTracking: false,
    ...overrides,
  };
}

export function credential(overrides: Partial<Wire<Credential>> = {}): Wire<Credential> {
  return {
    id: "credential",
    name: "Claude Code",
    clientId: "claude",
    createdAt: NOW,
    ...overrides,
  };
}
