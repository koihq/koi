import { expect, type Page, test } from "@playwright/test";

const brand = {
  id: "brand_1",
  organizationId: "org_1",
  name: "Acme",
  domain: "acme.com",
  githubRepo: "",
  country: "IT",
  description: "",
};

const page1 = <T>(items: T[]) => ({
  items,
  totalItems: items.length,
  totalPages: items.length ? 1 : 0,
  currentPage: 1,
  hasNext: false,
  hasPrev: false,
});

const run = (completedResults: number, state = "running") => ({
  id: "run_1",
  brandId: brand.id,
  engines: ["chatgpt", "perplexity", "gemini", "google_ai_overviews"],
  state,
  totalResults: 20,
  completedResults,
  createdAt: new Date().toISOString(),
  startedAt: new Date().toISOString(),
  finishedAt: null,
  errorCode: null,
  statusUrl: "/api/v1/runs/run_1",
});

const result = (engine: string, outcome: string | null, appearances: unknown[] = []) => ({
  id: `result_${engine}`,
  runId: "run_1",
  promptId: "prompt_1",
  promptText: "What is the best payment platform for startups?",
  promptPosition: 0,
  engine,
  collector: "cloro",
  fetchedAt: outcome ? new Date().toISOString() : null,
  outcome,
  rawAnswer: outcome ? "Stripe is the most common choice for startups." : "",
  sources: outcome ? ["https://stripe.com/docs"] : [],
  errorCode: null,
  appearances,
});

const answers = [
  result("chatgpt", "miss", [
    { competitorId: null, name: "Stripe", confirmed: false, evidence: [] },
  ]),
  result("perplexity", "mentioned"),
  result("gemini", null),
];

type State = {
  brand: typeof brand | null;
  autoStart: boolean;
  started: boolean;
  completed: number;
  runState: string;
  starts: number;
  credentials: unknown[];
  steps: string[];
};

async function mockOnboarding(page: Page, mcpUrl = "https://koi.example/mcp") {
  const state: State = {
    brand: null,
    autoStart: false,
    started: false,
    completed: 0,
    runState: "running",
    starts: 0,
    credentials: [],
    steps: [],
  };

  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    let json: unknown;

    if (path === "/api/v1/auth/me") {
      json = {
        id: "user_1",
        email: "founder@acme.com",
        organization: { id: "org_1", name: "Acme" },
        activeBrand: state.brand,
      };
    } else if (path === "/api/v1/capabilities") {
      json = {
        cloudBilling: true,
        aiDrafting: true,
        liveCollection: true,
        competitorSearch: false,
        mcpUrl,
      };
    } else if (path === "/api/v1/billing/subscription") {
      json = {
        subscription: null,
        promptLimit: 5,
        usedPrompts: 5,
        trialAvailable: true,
        firstLook: true,
      };
    } else if (path === "/api/v1/billing/catalog") {
      json = {
        currency: "eur",
        interval: "month",
        options: [{ prompts: 15, amountCents: 2900, currency: "eur", interval: "month" }],
      };
    } else if (path === "/api/v1/onboarding/events") {
      state.steps.push(request.postDataJSON().step);
      await route.fulfill({ status: 204 });
      return;
    } else if (path === "/api/v1/brands/countries") {
      json = [{ code: "IT", name: "Italy" }];
    } else if (path === "/api/v1/brands/lookup") {
      json = url.searchParams.get("domain")?.includes("unknown")
        ? { found: false, name: "", description: "", source: "" }
        : { found: true, name: "Acme Pay", description: "Payments", source: "website" };
    } else if (path === "/api/v1/brands" && request.method() === "POST") {
      state.brand = { ...brand, ...request.postDataJSON() };
      json = state.brand;
    } else if (path === "/api/v1/brands") {
      json = page1(state.brand ? [state.brand] : []);
    } else if (path === "/api/v1/credentials") {
      json = page1(state.credentials);
    } else if (path === "/api/v1/generations" && request.method() === "POST") {
      state.autoStart = request.postDataJSON().startTracking === true;
      await route.fulfill({
        status: 202,
        json: {
          id: "generation_1",
          state: "queued",
          statusUrl: "/api/v1/generations/generation_1",
        },
      });
      return;
    } else if (path === "/api/v1/generations") {
      json = page1([
        {
          id: "generation_1",
          state: "succeeded",
          attempt: 1,
          promptCount: 5,
          competitorCount: 3,
          createdAt: new Date().toISOString(),
          startTracking: state.autoStart,
          summary: {
            product: "Acme",
            category: "Payments for startups",
            audience: "Founders",
            functions: ["Takes payments"],
          },
        },
      ]);
    } else if (path === "/api/v1/tracking") {
      const now = new Date().toISOString();
      json = {
        confirmed: state.started,
        startedAt: state.started ? now : null,
        nextRunAt: state.started ? now : null,
        blockedReason: state.started ? "first_look_used" : "start_required",
      };
    } else if (path === "/api/v1/tracking/start") {
      state.started = true;
      state.starts += 1;
      await route.fulfill({ status: 204 });
      return;
    } else if (path === "/api/v1/runs") {
      json = page1(state.started ? [run(state.completed, state.runState)] : []);
    } else if (path === "/api/v1/runs/run_1/results") {
      json = page1(state.completed ? answers : answers.map((a) => ({ ...a, outcome: null })));
    } else if (path === "/api/v1/prompts" || path === "/api/v1/competitors") {
      json = page1([]);
    } else {
      await route.fulfill({
        status: 404,
        json: { error: { code: "not_found", message: "not_found" } },
      });
      return;
    }

    await route.fulfill({ json });
  });

  return state;
}

test("takes a person from the choice to a real answer before any payment", async ({
  context,
  page,
}, testInfo) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  const state = await mockOnboarding(page);

  await page.goto("/setup");
  await expect(
    page.getByRole("heading", { name: "Let's find out what AI says about you." }),
  ).toBeVisible();
  await expect(page.getByText("Step 1 of 4")).toBeVisible();
  await expect(page.getByRole("link", { name: "Back" })).toHaveCount(0);
  await expect(page.getByRole("radio", { name: /Show me here/ })).toBeChecked();
  await page.screenshot({ path: testInfo.outputPath("setup-choice.png") });
  await page.getByRole("link", { name: "Let's go" }).click();

  // The work email already names the site.
  await expect(page.getByRole("heading", { name: "What's your website?" })).toBeVisible();
  await expect(page.getByText("Step 2 of 4")).toBeVisible();
  const site = page.getByLabel("Website or GitHub repo");
  await expect(site).toHaveValue("acme.com");
  await page.screenshot({ path: testInfo.outputPath("setup-website.png") });

  await site.fill("not a site");
  await page.getByRole("button", { name: "Continue" }).click();
  await expect(page.getByText("Enter a website or a GitHub repository URL")).toBeVisible();

  await site.fill("https://www.acme.com/pricing");
  await page.getByRole("button", { name: "Continue" }).click();

  // The name comes from the site, says so, and stays editable.
  await expect(page.getByRole("heading", { name: "Is this you?" })).toBeVisible();
  await expect(page.getByText("Step 3 of 4")).toBeVisible();
  await expect(page.getByLabel("Brand name")).toHaveValue("Acme Pay");
  await expect(
    page.getByText("From www.acme.com").or(page.getByText("From acme.com")),
  ).toBeVisible();
  await page.locator("#brand-country").fill("IT");
  await page.getByRole("option", { name: "IT (Italy)" }).click();
  await page.screenshot({ path: testInfo.outputPath("setup-confirm.png") });

  // Going back keeps what was typed.
  await page.getByRole("link", { name: "Back" }).click();
  await expect(site).toHaveValue("https://www.acme.com/pricing");
  await page.getByRole("button", { name: "Continue" }).click();
  await page.locator("#brand-country").fill("IT");
  await page.getByRole("option", { name: "IT (Italy)" }).click();
  await page.getByRole("button", { name: "See what AI says" }).click();

  await expect(page.getByText("Step 4 of 4")).toBeVisible();
  await expect(page.getByText("Asking around…")).toBeVisible();
  await expect(page.getByText("Feel free to wander off.", { exact: false })).toBeVisible();
  await expect(page.getByText("Payments for startups")).toBeVisible();
  // The price and the way off the page: visible from the start, with navigation hidden.
  await expect(page.getByRole("heading", { name: "€0 today" })).toBeVisible();
  await expect(page.getByText("Then €29 a month after 7 days", { exact: false })).toBeVisible();
  await expect(page.getByRole("combobox", { name: /Active brand/ })).toHaveCount(0);
  await page.getByRole("button", { name: "Account" }).click();
  const signOut = page.getByRole("menuitem", { name: "Sign out" });
  await expect(signOut).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "Billing" })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(signOut).toHaveCount(0);
  expect(state.autoStart).toBe(true);
  expect(state.starts).toBe(1);
  await page.screenshot({ path: testInfo.outputPath("first-look-progress.png"), fullPage: true });

  state.completed = 2;
  await expect(
    page.getByRole("heading", { name: "ChatGPT recommends Stripe instead of Acme Pay" }),
  ).toBeVisible({ timeout: 10_000 });
  await expect(page.getByText("Stripe is the most common choice")).toBeVisible();
  await expect(page.getByRole("link", { name: /stripe\.com/ })).toBeVisible();
  await expect(page.getByText("1 of 2 answers mention Acme Pay")).toBeAttached();

  // Choosing another engine shows its answer without moving the headline.
  await page
    .locator("fieldset")
    .getByRole("button", { name: /Perplexity/ })
    .click();
  await expect(
    page.getByRole("heading", { name: "ChatGPT recommends Stripe instead of Acme Pay" }),
  ).toBeVisible();
  await expect(page.getByRole("link", { name: "Keep an eye on this" })).toHaveAttribute(
    "href",
    "/settings/billing",
  );
  await expect(page.getByRole("heading", { name: "Where to start" })).toHaveCount(0);

  state.runState = "succeeded";
  await expect(page.getByRole("heading", { name: "Where to start" })).toBeVisible({
    timeout: 70_000,
  });
  await expect(page.getByText("Stripe came up in 1 of 1 answers", { exact: false })).toBeVisible();
  await expect(page.getByRole("link", { name: "See your competitors" })).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath("first-look-answer.png"), fullPage: true });

  await page.getByRole("button", { name: "Fix this with my agent" }).click();
  await expect(page.getByText("Paste it into your agent.")).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toContain(
    "visibility brief for Acme Pay",
  );

  expect(state.starts).toBe(1);
  expect(state.steps).toEqual(
    expect.arrayContaining([
      "choice",
      "website",
      "confirm",
      "first_look",
      "first_answer",
      "agent_handoff",
    ]),
  );
});

test("does not claim a site named the brand when the lookup found nothing", async ({ page }) => {
  await mockOnboarding(page);

  await page.goto("/setup?path=koi&site=unknown-brand.com");
  await expect(page.getByLabel("Brand name")).toHaveValue("Unknown-brand");
  await expect(page.getByText("From unknown-brand.com")).toHaveCount(0);
});

test("waits for the agent, then moves to its check without starting one", async ({
  page,
}, testInfo) => {
  const state = await mockOnboarding(page);

  await page.goto("/setup");
  await page.getByRole("radio", { name: /Show me here/ }).focus();
  await page.keyboard.press("ArrowRight");
  await expect(page.getByRole("radio", { name: /Let my agent handle it/ })).toBeChecked();
  await page.getByRole("link", { name: "Let's go" }).click();

  await expect(page.getByText("Step 2 of 4")).toBeVisible();
  await page.getByRole("button", { name: "Claude Code" }).click();
  await expect(
    page.getByText("claude mcp add --transport http koi", { exact: false }),
  ).toBeVisible();
  await expect(page.getByText("npx skills add koihq/koi", { exact: false })).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath("setup-agent.png"), fullPage: true });

  // Signing in moves on by itself.
  state.credentials = [{ id: "credential_1", name: "Claude Code", clientId: "claude" }];
  await expect(page.getByText("Step 3 of 4")).toBeVisible({ timeout: 10_000 });
  await expect(page.getByRole("button", { name: "Copy prompt" })).toBeVisible();
  await expect(page.getByText("Claude Code's here.", { exact: false })).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath("setup-agent-prompt.png"), fullPage: true });

  state.brand = brand;
  await expect(page).toHaveURL(/\/$/, { timeout: 10_000 });

  // The agent may still be correcting its prompts, so the page must not spend the check.
  await expect(page.getByRole("button", { name: "Start tracking" })).toBeVisible();
  expect(state.starts).toBe(0);
});

test("hovering a link does not ask the server for the account again", async ({ page }) => {
  await mockOnboarding(page);

  await page.goto("/setup");
  const proceed = page.getByRole("link", { name: "Let's go" });
  await proceed.waitFor();

  let accountCalls = 0;
  page.on("request", (request) => {
    if (new URL(request.url()).pathname === "/api/v1/auth/me") accountCalls += 1;
  });

  await proceed.hover();
  await page.mouse.move(0, 0);
  await proceed.hover();
  await page.waitForTimeout(500);

  expect(accountCalls).toBe(0);
});

test("skips the choice when no agent can connect", async ({ page }) => {
  await mockOnboarding(page, "");

  await page.goto("/setup");
  await expect(page.getByRole("heading", { name: "What's your website?" })).toBeVisible();
  await expect(page.getByText("Step 1 of 3")).toBeVisible();
  await expect(page.getByRole("link", { name: "Back" })).toHaveCount(0);
});

test("keeps every setup view inside a narrow viewport", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 740 });
  await mockOnboarding(page);

  for (const path of [
    "/setup",
    "/setup?path=koi",
    "/setup?path=koi&site=acme.com",
    "/setup?path=agent",
    "/setup?path=agent&stage=prompt",
  ]) {
    await page.goto(path);
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(
      320,
    );
  }
});
