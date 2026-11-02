import { expect, type Page, test } from "@playwright/test";

async function mockWorkspace(page: Page, rates: (number | null)[] = [50, 50], compare = false) {
  const brand = {
    id: "brand",
    organizationId: "org",
    name: "Koi",
    domain: "koi.example",
    githubRepo: "",
    country: "IT",
    description: "",
  };
  const metric = { completed: 4, cited: 1, mentioned: 1, rate: rates.at(-1) ?? 50 };
  const series = rates.map((rate, index) => ({
    date: `2026-10-${String(1 + index).padStart(2, "0")}`,
    metric: { ...metric, rate },
  }));
  const report = {
    metric,
    previous: metric,
    series,
    engines: {},
    competitors: compare
      ? [
          {
            id: "rival",
            name: "Rival",
            domain: "rival.example",
            metric: { ...metric, rate: 100 - metric.rate },
            series: series.map((point) => ({
              ...point,
              metric: {
                ...point.metric,
                rate: point.metric.rate === null ? null : 100 - point.metric.rate,
              },
            })),
          },
        ]
      : [],
  };
  const prompts = [
    "First question about analytics tools",
    "Second question about analytics tools",
  ].map((text, index) => ({
    id: `prompt-${index}`,
    brandId: brand.id,
    text,
    source: "manual",
    confirmed: true,
    inactive: false,
    visibility: report,
  }));
  const run = {
    id: "run",
    brandId: brand.id,
    engines: ["chatgpt", "gemini", "perplexity"],
    state: "running",
    totalResults: 3,
    completedResults: 0,
    createdAt: "2026-10-23T12:00:00Z",
    startedAt: "2026-10-23T12:00:00Z",
    finishedAt: null,
    errorCode: null,
    statusUrl: "",
  };
  const state = {
    blockedReason: null as string | null,
    run,
    results: [] as Record<string, unknown>[],
    failHistory: false,
    activeBrand: brand,
    trackingFailed: false,
    started: true,
  };
  const requests: URL[] = [];
  const envelope = (items: unknown[]) => ({
    items,
    totalItems: items.length,
    totalPages: items.length ? 1 : 0,
    currentPage: 1,
    hasNext: false,
    hasPrev: false,
  });
  const result = (engine: string, outcome: string | null, rawAnswer = "") => ({
    id: engine,
    runId: run.id,
    promptId: "prompt-0",
    promptText: prompts[0].text,
    promptPosition: 0,
    engine,
    collector: "cloro",
    fetchedAt: "2026-10-23T12:00:00Z",
    outcome,
    rawAnswer,
    sources: [],
    errorCode: outcome === "error" ? "upstream_error" : null,
    appearances: [],
  });

  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    requests.push(url);
    let json: unknown;
    const path = url.pathname;
    if (path.endsWith("/auth/me"))
      json = {
        id: "user",
        email: "founder@example.com",
        organization: { id: "org", name: "Workspace", role: "owner" },
        activeBrand: state.activeBrand,
      };
    else if (path.endsWith("/capabilities"))
      json = {
        cloudBilling: false,
        aiDrafting: false,
        liveCollection: false,
        competitorSearch: false,
      };
    else if (path.endsWith("/tracking") && !state.trackingFailed)
      json = {
        promptsReviewed: true,
        competitorsReviewed: true,
        startedAt: state.started ? run.startedAt : null,
        nextRunAt: null,
        blockedReason: state.blockedReason,
      };
    else if (path.endsWith("/visibility/summary")) json = { ...report, hasConfirmedPrompts: true };
    else if (path === "/api/v1/visibility/runs")
      json = envelope(state.activeBrand.id === "brand" ? [run] : []);
    else if (path.endsWith("/prompts")) json = envelope(prompts);
    else if (path.endsWith("/prompts/prompt-0")) json = prompts[0];
    else if (path.endsWith("/prompt-history/prompt-0/visibility")) json = report;
    else if (path.endsWith("/prompt-history/prompt-0/runs") && !state.failHistory)
      json = envelope([{ run, results: state.results }]);
    else if (path.endsWith("/generations")) json = envelope([]);
    else if (path.endsWith("/competitors"))
      json = envelope([
        {
          id: "rival",
          brandId: "brand",
          name: "An unusually long competitor name for checking wrapping",
          domain: "rival.example",
          githubRepo: "",
          source: "manual",
          confirmed: true,
        },
      ]);
    else if (path.endsWith("/brands"))
      json = envelope([brand, { ...brand, id: "other", name: "Other brand" }]);
    else if (path.endsWith("/brands/other/activate")) {
      state.activeBrand = { ...brand, id: "other", name: "Other brand" };
      json = state.activeBrand;
    } else if (path.endsWith("/brands/brand")) json = brand;
    else if (path.endsWith("/countries") || path.endsWith("/credentials")) json = envelope([]);
    else {
      await route.fulfill({
        status: 500,
        json: { error: { code: "internal", message: "Try again." } },
      });
      return;
    }
    await route.fulfill({ json });
  });
  return { state, requests, result };
}

test("confirmed prompts do not hide collection blockers or tracking recovery", async ({ page }) => {
  const { state } = await mockWorkspace(page);
  state.blockedReason = "collection_unavailable";
  state.started = false;
  await page.goto("/");
  await expect(
    page.getByText(
      "Live collection is unavailable. Ask your instance administrator to configure Cloro.",
    ),
  ).toBeVisible();
  state.trackingFailed = true;
  await page.reload();
  const error = page.getByRole("alert").filter({ hasText: "Unable to load tracking status" });
  await expect(error.first()).toBeVisible({ timeout: 15000 });
  state.trackingFailed = false;
  await error.first().getByRole("button", { name: "Retry" }).click();
  await expect(error).toHaveCount(0);
});

test("run completion updates the open answer and stops rapid polling", async ({ page }) => {
  const { state, requests, result } = await mockWorkspace(page);
  await page.goto("/prompts/prompt-0?runId=run");
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByText("This answer is still being collected.")).toBeVisible();
  state.results = [result("chatgpt", "mentioned", "Koi is mentioned in this saved answer.")];
  state.run.state = "succeeded";
  await expect(dialog.getByText("Koi is mentioned in this saved answer.")).toBeVisible({
    timeout: 12000,
  });
  await expect(dialog.getByRole("button", { name: "Gemini Not collected" })).toBeVisible();
  await expect(
    dialog.getByRole("button", { name: "Google AI Overviews Not requested" }),
  ).toBeVisible();
  await dialog.getByRole("button", { name: "Gemini Not collected" }).click();
  await expect(
    dialog.getByText(
      "This run ended without collecting an answer for this engine. Check the next scheduled run.",
    ),
  ).toBeVisible();
  await expect(page).toHaveURL(/responseEngine=gemini/);
  const count = requests.filter((url) =>
    /visibility\/runs$|prompt-history/.test(url.pathname),
  ).length;
  await page.waitForTimeout(6000);
  expect(
    requests.filter((url) => /visibility\/runs$|prompt-history/.test(url.pathname)),
  ).toHaveLength(count);
});

test("failed and empty terminal answers never appear pending", async ({ page }) => {
  const { state, result } = await mockWorkspace(page);
  state.run.state = "failed";
  state.results = [result("chatgpt", "error"), result("gemini", "miss")];
  await page.goto("/prompts/prompt-0?runId=run");
  const dialog = page.getByRole("dialog");
  await expect(
    dialog.getByText("This answer could not be collected. Check the next scheduled run."),
  ).toBeVisible();
  await expect(dialog.getByText("This answer is still being collected.")).toHaveCount(0);
  await dialog.getByRole("button", { name: "Gemini Miss" }).click();
  await expect(dialog.getByText("No answer was returned.")).toBeVisible();
  await dialog.getByRole("button", { name: "Google AI Overviews Not requested" }).click();
  await expect(dialog.getByText("This engine was not included in this run.")).toBeVisible();
});

test("checkbox selection supports keyboard and mixed page selection", async ({ page }) => {
  const { state } = await mockWorkspace(page);
  state.started = false;
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto("/");
  const boxes = page.getByRole("checkbox").filter({ visible: true });
  await expect(boxes).toHaveCount(3);
  await boxes.nth(1).focus();
  await page.keyboard.press("Space");
  await expect(boxes.nth(1)).toBeChecked();
  await expect(boxes.first()).toHaveAttribute("aria-checked", "mixed");
  await boxes.first().click();
  await expect(boxes.nth(2)).toBeChecked();
  await page.getByRole("button", { name: "Clear selection" }).click();
  await expect(boxes.first()).not.toBeChecked();
});

test("competitor filters remain in the URL and reset the page", async ({ page }) => {
  await mockWorkspace(page);
  await page.goto("/competitors?page=2");
  await page.getByRole("combobox", { name: "Filter by status" }).click();
  await page.getByRole("option", { name: "Confirmed", exact: true }).click();
  await expect(page).toHaveURL(/status=confirmed/);
  expect(new URL(page.url()).searchParams.get("page")).not.toBe("2");
  await page.getByRole("combobox", { name: "Rows per page" }).click();
  await page.getByRole("option", { name: "50", exact: true }).click();
  await expect(page).toHaveURL(/pageSize=50/);
});

for (const theme of ["light", "dark"]) {
  test(`responsive surfaces and scroll access in ${theme} mode`, async ({ page }, testInfo) => {
    const { state } = await mockWorkspace(page);
    state.run.state = "succeeded";
    state.started = false;
    await page.addInitScript((value) => localStorage.setItem("koi:theme:v1", value), theme);
    for (const width of [1440, 960, 390, 320]) {
      await page.setViewportSize({ width, height: 1000 });
      for (const path of ["/", "/competitors", "/prompts/prompt-0"]) {
        await page.goto(path);
        await expect(page.locator("main")).toBeVisible();
        await expect(page.getByText("Loading…", { exact: true })).toHaveCount(0);
        await expect
          .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
          .toBe(true);
        if (path === "/competitors") {
          const row = page.getByRole("row").nth(1);
          await expect(row).toBeVisible();
          expect((await row.boundingBox())?.height).toBeLessThan(160);
        }
        if (path.includes("prompts/") && width <= 390) {
          const scroll = page.locator('[data-slot="table-container"][role="region"]');
          await expect(scroll).toBeVisible();
          await scroll.focus();
          await page.keyboard.press("ArrowRight");
          await expect
            .poll(() => scroll.evaluate((element) => element.scrollLeft))
            .toBeGreaterThan(0);
        }
        await page.screenshot({
          path: testInfo.outputPath(`${path.replaceAll("/", "-") || "home"}-${width}.png`),
          fullPage: true,
        });
      }
    }
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.goto("/settings/brand");
    const main = await page.locator("main").boundingBox();
    const header = await page.locator("header").boundingBox();
    expect(main?.x).toBe(header?.x);
    expect(main?.width).toBe(header?.width);
  });
}

test("switching brand stops polling the previous workspace", async ({ page }) => {
  const { requests } = await mockWorkspace(page);
  await page.goto("/");
  await page.getByLabel("Active brand: Koi", { exact: true }).click();
  await page.getByRole("option", { name: /Other brand/ }).click();
  await expect(page.getByLabel("Active brand: Other brand", { exact: true })).toBeVisible();
  const oldRequests = () => requests.filter((url) => url.searchParams.get("brandId") === "brand");
  const count = oldRequests().length;
  await page.waitForTimeout(6000);
  expect(oldRequests()).toHaveLength(count);
});

test("history refetch errors preserve the open response and allow retry", async ({ page }) => {
  const { state, result } = await mockWorkspace(page);
  state.results = [result("chatgpt", "mentioned", "Saved answer remains readable.")];
  await page.goto("/prompts/prompt-0?runId=run");
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByText("Saved answer remains readable.")).toBeVisible();
  state.failHistory = true;
  await expect(page.getByRole("alert", { includeHidden: true })).toContainText(
    "Could not load the saved answers.",
    { timeout: 12000 },
  );
  await expect(dialog.getByText("Saved answer remains readable.")).toBeVisible();
  await page.keyboard.press("Escape");
  state.failHistory = false;
  state.run.state = "succeeded";
  await page.getByRole("alert").getByRole("button", { name: "Retry" }).click();
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("keyboard dialog return, reduced motion, and enlarged layout", async ({ page }) => {
  const { state } = await mockWorkspace(page);
  state.started = false;
  state.run.state = "succeeded";
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto("/prompts/prompt-0");
  const trigger = page.getByRole("button", { name: "ChatGPT: Not collected", exact: true });
  await trigger.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(trigger).toBeFocused();
  const period = page.getByRole("button", { name: "7D", exact: true });
  await period.focus();
  await page.keyboard.down("Space");
  await expect(period).toHaveCSS("transform", "none");
  await page.keyboard.up("Space");
  await page.evaluate(() => {
    document.body.style.zoom = "2";
  });
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    .toBe(true);
  await expect(page.getByRole("button", { name: "90D", exact: true })).toBeVisible();
});

for (const rates of [
  [0, 0, 0],
  [100, 100, 100],
  [10, 50, 80],
  [10, null, 80],
]) {
  test(`summary chart style and hover remain consistent for ${rates.join(",")}`, async ({
    page,
  }, testInfo) => {
    const { state } = await mockWorkspace(page, rates, true);
    state.started = false;
    state.run.state = "succeeded";
    await page.setViewportSize({ width: 1440, height: 1000 });
    const appearances: unknown[] = [];
    for (const path of ["/?compareId=rival", "/prompts/prompt-0?compareId=rival"]) {
      await page.goto(path);
      const chart = page.locator('[data-slot="chart"]').first();
      const own = chart.locator('.recharts-line-curve[stroke-width="4"]').first();
      await expect(own).toHaveAttribute("d", /M/);
      // A clip on the exact Y domain removes half the stroke at 0 and 100.
      await expect(own).not.toHaveAttribute("clip-path", /.+/);
      await expect(own).toHaveCSS("stroke-width", "4px");
      appearances.push(
        await own.evaluate((element) => ({
          stroke: getComputedStyle(element).stroke,
          linecap: getComputedStyle(element).strokeLinecap,
        })),
      );
      const bounds = await chart.boundingBox();
      if (!bounds) throw new Error("Chart has no bounds");
      await page.mouse.move(bounds.x + bounds.width * 0.4, bounds.y + bounds.height / 2);
      await expect(chart.locator(".signal-active-marker")).toHaveCount(rates[1] === null ? 0 : 2);
      await expect(own).toHaveAttribute("stroke-opacity", "0.2");
      if (rates[1] !== null) {
        const marker = chart.locator(".signal-active-marker").first();
        await expect(marker).toHaveCSS("transition-duration", "0.14s");
        await page.emulateMedia({ reducedMotion: "reduce" });
        await expect(marker).toHaveCSS("transition-duration", "0s");
        await page.emulateMedia({ reducedMotion: "no-preference" });
      }
      await page.mouse.move(bounds.x + bounds.width * 0.7, bounds.y + bounds.height / 2);
      await expect(chart.locator(".signal-active-marker")).toHaveCount(2);
      await page.mouse.move(0, 0);
      await expect(chart.locator(".signal-active-marker")).toHaveCount(0);
      await expect(own).toHaveAttribute("stroke-opacity", "1");
      await expect(chart.locator("foreignObject")).toHaveCount(2);
      await page.screenshot({
        path: testInfo.outputPath(path.startsWith("/?") ? "summary.png" : "detail.png"),
        fullPage: true,
      });
    }
    expect(appearances[0]).toEqual(appearances[1]);
  });
}

for (const theme of ["light", "dark"]) {
  test(`header preserves visible geometry and extended targets in ${theme}`, async ({
    page,
  }, testInfo) => {
    const { state } = await mockWorkspace(page);
    state.started = false;
    await page.addInitScript((value) => localStorage.setItem("koi:theme:v1", value), theme);
    await page.goto("/");
    const account = page.getByRole("button", { name: "Account", exact: true });
    const competitor = page
      .locator("header")
      .getByRole("link", { name: "Competitors", exact: true });
    for (const control of [account, competitor]) {
      await expect(control).toHaveCSS("width", "32px");
      await expect(control).toHaveCSS("height", "32px");
      expect(await control.evaluate((element) => getComputedStyle(element, "::after").inset)).toBe(
        "-4px",
      );
    }
    await expect(account).toHaveCSS("background-color", "rgba(0, 0, 0, 0)");
    const box = await account.boundingBox();
    if (!box) throw new Error("Account control has no bounds");
    // Activate the transparent hit extension, outside the visible 32px circle.
    await page.mouse.click(box.x + box.width + 2, box.y + box.height / 2);
    await expect(page.getByRole("menuitem", { name: "Settings", exact: true })).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(account).toBeFocused();
    await page.mouse.move(0, 0);
    await page.locator("header").screenshot({ path: testInfo.outputPath("header.png") });
  });
}
