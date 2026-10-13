import { describe, expect, it } from "vitest";
import { FIXTURES, type Observation } from "../-lib/data";
import { visibilitySeries } from "./-visibility-data";
import { addPanel, defaultWorkspaces, validWorkspaces } from "./-workspace";

const sample: Observation[] = [
  {
    ...FIXTURES[0],
    day: 179,
    promptId: "first",
    engine: "chatgpt",
    outcome: "mentioned",
    brands: ["tally"],
  },
  {
    ...FIXTURES[0],
    day: 179,
    promptId: "first",
    engine: "perplexity",
    outcome: "miss",
    brands: ["typeform"],
  },
  { ...FIXTURES[0], day: 179, promptId: "first", engine: "gemini", outcome: "error", brands: [] },
  {
    ...FIXTURES[0],
    day: 179,
    promptId: "second",
    engine: "chatgpt",
    outcome: "miss",
    brands: ["typeform"],
  },
];

describe("visibility comparisons", () => {
  it("uses the same successful-answer denominator for all brands", () => {
    const series = visibilitySeries(
      sample,
      { days: 7, engine: "all" },
      ["tally", "typeform"],
      null,
    );

    expect(series[0].points.at(-1)?.value).toBeCloseTo(100 / 3);
    expect(series[1].points.at(-1)?.value).toBeCloseTo(200 / 3);
    expect(series[0].points[0].value).toBeNull();
  });

  it("applies prompt and engine scope to every brand", () => {
    const series = visibilitySeries(
      sample,
      { days: 7, engine: "chatgpt" },
      ["tally", "typeform"],
      "first",
    );

    expect(series.map((brand) => brand.points.at(-1)?.value)).toEqual([100, 0]);
  });

  it("keeps uncollected prompts and failed-only days as gaps", () => {
    const missing = visibilitySeries(sample, { days: 7, engine: "all" }, ["tally"], "new");
    const failed = visibilitySeries(sample, { days: 7, engine: "gemini" }, ["tally"], "first");

    expect(missing[0].points.every((point) => point.value === null)).toBe(true);
    expect(failed[0].points.at(-1)?.value).toBeNull();
  });
});

describe("consolidated workspaces", () => {
  it("keeps the revised presets and added panels persistable", () => {
    const workspaces = defaultWorkspaces();
    const expanded = addPanel(workspaces[0], "engines");

    expect(validWorkspaces(workspaces)).toBe(true);
    expect(validWorkspaces([expanded, ...workspaces.slice(1)])).toBe(true);
    expect(workspaces.map((workspace) => workspace.template)).toEqual(["research", "compare"]);
  });

  it("rejects layouts containing retired panels", () => {
    const workspaces = defaultWorkspaces();
    const retired = { ...workspaces[0], panels: [...workspaces[0].panels, "heatmap"] };

    expect(validWorkspaces([retired])).toBe(false);
  });
});
