import { HttpResponse, http } from "msw";

import type { GenerationAccepted, GenerationRequest } from "@/api/generated";
import {
  billingStatus,
  brand,
  capabilities,
  competitor,
  generation,
  me,
  page,
  prompt,
  result,
  run,
} from "@/test/fixtures";
import { apiError, serveWorkspace, type Workspace, worker, workspace } from "@/test/worker";

export const ACME = brand({ id: "brand-acme", name: "Acme", domain: "acme.com" });
export const DRAFTED_PROMPT = "What is the best payment platform for startups?";

/** What the onboarding server knows; the handlers derive every response from it. */
export type Onboarding = {
  workspace: Workspace;
  autoStart: boolean;
  started: boolean;
  starts: number;
  completed: number;
  runState: "running" | "succeeded" | "failed";
  blocked: "" | "first_look_used" | "first_look_site_used" | "first_look_paused" | null;
  draftBlocked: boolean;
  noAnswers: boolean;
};

/** A new founder on Koi Cloud, before their free check. */
export function serveOnboarding({ mcpUrl = "https://koi.example/mcp" } = {}): Onboarding {
  const state: Onboarding = {
    workspace: serveWorkspace(
      workspace({
        me: me({ email: "founder@acme.com", activeBrand: null }),
        brands: [],
        capabilities: capabilities({
          cloudBilling: true,
          aiDrafting: true,
          liveCollection: true,
          mcpUrl,
        }),
        billing: billingStatus({
          subscription: null,
          promptLimit: 5,
          usedPrompts: 5,
          trialAvailable: true,
          firstLook: true,
        }),
        lookup: (domain) =>
          domain.includes("unknown")
            ? { found: false, name: "", description: "", source: "" }
            : { found: true, name: "Acme Pay", description: "Payments", source: "website" },
      }),
    ),
    autoStart: false,
    started: false,
    starts: 0,
    completed: 0,
    runState: "running",
    blocked: null,
    draftBlocked: false,
    noAnswers: false,
  };

  const active = () => state.workspace.me.activeBrand ?? ACME;
  const answers = () => [
    result({
      engine: "chatgpt",
      runId: "run-first",
      promptId: "prompt-first",
      promptText: DRAFTED_PROMPT,
      outcome: state.completed ? "miss" : null,
      rawAnswer: state.completed ? "Stripe is the most common choice for startups." : "",
      sources: state.completed ? ["https://stripe.com/docs"] : [],
      appearances: [{ competitorId: null, name: "Stripe", confirmed: false, evidence: [] }],
    }),
    result({
      engine: "google_ai_mode",
      runId: "run-first",
      promptId: "prompt-first",
      promptText: DRAFTED_PROMPT,
      outcome: state.completed ? "mentioned" : null,
      rawAnswer: state.completed ? "Stripe is the most common choice for startups." : "",
      sources: state.completed ? ["https://stripe.com/docs"] : [],
    }),
    result({
      engine: "gemini",
      runId: "run-first",
      promptId: "prompt-first",
      promptText: DRAFTED_PROMPT,
      outcome: null,
    }),
  ];

  worker.use(
    http.post("/api/v1/brands", async ({ request }) => {
      const created = { ...ACME, ...((await request.json()) as Partial<typeof ACME>) };
      addBrand(state, created);

      return HttpResponse.json(created, { status: 201 });
    }),
    http.post("/api/v1/generations", async ({ request }) => {
      if (state.draftBlocked) return apiError(409, "first_look_unavailable");

      state.autoStart = ((await request.json()) as GenerationRequest).startTracking === true;

      const accepted: GenerationAccepted = {
        id: "generation",
        state: "queued",
        statusUrl: "/api/v1/generations/generation",
      };

      return HttpResponse.json(accepted, { status: 202 });
    }),
    http.get("/api/v1/generations", () =>
      HttpResponse.json(
        page([
          generation({
            startTracking: state.autoStart,
            summary: {
              product: "Acme",
              category: "Payments for startups",
              audience: "Founders",
              functions: ["Takes payments"],
            },
          }),
        ]),
      ),
    ),
    http.get("/api/v1/tracking", () =>
      HttpResponse.json({
        confirmed: state.started,
        startedAt: state.started ? new Date().toISOString() : null,
        nextRunAt: state.started ? new Date().toISOString() : null,
        blockedReason: state.blocked ?? (state.started ? "first_look_used" : "start_required"),
      }),
    ),
    http.post("/api/v1/tracking/start", () => {
      state.started = true;
      state.starts += 1;

      return new HttpResponse(null, { status: 204 });
    }),
    http.get("/api/v1/runs", () =>
      HttpResponse.json(
        page(
          state.started && (!state.blocked || state.noAnswers)
            ? [
                run({
                  id: "run-first",
                  brandId: active().id,
                  engines: ["chatgpt", "gemini", "google_ai_mode", "google_ai_overviews"],
                  state: state.runState,
                  totalResults: 20,
                  completedResults: state.completed,
                }),
              ]
            : [],
        ),
      ),
    ),
    http.get("/api/v1/runs/:id/results", () => HttpResponse.json(page(answers()))),
    http.get("/api/v1/prompts", () =>
      HttpResponse.json(
        page(
          state.workspace.me.activeBrand
            ? [
                prompt({
                  id: "prompt-first",
                  brandId: active().id,
                  text: DRAFTED_PROMPT,
                  source: "drafted",
                  hasResults: false,
                }),
              ]
            : [],
        ),
      ),
    ),
    http.get("/api/v1/competitors", () =>
      HttpResponse.json(
        page(
          state.workspace.me.activeBrand
            ? [
                competitor({
                  id: "stripe",
                  brandId: active().id,
                  name: "Stripe",
                  domain: "stripe.com",
                  source: "drafted",
                }),
              ]
            : [],
        ),
      ),
    ),
  );

  return state;
}

/** Gives the onboarding account its brand, as if an agent or the form had created it. */
export function addBrand(state: Onboarding, created: typeof ACME = ACME) {
  state.workspace.brands = [created];
  state.workspace.me = { ...state.workspace.me, activeBrand: created };
}
