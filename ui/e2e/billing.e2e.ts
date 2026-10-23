import { expect, type Page, test } from "@playwright/test";

const catalog = {
  currency: "eur",
  interval: "month",
  options: [
    { prompts: 5, amountCents: 900, currency: "eur", interval: "month" },
    { prompts: 25, amountCents: 2900, currency: "eur", interval: "month" },
    { prompts: 50, amountCents: 4900, currency: "eur", interval: "month" },
  ],
};

const active = {
  organizationId: "org_1",
  quantity: 25,
  status: "active",
  cancelAtPeriodEnd: false,
  pendingQuantity: null as number | null,
  pendingUpgradeQuantity: null as number | null,
  pendingUpgradeExpiresAt: null as string | null,
  paymentUrl: "",
};

async function mockAccount(page: Page, subscription = active) {
  const state = { subscription, promptLimit: 25, trialAvailable: false };

  await page.route("**/api/v1/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    let json: unknown;
    switch (path) {
      case "/api/v1/auth/me":
        json = {
          id: "user_1",
          email: "founder@example.com",
          organization: { id: "org_1", name: "Koi" },
          activeBrand: {
            id: "brand_1",
            organizationId: "org_1",
            name: "Koi",
            domain: "koi.example",
            githubRepo: "",
            country: "IT",
            description: "",
          },
        };
        break;
      case "/api/v1/brands":
        json = {
          items: [{ id: "brand_1", name: "Koi", domain: "koi.example", country: "IT" }],
          totalItems: 1,
          totalPages: 1,
          currentPage: 1,
          hasNext: false,
          hasPrev: false,
        };
        break;
      case "/api/v1/capabilities":
        json = {
          cloudBilling: true,
          aiDrafting: false,
          liveCollection: true,
          competitorSearch: false,
        };
        break;
      case "/api/v1/billing/catalog":
        json = catalog;
        break;
      case "/api/v1/billing/subscription":
        json = state;
        break;
      default:
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

test("starts from the current allowance and lets a pending downgrade be canceled", async ({
  page,
}, testInfo) => {
  const state = await mockAccount(page);
  await page.goto("/settings/billing");
  await expect(page.getByRole("combobox", { name: "Tracked prompts" })).toHaveText("25");
  await expect(page.getByRole("button", { name: "Change allowance" })).toBeDisabled();

  await page.screenshot({ path: testInfo.outputPath("billing-desktop.png"), fullPage: true });
  state.subscription.pendingQuantity = 5;
  await page.reload();
  await expect(page.getByRole("combobox", { name: "Tracked prompts" })).toHaveText("5");
  await page.route("**/api/v1/billing/change-allowance", async (route) => {
    expect(route.request().postDataJSON()).toEqual({ prompts: 25 });
    expect(route.request().headers()["idempotency-key"]).toBeTruthy();
    state.subscription.pendingQuantity = null;
    await route.fulfill({ json: { quantity: 25, effective: "current" } });
  });
  await page.getByRole("button", { name: "Keep current allowance" }).click();
  await expect(page.getByText("Your current allowance has been kept.")).toBeVisible();
  await expect(page.getByRole("combobox", { name: "Tracked prompts" })).toHaveText("25");
});

test("shows payment recovery and updates after confirmation", async ({ page }) => {
  const state = await mockAccount(page, {
    ...active,
    pendingUpgradeQuantity: 50,
    paymentUrl: "https://invoice.stripe.com/payment",
  });
  await page.goto("/settings/billing");
  await expect(page.getByRole("link", { name: "Complete payment" })).toHaveAttribute(
    "href",
    "https://invoice.stripe.com/payment",
  );
  await expect(page.getByRole("button", { name: "Change allowance" })).toBeDisabled();

  state.subscription = { ...active, quantity: 50 };
  state.promptLimit = 50;
  await expect(page.getByText("50 tracked prompts · Active")).toBeVisible();
  await expect(page.getByRole("link", { name: "Complete payment" })).toHaveCount(0);
});

test("stops waiting automatically and offers a manual refresh", async ({ page }) => {
  const state = await mockAccount(page);
  state.promptLimit = 0;
  await page.clock.install();
  await page.goto("/settings/billing?checkout=success");
  await expect(
    page.getByText("Your payment or subscription change is being confirmed.", { exact: false }),
  ).toBeVisible();
  await page.clock.fastForward(61_000);
  await expect(page.getByRole("button", { name: "Refresh status" })).toBeVisible();
  state.promptLimit = 25;
  await page.getByRole("button", { name: "Refresh status" }).click();
  await expect(page.getByRole("link", { name: "Continue to prompts" })).toBeVisible();
});

test("preserves useful billing errors and keeps the retry key", async ({ page }) => {
  await mockAccount(page, { ...active, pendingQuantity: 5 });
  const keys: string[] = [];
  await page.route("**/api/v1/billing/change-allowance", async (route) => {
    keys.push(route.request().headers()["idempotency-key"]);
    await route.fulfill({
      status: 409,
      json: { error: { code: "billing_conflict", message: "billing_conflict" } },
    });
  });
  await page.goto("/settings/billing");
  await page.getByRole("button", { name: "Keep current allowance" }).click();
  await expect(page.getByRole("alert")).toContainText("Open billing management");
  await page.getByRole("button", { name: "Keep current allowance" }).click();
  await expect.poll(() => keys.length).toBe(2);
  expect(keys[0]).toBe(keys[1]);
});

test("keeps controls visible on a narrow viewport", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 320, height: 740 });
  await mockAccount(page, { ...active, pendingQuantity: 5 });
  await page.goto("/settings/billing");
  await expect(page.getByRole("button", { name: "Keep current allowance" })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(320);
  await page.screenshot({ path: testInfo.outputPath("billing-mobile.png"), fullPage: true });
});

test("changes the future allowance during trial", async ({ page }) => {
  const state = await mockAccount(page, { ...active, status: "trialing" });
  state.promptLimit = 5;
  await page.route("**/api/v1/billing/change-allowance", async (route) => {
    expect(route.request().postDataJSON()).toEqual({ prompts: 50 });
    state.subscription.quantity = 50;
    await route.fulfill({ status: 202, json: { quantity: 50, effective: "trial_end" } });
  });
  await page.goto("/settings/billing");
  await expect(
    page.getByText("Your trial end date and 5-prompt allowance stay the same.", { exact: false }),
  ).toBeVisible();
  await page.getByRole("combobox", { name: "Tracked prompts" }).click();
  await page.getByRole("option", { name: "50", exact: true }).click();
  await page.getByRole("button", { name: "Change allowance" }).click();
  await expect(
    page.getByText("Your selected allowance will start when your trial ends."),
  ).toBeVisible();
  await expect(
    page.getByText("Tracking allowance now: 5 active prompts across all brands."),
  ).toBeVisible();
});

test("explains an expired upgrade and allows another attempt", async ({ page }) => {
  await mockAccount(page, {
    ...active,
    pendingUpgradeQuantity: 50,
    pendingUpgradeExpiresAt: new Date(Date.now() - 1000).toISOString(),
    paymentUrl: "https://invoice.stripe.com/expired",
  });
  await page.goto("/settings/billing");
  await expect(page.getByText("The upgrade payment expired.", { exact: false })).toBeVisible();
  await expect(page.getByRole("link", { name: "Complete payment" })).toHaveCount(0);
  await page.getByRole("combobox", { name: "Tracked prompts" }).click();
  await page.getByRole("option", { name: "50", exact: true }).click();
  await expect(page.getByRole("button", { name: "Change allowance" })).toBeEnabled();
});
