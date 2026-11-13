import { http } from "msw";
import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";
import { box, css, gate, layoutSettled, opacity } from "@/test/dom";
import {
  brand,
  competitorVisibility,
  me,
  metric,
  prompt,
  report,
  series,
  tracking,
} from "@/test/fixtures";
import { renderRoute } from "@/test/render-route";
import { serveWorkspace, type Workspace, worker, workspace } from "@/test/worker";

const PROMPT = "Which analytics tools should a founder consider?";

/** 121 brands, the last one active, each with a rival in the visibility report. */
function analytics(): Workspace {
  const brands = Array.from({ length: 121 }, (_, index) =>
    brand({
      id: `brand-${index}`,
      name: `Brand ${String(index).padStart(3, "0")}`,
      domain: `brand${index}.example`,
    }),
  );
  const days = series([50, 50]);
  const visibility = report({
    series: days,
    competitors: brands.map((_brand, index) =>
      competitorVisibility({
        id: `competitor-${index}`,
        name: `Rival ${String(index).padStart(3, "0")}`,
        domain: `rival${index}.example`,
        series: days,
      }),
    ),
  });

  return serveWorkspace(
    workspace({
      me: me({ activeBrand: brands[120] }),
      brands,
      tracking: tracking({ startedAt: null }),
      summary: visibility,
      promptVisibility: visibility,
      prompts: [prompt({ id: "prompt", brandId: "brand-120", text: PROMPT, visibility })],
    }),
  );
}

const requestsTo = (state: Workspace, path: string) =>
  state.requests.filter((url) => url.pathname === path);

const signal = () => page.getByRole("region", { name: "AI visibility" });
const rivals = () => page.getByRole("complementary", { name: "Recommended instead" });
const promptLink = () => page.getByRole("link", { name: PROMPT, exact: true });

describe("home", () => {
  test("uses complete reports without loading resource collections", async () => {
    const state = analytics();

    await renderRoute("/");

    await expect.element(page.getByRole("heading", { name: "AI visibility" })).toBeVisible();
    await expect.element(promptLink()).toBeVisible();
    await expect.element(page.getByText("Rival 120", { exact: true })).toBeVisible();
    expect(requestsTo(state, "/api/v1/brands")).toHaveLength(0);
    expect(requestsTo(state, "/api/v1/competitors")).toHaveLength(0);
    // One page for the list and one count per status tab, all for the active brand only.
    const prompts = requestsTo(state, "/api/v1/prompts");
    expect(prompts).toHaveLength(4);
    expect(new Set(prompts.map((url) => url.search)).size).toBe(4);
    expect(prompts.every((url) => url.searchParams.get("brandId") === "brand-120")).toBe(true);
    expect(requestsTo(state, "/api/v1/visibility/summary")).toHaveLength(1);

    await promptLink().click();

    await expect.element(page.getByRole("heading", { name: PROMPT })).toBeVisible();
    await expect.element(page.getByText("Rival 120", { exact: true })).toBeVisible();
    expect(requestsTo(state, "/api/v1/brands")).toHaveLength(0);
    expect(requestsTo(state, "/api/v1/competitors")).toHaveLength(0);
  });

  test.each([
    "back link",
    "direct url",
  ])("summary filters keep their position after a %s to the prompt list", async (arrival) => {
    analytics();

    const { router } = await renderRoute(arrival === "back link" ? "/" : "/#prompts");

    if (arrival === "back link") {
      await promptLink().click();
      await page.getByRole("link", { name: "All prompts" }).click();
      await expect.poll(() => router.state.location.pathname).toBe("/");
      // The link still leads to the list.
      await expect.poll(() => window.scrollY).toBeGreaterThan(0);
    }

    await expect.element(page.getByRole("heading", { name: "AI visibility" })).toBeVisible();

    const controls = [
      rivals().getByRole("button", { name: /Rival 000/ }),
      rivals().getByRole("button", { name: /Rival 000/ }),
      rivals().getByRole("button", { name: /Rival 003/ }),
      signal().getByRole("button", { name: "7D", exact: true }),
      signal().getByRole("button", { name: "90D", exact: true }),
      signal().getByRole("button", { name: /ChatGPT/ }),
      signal().getByRole("button", { name: /Gemini/ }),
    ];

    for (const control of controls) {
      window.scrollTo(0, 0);
      const before = router.state.location.href;

      await control.click();

      await expect.poll(() => router.state.location.href).not.toBe(before);
      // The router scrolls after it renders the new location, so give a jump time to happen.
      await new Promise((resolve) => setTimeout(resolve, 300));
      expect(Math.round(window.scrollY)).toBe(0);
    }

    expect(router.state.location.hash).toBe("");
  });

  test("an analytics error offers retry without showing a false empty state", async () => {
    const state = analytics();
    state.fail.add("summary");

    await renderRoute("/");

    const alert = page.getByRole("alert").filter({ hasText: "Visibility could not be loaded." });
    await expect.element(alert).toBeVisible();
    await expect
      .element(
        page.getByText("Add and confirm prompts to see how your brand appears across AI answers."),
      )
      .not.toBeInTheDocument();
    await expect
      .element(page.getByText("Add competitors to compare their visibility."))
      .not.toBeInTheDocument();

    state.fail.delete("summary");
    await alert.getByRole("button", { name: "Retry" }).click();

    await expect.element(alert).not.toBeInTheDocument();
    await expect.element(page.getByText("Rival 120", { exact: true })).toBeVisible();
  });

  test.each([
    1440, 960, 390, 320,
  ])("keeps chart and prompt geometry while filters load at %ipx", async (width) => {
    await page.viewport(width, 900);
    analytics();

    const { router } = await renderRoute("/", {
      theme: width === 960 || width === 320 ? "dark" : "light",
    });
    const hrefs: string[] = [];

    const panel = signal();
    const range = panel.getByRole("button", { name: "7D", exact: true });
    const prompts = await css("#prompts");

    await expect.element(panel).toHaveAttribute("aria-busy", "false");
    await expect.element(prompts).toHaveAttribute("aria-busy", "false");
    await expect.element(prompts.getByText(PROMPT).first()).toBeInTheDocument();
    await expect.poll(() => document.querySelector(".recharts-surface")).toBeTruthy();

    await layoutSettled(panel, prompts);
    const height = box(panel).height;
    const rangeTop = box(range).y;
    const promptHeight = box(prompts).height;
    const hold = gate();

    worker.use(
      http.get("/api/v1/visibility/summary", () => hold.opened),
      http.get("/api/v1/prompts", () => hold.opened),
    );

    try {
      await range.click();
      await new Promise((resolve) => setTimeout(resolve, 200));
      hrefs.push(router.state.location.href);

      await expect.element(panel).toHaveAttribute("aria-busy", "true");
      await expect.poll(() => opacity(panel.element().querySelector("figure"))).toBe(0.5);
      expect(opacity(range.element())).toBe(1);
      expect(document.querySelector(".recharts-surface")).toBeTruthy();
      expect(box(panel).height).toBe(height);
      expect(box(range).y).toBe(rangeTop);

      await expect.element(prompts).toHaveAttribute("aria-busy", "true");
      // Status counts keep their value while loading; an emptied count reflows the header.
      await expect
        .element(prompts.getByRole("button", { name: /^Tracking/ }))
        .toHaveTextContent("Tracking1");
      expect(box(prompts).height).toBe(promptHeight);
      await expect
        .poll(() => opacity(document.querySelector("#prompts li, #prompts tbody tr")))
        .toBe(0.5);

      // Rapid changes must keep the chart present too.
      await panel.getByRole("button", { name: /ChatGPT/ }).click();
      await new Promise((resolve) => setTimeout(resolve, 200));
      hrefs.push(router.state.location.href);
      expect(box(panel).height).toBe(height);
      await rivals()
        .getByRole("button", { name: /Rival 000/ })
        .click();
      await new Promise((resolve) => setTimeout(resolve, 200));
      hrefs.push(router.state.location.href);
      expect(document.querySelector(".recharts-surface")).toBeTruthy();
    } finally {
      hold.release();
    }

    await expect.element(panel).toHaveAttribute("aria-busy", "false");
    await expect.poll(() => opacity(panel.element().querySelector("figure"))).toBe(1);
    expect(hrefs.map((href) => new URL(href, location.origin).search)).toEqual([
      "?days=7",
      "?days=7&engines=chatgpt",
      "?days=7&engines=chatgpt&compareId=competitor-0",
    ]);
    await expect
      .element(panel.getByRole("button", { name: "7D", exact: true }))
      .toHaveAttribute("aria-pressed", "true");
  });

  test("confirmed prompts do not hide collection blockers or tracking recovery", async () => {
    const state = analytics();
    state.tracking = tracking({ startedAt: null, blockedReason: "collection_unavailable" });

    await renderRoute("/");

    await expect
      .element(
        page.getByText(
          "Live collection is unavailable. Ask your instance administrator to configure Cloro.",
        ),
      )
      .toBeVisible();
  });

  test("a tracking status error can be retried", async () => {
    const state = analytics();
    state.fail.add("tracking");

    await renderRoute("/");

    const error = page.getByRole("alert").filter({ hasText: "Unable to load tracking status" });
    await expect.element(error.first()).toBeVisible();

    state.fail.delete("tracking");
    await error.first().getByRole("button", { name: "Retry" }).click();

    await expect.element(error.first()).not.toBeInTheDocument();
  });

  test("checkbox selection supports keyboard and mixed page selection", async () => {
    await page.viewport(1440, 1000);
    const state = analytics();
    state.tracking = tracking({ startedAt: null });
    state.prompts = [
      prompt({ id: "prompt-0", brandId: "brand-120", text: "First question" }),
      prompt({ id: "prompt-1", brandId: "brand-120", text: "Second question" }),
    ];

    await renderRoute("/");

    const boxes = page.getByRole("checkbox");
    await expect.poll(() => boxes.all().length).toBe(3);

    await boxes.nth(1).click();
    await expect.element(boxes.nth(1)).toBeChecked();
    await expect.element(boxes.first()).toHaveAttribute("aria-checked", "mixed");

    await boxes.first().click();
    await expect.element(boxes.nth(2)).toBeChecked();

    await page.getByRole("button", { name: "Clear selection" }).click();
    await expect.element(boxes.first()).not.toBeChecked();
  });

  test("shows each prompt's own visibility, not the summary", async () => {
    const state = analytics();
    state.prompts = [
      prompt({
        id: "prompt",
        brandId: "brand-120",
        text: PROMPT,
        visibility: report({ metric: metric({ rate: 12 }), series: series([12, 12]) }),
      }),
    ];

    await renderRoute("/");

    await expect.element(promptLink()).toBeVisible();
    await expect
      .element(page.getByRole("region", { name: "Prompts" }).getByText("12.0%").first())
      .toBeVisible();
  });
});
