import { expect, test } from "@playwright/test";

import { DEMO, koiAdmin, signIn } from "./support";

// The journey prepares its own data, as an owner would, instead of relying on server startup.
test.beforeAll(() => {
  koiAdmin(["seed-demo"]);
});

test("saved results lead from the dashboard to a brief for the founder's agent", async ({
  page,
}) => {
  await signIn(page, DEMO.email, DEMO.password);

  // The demo account opens on a brand with months of collected answers.
  const switcher = page.getByRole("combobox", { name: /^Active brand: / });
  await expect(switcher).toBeVisible();
  if (!(await switcher.getAttribute("aria-label"))?.includes("History")) {
    await switcher.click();
    await page.getByRole("option", { name: /History/ }).click();
  }
  await expect(page.getByLabel("Active brand: Koi · History", { exact: true })).toBeVisible();

  const visibility = page.getByRole("region", { name: "AI visibility" });
  await expect(visibility.locator(".recharts-surface").first()).toBeVisible();
  await expect(visibility.getByText(/\d+(\.\d)?%/).first()).toBeVisible();

  // A prompt's history opens a saved answer, which hands off as a brief.
  await page.locator("#prompts").getByRole("link").first().click();
  await expect(page).toHaveURL(/\/prompts\//);
  await page
    .getByRole("button", { name: /^Open the .+ run$/ })
    .first()
    .click();

  const answer = page.getByRole("dialog");
  await expect(answer.getByRole("group", { name: "Answers by engine" })).toBeVisible();
  await answer.getByRole("button", { name: "Copy for your agent" }).click();
  await expect(answer.getByRole("button", { name: "Copied" })).toBeVisible();

  const brief = await page.evaluate(() => navigator.clipboard.readText());
  expect(brief).toContain("Koi · History");
  expect(brief.length).toBeGreaterThan(200);
});
