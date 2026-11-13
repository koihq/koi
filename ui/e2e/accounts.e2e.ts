import { expect, type Page, test } from "@playwright/test";

import { koiAdmin, newEmail, PASSWORD, signIn, signUp } from "./support";

async function createBrand(page: Page, name: string, site: string) {
  await page.getByRole("link", { name: "Let's go" }).click();
  await page.getByLabel("Website or GitHub repo").fill(site);
  await page.getByRole("button", { name: "Continue" }).click();
  await page.getByLabel("Brand name").fill(name);
  await page.locator("#brand-country").fill("IT");
  await page.getByRole("option", { name: "IT (Italy)" }).click();
  await page.getByRole("button", { name: "See what AI says" }).click();
  await expect(page.getByLabel(`Active brand: ${name}`, { exact: true })).toBeVisible();
}

test("one organization never sees another's prompts", async ({ browser }) => {
  const owner = await browser.newPage();
  await signUp(owner, newEmail("first"));
  await createBrand(owner, "First", "first.example");

  await owner.getByRole("button", { name: "Add prompt" }).first().click();
  await owner.getByRole("dialog").getByRole("textbox").first().fill("A private question");
  await owner.getByRole("dialog").getByRole("button", { name: "Save prompt" }).click();
  await owner.getByRole("link", { name: "A private question" }).click();
  await expect(owner).toHaveURL(/\/prompts\/[0-9a-f-]+$/);
  const promptPath = new URL(owner.url()).pathname;
  const promptID = promptPath.split("/").at(-1);

  const other = await browser.newPage();
  await signUp(other, newEmail("second"));
  await createBrand(other, "Second", "second.example");

  const response = await other.request.get(`/api/v1/prompts/${promptID}`);
  expect(response.status()).toBe(404);

  await other.goto(promptPath);
  await expect(other.getByText("A private question")).toHaveCount(0);

  await owner.close();
  await other.close();
});

test("an owner signs out, signs back in, and recovers the account with koi admin", async ({
  page,
}) => {
  const email = newEmail("recover");
  await signUp(page, email);
  await createBrand(page, "Recover", "recover.example");

  await page.getByRole("button", { name: "Account" }).click();
  await page.getByRole("menuitem", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/sign-in/);

  await signIn(page, email, PASSWORD);
  await expect(page.getByLabel("Active brand: Recover", { exact: true })).toBeVisible();

  // Without email, the instance owner resets the password from the command line.
  const replacement = "replacement-password-5678";
  koiAdmin(["reset-password", email], `${replacement}\n`);

  await page.goto("/");
  await expect(page).toHaveURL(/\/sign-in/);

  await signIn(page, email, PASSWORD);
  await expect(page.getByText("Invalid email or password")).toBeVisible();

  await signIn(page, email, replacement);
  await expect(page.getByLabel("Active brand: Recover", { exact: true })).toBeVisible();
});
