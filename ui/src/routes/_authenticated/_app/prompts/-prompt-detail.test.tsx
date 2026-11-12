import { http } from "msw";
import { describe, expect, test } from "vitest";
import { commands, page, userEvent } from "vitest/browser";
import { box, gate, opacity } from "@/test/dom";
import {
  competitorVisibility,
  metric,
  prompt,
  report,
  result,
  run,
  series,
  tracking,
} from "@/test/fixtures";
import { renderRoute } from "@/test/render-route";
import { serveWorkspace, type Workspace, worker, workspace } from "@/test/worker";

const TEXT = "First question about analytics tools";

function detail(overrides: Partial<Workspace> = {}): Workspace {
  return serveWorkspace(
    workspace({
      tracking: tracking(),
      prompts: [prompt({ text: TEXT, visibility: report() })],
      runs: [run()],
      ...overrides,
    }),
  );
}

const dialog = () => page.getByRole("dialog");
const panel = () => page.getByRole("region", { name: "AI visibility" });
const engine = (name: string) => dialog().getByRole("button", { name: new RegExp(name) });
const outcome = (label: string) =>
  dialog().getByRole("group", { name: "Answers by engine" }).getByRole("img", { name: label });
const openRun = () => page.getByRole("button", { name: /^Open the .+ run$/ });

const historyPolls = (state: Workspace) =>
  state.requests.filter((url) => /\/runs$|prompt-history/.test(url.pathname)).length;

describe("prompt detail", () => {
  test("run completion updates the open answer and stops rapid polling", async () => {
    const state = detail();

    const { router } = await renderRoute("/prompts/prompt-0?runId=run");

    await expect.element(dialog().getByText("This answer is still being collected.")).toBeVisible();

    state.results.run = [
      result({ engine: "chatgpt", outcome: "mentioned", rawAnswer: "Koi is mentioned here." }),
    ];
    state.runs = [run({ state: "succeeded", completedResults: 1, finishedAt: run().createdAt })];

    await expect
      .element(dialog().getByText("Koi is mentioned here."), { timeout: 12_000 })
      .toBeVisible();
    await expect.element(outcome("ChatGPT: Mentioned")).toBeVisible();
    await expect.element(outcome("Gemini: Not collected")).toBeVisible();
    await expect.element(outcome("Google AI Overviews: Not requested")).toBeVisible();

    await engine("Gemini").click();

    await expect
      .element(
        dialog().getByText(
          "This run ended without collecting an answer for this engine. Check the next scheduled run.",
        ),
      )
      .toBeVisible();
    expect(router.state.location.search).toMatchObject({ responseEngine: "gemini" });

    const count = historyPolls(state);
    await new Promise((resolve) => setTimeout(resolve, 6_000));
    expect(historyPolls(state)).toBe(count);
  });

  test("failed and empty terminal answers never appear pending", async () => {
    detail({
      runs: [run({ state: "failed" })],
      results: {
        run: [
          result({ engine: "chatgpt", outcome: "error" }),
          result({ engine: "gemini", outcome: "miss" }),
        ],
      },
    });

    await renderRoute("/prompts/prompt-0?runId=run");

    await expect
      .element(
        dialog().getByText("This answer could not be collected. Check the next scheduled run."),
      )
      .toBeVisible();
    await expect
      .element(dialog().getByText("This answer is still being collected."))
      .not.toBeInTheDocument();

    await expect.element(outcome("Gemini: Miss")).toBeVisible();
    await engine("Gemini").click();
    await expect.element(dialog().getByText("No answer was returned.")).toBeVisible();

    await expect.element(outcome("Google AI Overviews: Not requested")).toBeVisible();
    await engine("Google AI Overviews").click();
    await expect
      .element(dialog().getByText("This engine was not included in this run."))
      .toBeVisible();
  });

  test("history refetch errors preserve the open response and allow retry", async () => {
    const state = detail({
      results: {
        run: [result({ engine: "chatgpt", rawAnswer: "Saved answer remains readable." })],
      },
    });

    await renderRoute("/prompts/prompt-0?runId=run");
    await expect.element(dialog().getByText("Saved answer remains readable.")).toBeVisible();

    state.fail.add("history");

    // The modal dialog hides the page behind it from the accessibility tree.
    const error = page
      .getByRole("alert", { includeHidden: true })
      .filter({ hasText: "Could not load the saved answers." });
    await expect.element(error, { timeout: 12_000 }).toBeInTheDocument();
    await expect.element(dialog().getByText("Saved answer remains readable.")).toBeVisible();

    await userEvent.keyboard("{Escape}");
    await expect.element(dialog()).not.toBeInTheDocument();

    const visibleError = page
      .getByRole("alert")
      .filter({ hasText: "Could not load the saved answers." });
    const retry = visibleError.getByRole("button", { name: "Retry" });
    await expect.element(retry).toBeVisible();

    // Recover and retry in one tick, so a background poll cannot clear the error first.
    state.fail.delete("history");
    state.runs = [run({ state: "succeeded" })];
    (retry.element() as HTMLButtonElement).click();

    await expect.element(visibleError).not.toBeInTheDocument();
  });

  test("prompt analytics errors remain distinct from an empty report", async () => {
    const state = detail({ tracking: tracking({ startedAt: null }) });
    state.fail.add("promptVisibility");

    await renderRoute("/prompts/prompt-0");

    const error = page.getByRole("alert").filter({ hasText: "Visibility could not be loaded." });
    await expect.element(error).toBeVisible();
    await expect
      .element(page.getByText("Add competitors to compare their visibility."))
      .not.toBeInTheDocument();

    state.fail.delete("promptVisibility");
    await error.getByRole("button", { name: "Retry" }).click();

    await expect.element(error).not.toBeInTheDocument();
  });

  test("keyboard dialog return, reduced motion, and enlarged layout", async () => {
    await page.viewport(1440, 1000);
    await commands.emulateMedia({ reducedMotion: "reduce" });
    detail({ tracking: tracking({ startedAt: null }), runs: [run({ state: "succeeded" })] });

    try {
      await renderRoute("/prompts/prompt-0");

      const trigger = openRun();
      await expect.element(trigger).toBeVisible();
      trigger.element().focus();
      await userEvent.keyboard("{Enter}");
      await expect.element(dialog()).toBeVisible();

      await userEvent.keyboard("{Escape}");
      await expect.element(trigger).toHaveFocus();

      const period = page.getByRole("button", { name: "7D", exact: true });
      await expect.element(period).toBeVisible();
      period.element().focus();
      await userEvent.keyboard("{Space>}");
      expect(getComputedStyle(period.element()).transform).toBe("none");
      await userEvent.keyboard("{/Space}");

      document.body.style.zoom = "2";
      await expect.poll(() => document.documentElement.scrollWidth <= window.innerWidth).toBe(true);
      await expect.element(page.getByRole("button", { name: "90D", exact: true })).toBeVisible();
    } finally {
      document.body.style.zoom = "";
      await commands.emulateMedia({ reducedMotion: "no-preference" });
    }
  });

  test.each([
    1440, 960, 390, 320,
  ])("retains chart geometry while filters load at %ipx", async (width) => {
    await page.viewport(width, 900);
    if (width === 320) await commands.emulateMedia({ reducedMotion: "reduce" });
    detail({ tracking: tracking({ startedAt: null }), runs: [run({ state: "succeeded" })] });

    try {
      await renderRoute("/prompts/prompt-0", {
        theme: width === 960 || width === 320 ? "dark" : "light",
      });

      const section = panel();
      const range = section.getByRole("button", { name: "7D", exact: true });

      await expect.element(section).toHaveAttribute("aria-busy", "false");
      await expect.poll(() => section.element().querySelector(".recharts-surface")).toBeTruthy();

      const height = box(section).height;
      const rangeTop = box(range).y;
      const hold = gate();

      worker.use(http.get("/api/v1/prompt-history/:id/visibility", () => hold.opened));

      try {
        await range.click();

        await expect.element(section).toHaveAttribute("aria-busy", "true");
        await expect.poll(() => opacity(section.element().querySelector("figure"))).toBe(0.5);
        expect(opacity(range.element())).toBe(1);
        expect(section.element().querySelector(".recharts-surface")).toBeTruthy();
        expect(box(section).height).toBe(height);
        expect(box(range).y).toBe(rangeTop);
      } finally {
        hold.release();
      }

      await expect.element(section).toHaveAttribute("aria-busy", "false");
      await expect.poll(() => opacity(section.element().querySelector("figure"))).toBe(1);
      await expect.element(range).toHaveAttribute("aria-pressed", "true");
    } finally {
      await commands.emulateMedia({ reducedMotion: "no-preference" });
    }
  });

  test("compares the prompt with a competitor on the same scale", async () => {
    const days = series([10, 50, 80]);
    detail({
      tracking: tracking({ startedAt: null }),
      runs: [run({ state: "succeeded" })],
      promptVisibility: report({
        metric: metric({ rate: 80 }),
        series: days,
        competitors: [competitorVisibility({ series: series([90, 50, 20]) })],
      }),
    });

    await renderRoute("/prompts/prompt-0?compareId=rival");

    await expect
      .poll(() => panel().element().querySelectorAll(".recharts-line-curve").length)
      .toBeGreaterThan(1);
  });
});
