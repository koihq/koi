import { describe, expect, test } from "vitest";
import { commands, type Locator, page, userEvent } from "vitest/browser";
import { cleanup } from "vitest-browser-react";
import { box, css, deepText, pointerAway } from "@/test/dom";
import {
  competitorVisibility,
  metric,
  prompt,
  report,
  run,
  series,
  tracking,
} from "@/test/fixtures";
import { renderRoute } from "@/test/render-route";
import { serveWorkspace, workspace } from "@/test/worker";

const PATHS = ["/?compareId=rival", "/prompts/prompt-0?compareId=rival"];

/** A brand at `rates` per day, compared with a rival at the complementary rate. */
function compared(rates: (number | null)[]) {
  const visibility = report({
    metric: metric({ rate: rates.at(-1) ?? 50 }),
    series: series(rates),
    competitors: [
      competitorVisibility({
        metric: metric({ rate: 100 - (rates.at(-1) ?? 50) }),
        series: series(rates.map((rate) => (rate === null ? null : 100 - rate))),
      }),
    ],
  });

  serveWorkspace(
    workspace({
      tracking: tracking({ startedAt: null }),
      summary: visibility,
      promptVisibility: visibility,
      prompts: [prompt({ visibility })],
      runs: [run({ state: "succeeded" })],
    }),
  );
}

/** Renders each compared view in turn, starting fresh for every path. */
async function eachView(check: (chart: Locator) => Promise<void>) {
  for (const path of PATHS) {
    await renderRoute(path);
    await check(await css('[data-slot="chart"]'));
    await cleanup();
  }
}

function hoverAt(chart: Locator, fraction: number) {
  const { width, height } = box(chart);

  return userEvent.hover(chart, { position: { x: width * fraction, y: height / 2 } });
}

const markers = (chart: Locator) =>
  chart.element().querySelectorAll(".signal-active-marker").length;

const curves = (chart: Locator, stroke: string) => [
  ...chart
    .element()
    .querySelectorAll<SVGPathElement>(`.recharts-line-curve[stroke-width="${stroke}"]`),
];

describe("visibility chart", () => {
  test.each([
    [[0, 0, 0]],
    [[100, 100, 100]],
    [[10, 50, 80]],
    [[10, null, 80]],
  ])("keeps style and hover consistent for %j", async (rates) => {
    await page.viewport(1440, 1000);
    compared(rates);

    const appearances: { stroke: string; linecap: string }[] = [];

    await eachView(async (chart) => {
      await expect.poll(() => curves(chart, "4")[0]?.getAttribute("d") ?? "").toMatch(/M/);
      const own = curves(chart, "4")[0];

      // A clip on the exact Y domain removes half the stroke at 0 and 100.
      expect(own.getAttribute("clip-path")).toBeNull();
      expect(getComputedStyle(own).strokeWidth).toBe("4px");
      appearances.push({
        stroke: getComputedStyle(own).stroke,
        linecap: getComputedStyle(own).strokeLinecap,
      });

      await hoverAt(chart, 0.4);
      await expect.poll(() => markers(chart)).toBe(rates[1] === null ? 0 : 2);
      await expect
        .poll(() => curves(chart, "4")[0].getAttribute("stroke-opacity"))
        .toBe(rates[1] === null ? "1" : "0.2");

      if (rates[1] !== null) {
        for (const stroke of ["4", "1.75"]) {
          const paths = curves(chart, stroke);

          expect(paths).toHaveLength(2);
          expect(paths[1].getAttribute("d")).toBe(paths[0].getAttribute("d"));
          expect(getComputedStyle(paths[1]).clipPath).toMatch(/url\(.+\)/);
        }

        const marker = chart.element().querySelector(".signal-active-marker") as Element;
        expect(getComputedStyle(marker).transitionDuration).toBe("0.14s");

        await commands.emulateMedia({ reducedMotion: "reduce" });
        await expect.poll(() => getComputedStyle(marker).transitionDuration).toBe("0s");
        await commands.emulateMedia({ reducedMotion: "no-preference" });
      }

      await hoverAt(chart, 0.7);
      await expect.poll(() => markers(chart)).toBe(2);

      await pointerAway();
      await expect.poll(() => markers(chart)).toBe(0);
      expect(curves(chart, "4")[0].getAttribute("stroke-opacity")).toBe("1");
      expect(chart.element().querySelectorAll("foreignObject")).toHaveLength(2);
    });

    expect(appearances[0]).toEqual(appearances[1]);
  });

  test.each([
    [[null, null, 75]],
    [[75, null, null]],
    [[75, null, 50]],
  ])("preserves data while hovering empty days of %j", async (rates) => {
    compared(rates);

    const recorded = rates.filter((rate) => rate !== null).length;
    const latest = `${(rates.at(-1) ?? 50).toFixed(1)}%`;

    await eachView(async (chart) => {
      const panel = page.elementLocator(chart.element().closest("section") as Element);
      const headline = () =>
        deepText(panel.element().querySelector("strong") as Element).replace(/\s/g, "");
      const dots = () => chart.element().querySelectorAll(".recharts-reference-dot circle");

      await expect.poll(headline).toBe(latest);
      await expect.poll(() => dots().length).toBe(recorded * 2);

      await hoverAt(chart, 0.4);
      expect(headline()).toBe(latest);
      expect(markers(chart)).toBe(0);
      expect(dots()).toHaveLength(recorded * 2);

      // Inspect an actual reading, then return to the gap.
      const point = dots()[0].getBoundingClientRect();
      const frame = box(chart);
      await userEvent.hover(chart, {
        position: {
          x: point.x + point.width / 2 - frame.x,
          y: point.y + point.height / 2 - frame.y,
        },
      });
      await expect.poll(() => markers(chart)).toBe(2);

      await hoverAt(chart, 0.4);
      await expect.poll(headline).toBe(latest);
      expect(dots()).toHaveLength(recorded * 2);

      // Keyboard inspection also visits only recorded days.
      const slider = panel.element().querySelector("input[type=range]") as HTMLInputElement;
      const readout = () => slider.getAttribute("aria-valuetext") ?? "";
      slider.focus();
      await expect.poll(readout).toMatch(/\d+\.\d%/);
      expect(readout()).not.toMatch(/No data/);
      await userEvent.keyboard("{ArrowLeft}");
      await expect.poll(readout).toMatch(/\d+\.\d%/);
      expect(readout()).not.toMatch(/No data/);
      await userEvent.keyboard("{Escape}");
      await expect.poll(headline).toBe(latest);
    });
  });
});
