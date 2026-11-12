import { HttpResponse, http } from "msw";
import { setupWorker } from "msw/browser";

import type {
  BillingStatus,
  BrandLookup,
  BrandRes,
  CapabilitiesResponse,
  CompetitorRes,
  CountryOption,
  Credential,
  GenerationStatus,
  MeRes,
  PromptHistoryRun,
  PromptRes,
  TrackingStatus,
  VisibilityReport,
  VisibilityResult,
  VisibilityRunStatus,
} from "@/api/generated";
import {
  billingCatalog,
  billingStatus,
  brand,
  capabilities,
  me,
  page,
  report,
  slice,
  tracking,
  type Wire,
} from "@/test/fixtures";

export const worker = setupWorker();

/** Endpoints a test can make fail with a 500 while the rest of the workspace keeps working. */
export type Failure =
  | "activate"
  | "brands"
  | "history"
  | "promptVisibility"
  | "summary"
  | "tracking";

/** Server state behind the workspace handlers. Tests mutate it to change what the API returns. */
export type Workspace = {
  me: Wire<MeRes>;
  brands: Wire<BrandRes>[];
  capabilities: Wire<CapabilitiesResponse>;
  billing: Wire<BillingStatus>;
  tracking: Wire<TrackingStatus>;
  summary: Wire<VisibilityReport>;
  prompts: Wire<PromptRes>[];
  promptVisibility: Wire<VisibilityReport>;
  competitors: Wire<CompetitorRes>[];
  runs: Wire<VisibilityRunStatus>[];
  /** Results of every run, keyed by run ID. Prompt history groups them by prompt. */
  results: Record<string, Wire<VisibilityResult>[]>;
  generations: Wire<GenerationStatus>[];
  credentials: Wire<Credential>[];
  countries: CountryOption[];
  lookup: (domain: string) => BrandLookup;
  fail: Set<Failure>;
  /** Every API request, in order. */
  requests: URL[];
  /** Onboarding steps the page reported. */
  steps: string[];
};

export function workspace(overrides: Partial<Workspace> = {}): Workspace {
  const account = overrides.me ?? me();

  return {
    me: account,
    brands: account.activeBrand ? [account.activeBrand] : [],
    capabilities: capabilities(),
    billing: billingStatus(),
    tracking: tracking(),
    summary: report(),
    prompts: [],
    promptVisibility: report(),
    competitors: [],
    runs: [],
    results: {},
    generations: [],
    credentials: [],
    countries: [{ code: "IT", name: "Italy" }],
    lookup: () => ({ found: false, name: "", description: "", source: "" }),
    fail: new Set(),
    requests: [],
    steps: [],
    ...overrides,
  };
}

export function apiError(status: number, code: string) {
  return HttpResponse.json({ error: { code, message: code } }, { status });
}

const internal = () => apiError(500, "internal");

function paged<T>(items: T[], url: URL) {
  return slice(
    items,
    Number(url.searchParams.get("page") ?? 1),
    Number(url.searchParams.get("pageSize") ?? 20),
  );
}

/** Serves `state` as the signed-in account's API. Handlers read it on every request. */
export function serveWorkspace(state: Workspace) {
  const activeBrand = () => state.me.activeBrand;

  worker.use(
    http.all("/api/v1/*", ({ request }) => {
      state.requests.push(new URL(request.url));
    }),

    http.get("/api/v1/auth/me", () => HttpResponse.json(state.me)),
    http.get("/api/v1/capabilities", () => HttpResponse.json(state.capabilities)),

    http.get("/api/v1/billing/catalog", () => HttpResponse.json(billingCatalog())),
    http.get("/api/v1/billing/subscription", () => HttpResponse.json(state.billing)),

    http.get("/api/v1/brands/countries", () => HttpResponse.json(state.countries)),
    http.get("/api/v1/brands/lookup", ({ request }) =>
      HttpResponse.json(state.lookup(new URL(request.url).searchParams.get("domain") ?? "")),
    ),
    http.get("/api/v1/brands", ({ request }) => {
      if (state.fail.has("brands")) return internal();

      const url = new URL(request.url);
      const query = (url.searchParams.get("q") ?? "").toLowerCase();
      const matches = state.brands.filter((brand) =>
        `${brand.name} ${brand.domain}`.toLowerCase().includes(query),
      );

      return HttpResponse.json(paged(matches, url));
    }),
    http.post("/api/v1/brands", async ({ request }) => {
      const created = brand({
        id: `brand-${state.brands.length + 1}`,
        ...((await request.json()) as Partial<Wire<BrandRes>>),
      });

      state.brands = [...state.brands, created];
      state.me = { ...state.me, activeBrand: created };

      return HttpResponse.json(created, { status: 201 });
    }),
    http.get("/api/v1/brands/:id", ({ params }) => {
      const found = state.brands.find((brand) => brand.id === params.id);

      return found ? HttpResponse.json(found) : apiError(404, "not_found");
    }),
    http.patch("/api/v1/brands/:id", async ({ params, request }) => {
      const index = state.brands.findIndex((brand) => brand.id === params.id);
      if (index < 0) return apiError(404, "not_found");

      state.brands[index] = {
        ...state.brands[index],
        ...((await request.json()) as Partial<Wire<BrandRes>>),
      };

      return HttpResponse.json(state.brands[index]);
    }),
    http.post("/api/v1/brands/:id/activate", ({ params }) => {
      if (state.fail.has("activate")) return internal();

      const found = state.brands.find((brand) => brand.id === params.id);
      if (!found) return apiError(404, "not_found");

      state.me = { ...state.me, activeBrand: found };

      return HttpResponse.json(found);
    }),

    http.post("/api/v1/onboarding/events", async ({ request }) => {
      state.steps.push(((await request.json()) as { step: string }).step);

      return new HttpResponse(null, { status: 204 });
    }),

    http.get("/api/v1/visibility/summary", () => {
      if (state.fail.has("summary")) return internal();

      return HttpResponse.json({ ...state.summary, hasConfirmedPrompts: state.prompts.length > 0 });
    }),
    http.get("/api/v1/tracking", () => {
      if (state.fail.has("tracking")) return internal();

      return HttpResponse.json(state.tracking);
    }),

    http.get("/api/v1/prompts", ({ request }) => {
      const url = new URL(request.url);
      const confirmed = url.searchParams.get("confirmed");
      const inactive = url.searchParams.get("inactive");
      const query = (url.searchParams.get("q") ?? "").toLowerCase();
      const matches = state.prompts.filter(
        (prompt) =>
          (confirmed === null || String(prompt.confirmed) === confirmed) &&
          (inactive === null || String(prompt.inactive) === inactive) &&
          prompt.text.toLowerCase().includes(query),
      );

      return HttpResponse.json(paged(matches, url));
    }),
    http.get("/api/v1/prompts/:id", ({ params }) => {
      const found = state.prompts.find((prompt) => prompt.id === params.id);

      return found ? HttpResponse.json(found) : apiError(404, "not_found");
    }),
    http.get("/api/v1/prompt-history/:id/visibility", () => {
      if (state.fail.has("promptVisibility")) return internal();

      return HttpResponse.json(state.promptVisibility);
    }),
    http.get("/api/v1/prompt-history/:id/runs", ({ params }) => {
      if (state.fail.has("history")) return internal();

      const history: Wire<PromptHistoryRun>[] = state.runs
        .filter((run) => run.brandId === activeBrand()?.id)
        .map((run) => ({
          run,
          results: (state.results[run.id] ?? []).filter((result) => result.promptId === params.id),
        }));

      return HttpResponse.json(page(history));
    }),

    http.get("/api/v1/runs", () =>
      HttpResponse.json(page(state.runs.filter((run) => run.brandId === activeBrand()?.id))),
    ),
    http.get("/api/v1/runs/:id/results", ({ params }) =>
      HttpResponse.json(page(state.results[params.id as string] ?? [])),
    ),

    http.get("/api/v1/competitors", ({ request }) => {
      const url = new URL(request.url);
      const query = (url.searchParams.get("q") ?? "").toLowerCase();
      const confirmed = url.searchParams.get("confirmed");
      const matches = state.competitors.filter(
        (competitor) =>
          (confirmed === null || String(competitor.confirmed) === confirmed) &&
          `${competitor.name} ${competitor.domain}`.toLowerCase().includes(query),
      );

      return HttpResponse.json(paged(matches, url));
    }),

    http.get("/api/v1/generations", () => HttpResponse.json(page(state.generations))),
    http.get("/api/v1/credentials", () => HttpResponse.json(page(state.credentials))),
  );

  return state;
}
