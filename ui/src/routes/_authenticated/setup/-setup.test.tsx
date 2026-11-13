import { describe, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";
import { cleanup } from "vitest-browser-react";
import { css, fitsViewport } from "@/test/dom";
import { credential } from "@/test/fixtures";
import { addBrand, DRAFTED_PROMPT, serveOnboarding } from "@/test/onboarding";
import { renderRoute } from "@/test/render-route";

const heading = (name: string) => page.getByRole("heading", { name });
const step = (text: string) => page.getByText(text, { exact: true });

async function chooseCountry() {
  await (await css("#brand-country")).fill("IT");
  await page.getByRole("option", { name: "IT (Italy)" }).click();
}

describe("setup", () => {
  test("takes a person from the choice to a real answer before any payment", async () => {
    const state = serveOnboarding();

    const { router } = await renderRoute("/setup");

    await expect.element(heading("Let's find out what AI says about you.")).toBeVisible();
    await expect.element(step("Step 1 of 4")).toBeVisible();
    await expect.element(page.getByRole("link", { name: "Back" })).not.toBeInTheDocument();
    await expect.element(page.getByRole("radio", { name: /Show me here/ })).toBeChecked();
    await page.getByRole("link", { name: "Let's go" }).click();

    // The work email already names the site.
    await expect.element(heading("What's your website?")).toBeVisible();
    await expect.element(step("Step 2 of 4")).toBeVisible();
    const site = page.getByLabelText("Website or GitHub repo");
    await expect.element(site).toHaveValue("acme.com");

    await site.fill("not a site");
    await page.getByRole("button", { name: "Continue" }).click();
    await expect
      .element(page.getByText("Enter a website or a GitHub repository URL"))
      .toBeVisible();

    await site.fill("https://www.acme.com/pricing");
    await page.getByRole("button", { name: "Continue" }).click();

    // The name comes from the site, says so, and stays editable.
    await expect.element(heading("Is this you?")).toBeVisible();
    await expect.element(step("Step 3 of 4")).toBeVisible();
    await expect.element(page.getByLabelText("Brand name")).toHaveValue("Acme Pay");
    await expect.element(page.getByText(/^From (www\.)?acme\.com$/)).toBeVisible();
    await chooseCountry();

    // Going back keeps what was typed.
    await page.getByRole("link", { name: "Back" }).click();
    await expect.element(site).toHaveValue("https://www.acme.com/pricing");
    await page.getByRole("button", { name: "Continue" }).click();
    await chooseCountry();
    await page.getByRole("button", { name: "See what AI says" }).click();

    // While it waits, the page explains what it is doing, with navigation hidden.
    await expect.element(step("Step 4 of 4")).toBeVisible();
    await expect.element(heading("Checking how AI answers about Acme Pay")).toBeVisible();
    await expect
      .element(page.getByText("You can leave this page.", { exact: false }))
      .toBeVisible();
    await expect.element(page.getByText("Payments for startups")).toBeVisible();
    await expect
      .element(
        page.getByText("Asking ChatGPT, Gemini, Google AI Mode, and Google AI Overviews").first(),
      )
      .toBeVisible();
    await expect.element(page.getByText("0 of 20 answers in")).toBeVisible();
    await expect
      .element(page.getByRole("combobox", { name: /Active brand/ }))
      .not.toBeInTheDocument();

    await page.getByRole("button", { name: "Account" }).click();
    const signOut = page.getByRole("menuitem", { name: "Sign out" });
    await expect.element(signOut).toBeVisible();
    await expect.element(page.getByRole("menuitem", { name: "Billing" })).not.toBeInTheDocument();
    await userEvent.keyboard("{Escape}");
    await expect.element(signOut).not.toBeInTheDocument();

    expect(state.autoStart).toBe(true);
    expect(state.starts).toBe(1);

    state.completed = 2;
    await expect
      .element(heading("ChatGPT recommends Stripe instead of Acme Pay"), { timeout: 10_000 })
      .toBeVisible();
    await expect
      .element(page.getByText("Stripe is the most common choice", { exact: false }).first())
      .toBeVisible();
    await expect.element(page.getByRole("link", { name: /stripe\.com/ }).first()).toBeVisible();
    await expect.element(page.getByText("1 of 2 answers mention Acme Pay")).toBeInTheDocument();

    // The price and the way on appear with the first answers.
    await expect.element(heading("€0 today")).toBeVisible();
    await expect
      .element(page.getByText("Then €29 a month after 7 days", { exact: false }))
      .toBeVisible();

    // Choosing another engine shows its answer without moving the headline.
    await (await css("fieldset")).getByRole("button", { name: /Google AI Mode/ }).click();
    await expect.element(heading("ChatGPT recommends Stripe instead of Acme Pay")).toBeVisible();
    await expect
      .element(page.getByRole("link", { name: "Keep an eye on this" }))
      .toHaveAttribute("href", "/settings/billing");
    await expect.element(heading("Where to start")).not.toBeInTheDocument();

    state.runState = "succeeded";
    await expect.element(heading("Where to start"), { timeout: 70_000 }).toBeVisible();
    await expect
      .element(page.getByText("Stripe came up in 1 of 1 answers", { exact: false }))
      .toBeVisible();
    await expect.element(page.getByRole("link", { name: "See your competitors" })).toBeVisible();

    await page.getByRole("button", { name: "Fix this with my agent" }).click();
    await expect
      .element(page.getByText("Paste it into your agent.", { exact: false }))
      .toBeVisible();
    expect(await navigator.clipboard.readText()).toContain("visibility brief for Acme Pay");

    expect(state.starts).toBe(1);
    expect(state.workspace.steps).toEqual(
      expect.arrayContaining([
        "choice",
        "website",
        "confirm",
        "first_look",
        "first_answer",
        "agent_handoff",
      ]),
    );
    expect(router.state.location.pathname).toBe("/");
  }, 120_000);

  test("does not claim a site named the brand when the lookup found nothing", async () => {
    serveOnboarding();

    await renderRoute("/setup?path=koi&site=unknown-brand.com");

    await expect.element(page.getByLabelText("Brand name")).toHaveValue("Unknown-brand");
    await expect.element(page.getByText("From unknown-brand.com")).not.toBeInTheDocument();
  });

  test("waits for the agent, then moves to its check without starting one", async () => {
    const state = serveOnboarding();

    const { router } = await renderRoute("/setup");

    await page.getByRole("radio", { name: /Show me here/ }).click();
    await userEvent.keyboard("{ArrowRight}");
    await expect.element(page.getByRole("radio", { name: /Let my agent handle it/ })).toBeChecked();
    await page.getByRole("link", { name: "Let's go" }).click();

    await expect.element(step("Step 2 of 4")).toBeVisible();
    await page.getByRole("button", { name: "Claude Code" }).click();
    await expect
      .element(page.getByText("claude mcp add --transport http koi", { exact: false }))
      .toBeVisible();
    await expect
      .element(page.getByText("npx skills add koihq/koi", { exact: false }))
      .toBeVisible();

    // Signing in moves on by itself.
    state.workspace.credentials = [credential()];
    await expect.element(step("Step 3 of 4"), { timeout: 10_000 }).toBeVisible();
    await expect.element(page.getByRole("button", { name: "Copy prompt" })).toBeVisible();
    await expect.element(page.getByText("Claude Code's here.", { exact: false })).toBeVisible();

    addBrand(state);
    await expect.poll(() => router.state.location.pathname, { timeout: 10_000 }).toBe("/");

    // The agent may still be correcting its prompts, so the page must not spend the check.
    // Nothing has been asked yet, so this is not the last step, and there is nothing to hand back.
    await expect.element(page.getByRole("button", { name: "Run my free check" })).toBeVisible();
    await expect
      .element(page.getByText("Your agent drafted these.", { exact: false }))
      .toBeVisible();
    await expect.element(step("Step 3 of 4")).toBeVisible();
    await expect.element(page.getByRole("link", { name: DRAFTED_PROMPT })).toBeVisible();
    await expect.element(page.getByRole("listitem").filter({ hasText: "Stripe" })).toBeVisible();
    await expect.element(page.getByRole("link", { name: "Review competitors" })).toBeVisible();
    await expect
      .element(page.getByRole("button", { name: "Fix this with my agent" }))
      .not.toBeInTheDocument();
    expect(state.starts).toBe(0);
  });

  test("offers manual prompts and the trial when a free draft is unavailable", async () => {
    const state = serveOnboarding();
    state.draftBlocked = true;

    const { router } = await renderRoute("/setup?path=koi&site=acme.com");
    await page.getByRole("button", { name: "See what AI says" }).click();

    await expect.element(heading("We couldn't draft your prompts for free today")).toBeVisible();
    expect(router.state.location.search).toMatchObject({ draftStart: "unavailable" });
    await expect.element(page.getByRole("link", { name: "Keep an eye on this" })).toBeVisible();
    await expect.element(page.getByText("drafting didn’t start")).not.toBeInTheDocument();
  });

  test("hovering a link does not ask the server for the account again", async () => {
    const state = serveOnboarding();

    await renderRoute("/setup");

    const proceed = page.getByRole("link", { name: "Let's go" });
    await expect.element(proceed).toBeVisible();
    const accountCalls = () =>
      state.workspace.requests.filter((url) => url.pathname === "/api/v1/auth/me").length;
    const before = accountCalls();

    await proceed.hover();
    await userEvent.unhover(proceed);
    await proceed.hover();
    await new Promise((resolve) => setTimeout(resolve, 500));

    expect(accountCalls()).toBe(before);
  });

  test("skips the choice when no agent can connect", async () => {
    serveOnboarding({ mcpUrl: "" });

    await renderRoute("/setup");

    await expect.element(heading("What's your website?")).toBeVisible();
    await expect.element(step("Step 1 of 3")).toBeVisible();
    await expect.element(page.getByRole("link", { name: "Back" })).not.toBeInTheDocument();
  });

  test("keeps every setup view inside a narrow viewport", async () => {
    await page.viewport(320, 740);
    serveOnboarding();

    for (const path of [
      "/setup",
      "/setup?path=koi",
      "/setup?path=koi&site=acme.com",
      "/setup?path=agent",
      "/setup?path=agent&stage=prompt",
    ]) {
      await renderRoute(path);

      await expect.element(page.getByRole("heading", { level: 1 })).toBeVisible();
      await expect.poll(fitsViewport, { message: path }).toBe(true);

      await cleanup();
    }
  });
});
