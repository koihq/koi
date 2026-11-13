import { execFileSync } from "node:child_process";
import { randomUUID } from "node:crypto";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

import { expect, type Page } from "@playwright/test";

export const PASSWORD = "password-e2e-1234";
export const DEMO = { email: "demo@koi.local", password: "KoiDemo2026!" };

const repo = fileURLToPath(new URL("../..", import.meta.url));
const koi = fileURLToPath(new URL("../.e2e/koi", import.meta.url));

/** A fresh account email, so journeys never share state. */
export function newEmail(label: string) {
  return `${label}-${randomUUID()}@e2e.koi.local`;
}

export async function signUp(page: Page, email: string) {
  await page.goto("/sign-up");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/setup/);
}

export async function signIn(page: Page, email: string, password: string) {
  await page.goto("/sign-in");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
}

/** Runs `koi admin` against the e2e server's database, as an instance owner would. */
export function koiAdmin(args: string[], stdin = "") {
  const databaseURL = readFileSync(new URL("../.e2e/database-url", import.meta.url), "utf8");

  return execFileSync(koi, ["admin", ...args], {
    cwd: repo,
    input: stdin,
    // Same mode as the e2e server, whatever the developer's .env says.
    env: { ...process.env, KOI_DATABASE_URL: databaseURL, KOI_DEPLOYMENT_MODE: "selfhosted" },
    encoding: "utf8",
  });
}
