import { expect, test } from "@playwright/test";

import { newEmail, signUp } from "./support";

test("a new self-hosted owner sets up a brand, starts tracking and reads the collected answers", async ({
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

  // Everything was saved. Collection runs through the e2e server's fake Cloro, never the real one.
  await page.goto("/");
  await expect(
    page.getByRole("link", { name: "Which payment tools suit a startup?" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Start tracking" }).click();

  // Each engine's answer cites acme.example, so every collected result is a citation.
  await page.getByRole("link", { name: "Which payment tools suit a startup?" }).click();
  await expect(page.getByRole("img", { name: "ChatGPT: Cited" }).first()).toBeVisible({
    timeout: 60_000,
  });

  await page
    .getByRole("button", { name: /^Open the .+ run$/ })
    .first()
    .click();
  const answer = page.getByRole("dialog");
  await expect(answer.getByText("Acme is a strong choice", { exact: false })).toBeVisible();
  await expect(answer.getByRole("link", { name: /acme\.example/ }).first()).toBeVisible();
});
