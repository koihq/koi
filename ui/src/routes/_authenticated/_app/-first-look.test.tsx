import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";

import { addBrand, DRAFTED_PROMPT, serveOnboarding } from "@/test/onboarding";
import { renderRoute } from "@/test/render-route";

describe("first look", () => {
  test.each([
    ["first_look_site_used", "This site has used its free checks", "Each website or repo gets 3."],
    ["first_look_paused", "Free checks are paused for today", "Yours starts by itself"],
  ] as const)("explains %s and offers the trial", async (reason, title, description) => {
    const state = serveOnboarding();
    addBrand(state);
    state.started = true;
    state.blocked = reason;

    await renderRoute("/");

    await expect.element(page.getByRole("heading", { name: title })).toBeVisible();
    await expect.element(page.getByText(description, { exact: false })).toBeVisible();
    await expect.element(page.getByRole("link", { name: "Keep an eye on this" })).toBeVisible();
    await expect.element(page.getByRole("link", { name: DRAFTED_PROMPT })).toBeVisible();
    await expect.element(page.getByRole("link", { name: "Review competitors" })).toBeVisible();
    await expect.element(page.getByText(/Step \d of \d/)).not.toBeInTheDocument();
    await expect
      .element(page.getByRole("button", { name: "Fix this with my agent" }))
      .not.toBeInTheDocument();
  });

  test.each([
    ["", "Koi tries again for free within a day"],
    ["first_look_used", "Start your trial to run the next check automatically"],
  ] as const)('explains a check without answers when blocked is "%s"', async (blocked, description) => {
    const state = serveOnboarding();
    addBrand(state);
    state.started = true;
    state.blocked = blocked;
    state.runState = "failed";
    state.noAnswers = true;

    await renderRoute("/");

    await expect
      .element(page.getByRole("heading", { name: "Koi couldn't finish this check" }))
      .toBeVisible();
    await expect.element(page.getByText(description, { exact: false })).toBeVisible();
  });
});
