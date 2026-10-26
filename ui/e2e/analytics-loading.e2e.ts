import { expect, type Page, test } from "@playwright/test";

async function mockProduct(page: Page) {
  const brands = Array.from({ length: 121 }, (_, index) => ({
    id: `brand-${index}`,
    organizationId: "org",
    name: `Brand ${String(index).padStart(3, "0")}`,
    domain: `brand${index}.example`,
    githubRepo: "",
    country: "IT",
    description: "",
  }));
  const metric = { completed: 4, cited: 1, mentioned: 1, rate: 50 };
  const series = ["2026-10-22", "2026-10-23"].map((date) => ({ date, metric }));
  const competitors = brands.map((_brand, index) => ({
    id: `competitor-${index}`,
    name: `Rival ${String(index).padStart(3, "0")}`,
    domain: `rival${index}.example`,
    metric: { ...metric, rate: 25 },
    series,
  }));
  const report = { metric, previous: metric, series, engines: {}, competitors };
  const state = {
    active: brands[120],
    failActivation: false,
    failSearch: false,
    failSummary: false,
    failPromptVisibility: false,
  };
  const requests: URL[] = [];
  const prompt = () => ({
    id: "prompt",
    brandId: state.active.id,
    text: "Which analytics tools should a founder consider?",
    source: "manual",
    confirmed: true,
    inactive: false,
    visibility: report,
  });
  const empty = {
    items: [],
    totalItems: 0,
    totalPages: 0,
    currentPage: 1,
    hasNext: false,
    hasPrev: false,
  };

  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    requests.push(url);
    let json: unknown;

    if (url.pathname === "/api/v1/auth/me") {
      json = {
        id: "user",
        email: "founder@example.com",
        organization: { id: "org", name: "Organization", role: "owner" },
        activeBrand: state.active,
      };
    } else if (url.pathname === "/api/v1/capabilities") {
      json = {
        cloudBilling: false,
        aiDrafting: false,
        liveCollection: false,
        competitorSearch: false,
      };
    } else if (url.pathname === "/api/v1/visibility/summary" && !state.failSummary) {
      json = { ...report, hasConfirmedPrompts: true };
    } else if (url.pathname === "/api/v1/tracking") {
      json = {
        promptsReviewed: true,
        competitorsReviewed: true,
        startedAt: null,
        nextRunAt: null,
        blockedReason: null,
      };
    } else if (url.pathname === "/api/v1/prompts") {
      json = { ...empty, items: [prompt()], totalItems: 1, totalPages: 1 };
    } else if (url.pathname === "/api/v1/prompts/prompt") {
      json = prompt();
    } else if (
      url.pathname === "/api/v1/prompt-history/prompt/visibility" &&
      !state.failPromptVisibility
    ) {
      json = report;
    } else if (
      url.pathname === "/api/v1/prompt-history/prompt/runs" ||
      url.pathname === "/api/v1/generations"
    ) {
      json = empty;
    } else if (url.pathname === "/api/v1/brands" && !state.failSearch) {
      const q = (url.searchParams.get("q") ?? "").toLowerCase();
      const currentPage = Number(url.searchParams.get("page"));
      const pageSize = Number(url.searchParams.get("pageSize"));
      const items = brands.filter((brand) =>
        `${brand.name} ${brand.domain}`.toLowerCase().includes(q),
      );
      const totalPages = Math.ceil(items.length / pageSize);
      json = {
        items: items.slice((currentPage - 1) * pageSize, currentPage * pageSize),
        totalItems: items.length,
        totalPages,
        currentPage,
        hasNext: currentPage < totalPages,
        hasPrev: currentPage > 1,
      };
    } else if (url.pathname.endsWith("/activate") && !state.failActivation) {
      state.active =
        brands.find((brand) => brand.id === url.pathname.split("/")[4]) ?? state.active;
      json = state.active;
    } else {
      await route.fulfill({
        status: 500,
        json: { error: { code: "internal", message: "internal error" } },
      });
      return;
    }

    await route.fulfill({ json });
  });

  return { requests, state };
}

test("home and prompt detail use complete reports without loading resource collections", async ({
  page,
}) => {
  const { requests } = await mockProduct(page);
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "AI visibility" })).toBeVisible();
  await expect(
    page.getByRole("link", {
      name: "Which analytics tools should a founder consider?",
      exact: true,
    }),
  ).toBeVisible();
  await expect(page.getByText("Rival 120", { exact: true })).toBeVisible();
  expect(requests.filter((url) => url.pathname === "/api/v1/brands")).toHaveLength(0);
  expect(requests.filter((url) => url.pathname === "/api/v1/competitors")).toHaveLength(0);
  expect(requests.filter((url) => url.pathname === "/api/v1/prompts")).toHaveLength(1);
  expect(requests.filter((url) => url.pathname === "/api/v1/visibility/summary")).toHaveLength(1);

  await page
    .getByRole("link", { name: "Which analytics tools should a founder consider?", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Which analytics tools should a founder consider?" }),
  ).toBeVisible();
  await expect(page.getByText("Rival 120", { exact: true })).toBeVisible();
  expect(requests.filter((url) => url.pathname === "/api/v1/brands")).toHaveLength(0);
  expect(requests.filter((url) => url.pathname === "/api/v1/competitors")).toHaveLength(0);
});

test("the brand selector fetches only requested pages and searches before switching", async ({
  page,
}) => {
  const { requests } = await mockProduct(page);
  await page.goto("/");
  await page.getByLabel("Active brand: Brand 120", { exact: true }).click();
  await expect(page.getByRole("option", { name: /Brand 000/ })).toBeVisible();
  const lists = () => requests.filter((url) => url.pathname === "/api/v1/brands");
  expect(lists()).toHaveLength(1);
  expect(lists()[0].searchParams.get("pageSize")).toBe("20");
  await page
    .locator('[data-slot="combobox-content"]')
    .getByRole("button", { name: "Next", exact: true })
    .click();
  await expect(page.getByRole("option", { name: /Brand 020/ })).toBeVisible();
  expect(lists()).toHaveLength(2);
  expect(lists()[1].searchParams.get("page")).toBe("2");

  await page.getByPlaceholder("Search brands").fill("Brand 105");
  await expect(page.getByRole("option", { name: /Brand 105/ })).toBeVisible();
  expect(lists()).toHaveLength(3);
  expect(lists()[2].searchParams.get("q")).toBe("Brand 105");
  expect(lists()[2].searchParams.get("page")).toBe("1");
  await page.getByPlaceholder("Search brands").press("ArrowDown");
  await page.getByPlaceholder("Search brands").press("Enter");
  await expect(page.getByLabel("Active brand: Brand 105", { exact: true })).toBeVisible();
  expect(requests.filter((url) => url.pathname.endsWith("/activate"))).toHaveLength(1);
  expect(lists()).toHaveLength(3);
});

test("a failed brand activation keeps the current brand and can be retried", async ({ page }) => {
  const { state } = await mockProduct(page);
  state.failActivation = true;
  await page.goto("/");
  await page.getByLabel("Active brand: Brand 120", { exact: true }).click();
  await page.getByRole("option", { name: /Brand 000/ }).click();
  await expect(page.getByRole("alert")).toHaveText(
    "Unable to switch brand. Open the selector to try again.",
  );
  await expect(page.getByLabel("Active brand: Brand 120", { exact: true })).toBeVisible();

  state.failActivation = false;
  await page.getByLabel("Active brand: Brand 120", { exact: true }).click();
  await page.getByRole("option", { name: /Brand 000/ }).click();
  await expect(page.getByLabel("Active brand: Brand 000", { exact: true })).toBeVisible();
});

test("an analytics error offers retry without showing a false empty state", async ({ page }) => {
  const { state } = await mockProduct(page);
  state.failSummary = true;
  await page.goto("/");
  await expect(page.getByRole("alert")).toContainText("Visibility could not be loaded.", {
    timeout: 15000,
  });
  await expect(
    page.getByText("Add and confirm prompts to see how your brand appears across AI answers."),
  ).toHaveCount(0);
  state.failSummary = false;
  await expect(page.getByText("Add competitors to compare their visibility.")).toHaveCount(0);
  await page.getByRole("alert").getByRole("button", { name: "Retry" }).click();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.getByText("Rival 120", { exact: true })).toBeVisible();
});

test("prompt analytics errors remain distinct from an empty report", async ({ page }) => {
  const { state } = await mockProduct(page);
  state.failPromptVisibility = true;
  await page.goto("/prompts/prompt");
  await expect(page.getByRole("alert")).toContainText("Visibility could not be loaded.", {
    timeout: 15000,
  });
  await expect(page.getByText("Add competitors to compare their visibility.")).toHaveCount(0);
  state.failPromptVisibility = false;
  await page.getByRole("alert").getByRole("button", { name: "Retry" }).click();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.getByText("Rival 120", { exact: true })).toBeVisible();
});

test("brand search handles no results and a recoverable API failure", async ({ page }) => {
  const { state } = await mockProduct(page);
  await page.goto("/");
  await page.getByLabel("Active brand: Brand 120", { exact: true }).click();
  await page.getByPlaceholder("Search brands").fill("Missing brand");
  await expect(page.getByText("No matching brands.")).toBeVisible();

  state.failSearch = true;
  await page.getByPlaceholder("Search brands").fill("Brand 100");
  await expect(page.getByText("Brands could not be loaded.")).toBeVisible({ timeout: 15000 });
  state.failSearch = false;
  await page
    .locator('[data-slot="combobox-content"]')
    .getByRole("button", { name: "Retry" })
    .click();
  await expect(page.getByRole("option", { name: /Brand 100/ })).toBeVisible();
});

test("brand selection remains usable at 320px", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 320, height: 740 });
  await mockProduct(page);
  await page.goto("/");
  await page.getByLabel("Active brand: Brand 120", { exact: true }).click();
  const popup = page.locator('[data-slot="combobox-content"]');
  await expect(popup.getByRole("option", { name: /Brand 000/ })).toBeVisible();
  await expect(popup.getByRole("button", { name: "Next", exact: true })).toBeVisible();
  const bounds = await popup.boundingBox();
  expect(bounds?.x).toBeGreaterThanOrEqual(0);
  expect((bounds?.x ?? 0) + (bounds?.width ?? 0)).toBeLessThanOrEqual(320);
  await page.screenshot({ path: testInfo.outputPath("brand-selector-mobile.png") });
  await popup.getByRole("option", { name: /Brand 000/ }).click();
  await expect(page.getByLabel("Active brand: Brand 000", { exact: true })).toBeVisible();
});
