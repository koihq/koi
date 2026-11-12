import { expect, test } from "@playwright/test";

import { newEmail, signUp } from "./support";

test("a new self-hosted owner sets up a brand and tracks prompts without optional services", async ({
  page,
}) => {
  await signUp(page, newEmail("owner"));

  // Setup creates the brand from the website, with no drafting service to call.
  await page.getByRole("link", { name: "Let's go" }).click();
  await page.getByLabel("Website or GitHub repo").fill("acme.example");
  await page.getByRole("button", { name: "Continue" }).click();
  await page.getByLabel("Brand name").fill("Acme");
  await page.locator("#brand-country").fill("IT");
  await page.getByRole("option", { name: "IT (Italy)" }).click();
  await page.getByRole("button", { name: "See what AI says" }).click();

  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByLabel("Active brand: Acme", { exact: true })).toBeVisible();

  // Prompts and competitors are managed by hand.
  await page.getByRole("button", { name: "Add prompt" }).first().click();
  const promptDialog = page.getByRole("dialog");
  await promptDialog.getByRole("textbox").first().fill("Which payment tools suit a startup?");
  await promptDialog.getByRole("button", { name: "Save prompt" }).click();
  await expect(
    page.getByRole("link", { name: "Which payment tools suit a startup?" }),
  ).toBeVisible();

  await page.goto("/competitors");
  await page.getByRole("button", { name: "Add competitor" }).first().click();
  const competitorDialog = page.getByRole("dialog");
  await competitorDialog.getByRole("textbox", { name: "Name", exact: true }).fill("Rival");
  await competitorDialog
    .getByRole("textbox", { name: "Domain", exact: true })
    .fill("rival.example");
  await competitorDialog.getByRole("button", { name: "Save competitor" }).click();
  await expect(page.getByText("Rival", { exact: true }).first()).toBeVisible();

  // Everything was saved, and without the owner's Cloro key the instance says why nothing is
  // collected yet.
  await page.goto("/");
  await expect(
    page.getByRole("link", { name: "Which payment tools suit a startup?" }),
  ).toBeVisible();
  await expect(
    page.getByText(
      "Live collection is unavailable. Ask your instance administrator to configure Cloro.",
    ),
  ).toBeVisible();
});
