import { describe, expect, it } from "vitest";
import type { VisibilityEngine } from "@/api/generated";
import { ENGINES } from "./data";
import { createPreviewBrand, parseExplorationSearch } from "./exploration-fixtures";
import {
  buildEvidence,
  comparedBrands,
  dailySeries,
  type EvidenceRecord,
  filterPromptRows,
  gapChange,
  inScope,
  promptRows,
  promptRuns,
  rateFor,
  runCompetitors,
  summarize,
  type VisibilityScope,
} from "./prompt-visibility";

const brand = "tally";
const data = createPreviewBrand(brand);
const records = buildEvidence(brand);
const brands = comparedBrands(brand, data);
const scope = { days: 30, promptIds: [data.prompts[0].id] };

function sample(day: number, index: number, mentioned: boolean, rival: boolean): EvidenceRecord {
  const base = records[0];
  const appearance = records
    .flatMap((record) => record.result.appearances)
    .find((item) => item.competitorId === brands[1].id);
  if (!appearance) throw new Error("Missing rival fixture");

  return {
    answer: { ...base.answer, day },
    result: {
      ...base.result,
      id: `${day}:${index}`,
      promptId: scope.promptIds[0],
      outcome: mentioned ? "mentioned" : "miss",
      appearances: rival ? [appearance, appearance] : [],
    },
    stopped: false,
  };
}

describe("prompt visibility", () => {
  it("keeps the captured Deja prompt history to its two recorded runs", () => {
    const deja = createPreviewBrand("deja");
    const promptId = deja.prompts[0].id;
    const captured = promptRuns(buildEvidence("deja"), promptId);

    expect(captured.map((run) => run.id)).toEqual([
      "recorded:deja:2026-10-09",
      "recorded:deja:2026-10-08",
    ]);
    expect(captured[0].results.gemini?.result.outcome).toBe("error");
    expect(captured[0].results.chatgpt?.result.outcome).toBe("cited");
    expect(captured[1].results.gemini?.result.outcome).toBe("miss");
    expect(captured[1].results.perplexity?.result.rawAnswer).toContain("solid default");
    expect(captured[0].timestamp).toBe("2026-10-09T15:57:55.300Z");
    expect(captured[1].timestamp).toBe("2026-10-08T11:06:09.739Z");
  });

  it("lists each competitor once across every engine in a run", () => {
    const first = sample(89, 0, true, true);
    const second = sample(89, 1, false, true);

    expect(runCompetitors({ chatgpt: first, gemini: second })).toEqual([
      { id: brands[1].id, name: brands[1].name },
    ]);
    expect(runCompetitors({ chatgpt: sample(89, 2, false, false) })).toEqual([]);
  });

  it("groups a prompt by run across all engines and keeps older runs outside the metric period", () => {
    const runs = promptRuns(records, data.prompts[0].id);

    expect(runs).toHaveLength(180);
    expect(runs[0].id).toBe("tally:89");
    expect(runs.at(-1)?.id).toBe("tally:-90");
    expect(ENGINES.every((engine) => runs[0].results[engine.id]?.result.engine === engine.id)).toBe(
      true,
    );
    expect(inScope(records, { days: 7, promptIds: [data.prompts[0].id] })).toHaveLength(28);
  });

  it("uses answer counts for rates and keeps the comparison identity fixed across periods", () => {
    const answers = [
      ...Array.from({ length: 100 }, (_, i) => sample(89, i, i < 50, i < 68)),
      ...Array.from({ length: 100 }, (_, i) => sample(59, i, i < 42, i < 65)),
    ];
    const you = summarize(answers, brand, scope, brands[0]);
    const rival = summarize(answers, brand, scope, brands[1]);

    expect(you).toEqual({
      mentioned: 50,
      completed: 100,
      rate: 50,
      previousRate: 42,
      changePoints: 8,
    });
    expect(rival.rate).toBe(68);
    expect(gapChange(you, rival)).toBe(5);
    expect((rival.rate ?? 0) - (you.rate ?? 0)).toBe(18);
    expect(promptRows(answers, brand, data, 30)[0].topBrands.slice(0, 2)).toEqual([
      { brand: brands[1], rate: 68 },
      { brand: brands[0], rate: 50 },
    ]);
  });

  it("excludes failed and pending answers, while retaining real zeroes and cited outcomes", () => {
    const yes = sample(89, 0, true, true);
    const no = sample(89, 1, false, false);
    const failed = { ...no, result: { ...no.result, outcome: "error" as const } };
    const pending = { ...no, result: { ...no.result, outcome: null }, stopped: true };
    const cited = { ...yes, result: { ...yes.result, outcome: "cited" as const } };

    expect(rateFor([cited, no, failed, pending], brand)).toEqual({
      mentioned: 1,
      completed: 2,
      rate: 50,
    });
    expect(rateFor([failed, pending], brand).rate).toBeNull();
    expect(rateFor([no], brand).rate).toBe(0);
    expect(summarize([yes], brand, scope, brands[0]).changePoints).toBeNull();
  });

  it("uses matching evidence for table rows, scoped details, engine cells, and daily points", () => {
    const rows = promptRows(records, brand, data, 30, ["chatgpt"]);
    const selectedScope: VisibilityScope = { ...scope, engines: ["chatgpt"] };
    const summary = summarize(records, brand, selectedScope, brands[0]);
    const history = inScope(records, selectedScope);
    const series = dailySeries(records, brand, selectedScope, brands[0]);

    expect(rows[0].stats).toEqual(summary);
    expect(rows[0].engines).toEqual([{ id: "chatgpt", rate: summary.rate }]);
    expect(history).toHaveLength(30);
    expect(series.reduce((total, point) => total + point.completed, 0)).toBe(history.length);
    expect(series.reduce((total, point) => total + point.mentioned, 0)).toBe(summary.mentioned);
    expect(history.every((record) => record.result.promptId === rows[0].prompt.id)).toBe(true);
  });

  it("has complete previous 90-day evidence and leaves missing dates as null", () => {
    const longScope = { ...scope, days: 90 };
    expect(inScope(records, longScope)).toHaveLength(360);
    expect(inScope(records, longScope, true)).toHaveLength(360);
    const series = dailySeries([sample(89, 0, false, false)], brand, scope, brands[0]);

    expect(series).toHaveLength(30);
    expect(series.slice(0, -1).every((point) => point.rate === null)).toBe(true);
    expect(series.at(-1)?.rate).toBe(0);
  });

  it("preserves snapshots after management edits and keeps removed prompts discoverable", () => {
    const edited = createPreviewBrand(brand);
    const originalText = edited.prompts[0].text;
    const removed = edited.prompts.pop();
    edited.prompts[0].text = "New question";
    const rows = promptRows(records, brand, edited, 30);

    expect(rows[0].prompt.text).toBe("New question");
    expect(
      inScope(records, scope).every((record) => record.result.promptText === originalText),
    ).toBe(true);
    expect(filterPromptRows(rows, { status: "past" }).map((row) => row.prompt.id)).toEqual([
      removed?.id,
    ]);
    expect(filterPromptRows(rows, {}).every((row) => !row.past)).toBe(true);
  });

  it("does not fabricate observations and ranks up to four mentioned brands", () => {
    const edited = createPreviewBrand(brand);
    edited.prompts.push({ ...edited.prompts[0], id: "new", text: "New question" });
    edited.competitors.push({ ...edited.competitors[0], id: "new-brand", name: "New brand" });
    const rows = promptRows([sample(89, 0, true, true)], brand, edited, 30);

    expect(rows[0].topBrands.map(({ brand }) => brand.id)).toEqual([brands[0].id, brands[1].id]);
    expect(rows.find((row) => row.prompt.id === "new")?.stats.rate).toBeNull();
    const newBrand = comparedBrands(brand, edited).find((item) => item.id === "new-brand");
    if (!newBrand) throw new Error("Missing new competitor");
    expect(summarize(records, brand, scope, newBrand).rate).toBeNull();
    expect(promptRows([sample(89, 0, false, false)], brand, data, 30)[0].topBrands).toEqual([]);
  });

  it("filters before sorting and keeps unavailable rates last in either direction", () => {
    const edited = createPreviewBrand(brand);
    edited.prompts.push({ ...edited.prompts[0], id: "new", text: "ZZ new" });
    const rows = promptRows(records, brand, edited, 30);

    expect(filterPromptRows(rows, { sort: "rate", descending: true }).at(-1)?.prompt.id).toBe(
      "new",
    );
    expect(filterPromptRows(rows, { sort: "rate" }).at(-1)?.prompt.id).toBe("new");
    expect(filterPromptRows(rows, { q: "ZZ", sort: "rate" })).toHaveLength(1);
  });

  it("validates URL state and isolates each brand's evidence", () => {
    const parsed = parseExplorationSearch({
      days: "90",
      engines: ["invented"],
      page: -1,
      pageSize: "50",
      status: "past",
      sort: "rate",
      descending: "true",
      runId: [],
    });
    expect(parsed).toMatchObject({
      days: 90,
      pageSize: 50,
      status: "past",
      sort: "rate",
      descending: true,
    });
    expect(parsed.engines).toBeUndefined();
    expect(parsed.page).toBeUndefined();
    expect(parsed.runId).toBeUndefined();
    expect(inScope(buildEvidence("deja"), scope)).toEqual([]);
    expect(buildEvidence(brand, "empty")).toEqual([]);
    expect(buildEvidence(brand, "review")).toEqual([]);
    expect(
      buildEvidence(brand, "partial").some(
        (record) => record.stopped && record.result.outcome === null,
      ),
    ).toBe(true);
  });

  it("aggregates selected engines by answer count across periods, charts, and rows", () => {
    const withEngine = (record: EvidenceRecord, engine: VisibilityEngine) => ({
      ...record,
      result: { ...record.result, engine },
    });
    const answers = [
      ...Array.from({ length: 9 }, (_, i) => withEngine(sample(89, i, true, false), "chatgpt")),
      withEngine(sample(89, 9, false, true), "gemini"),
      withEngine(sample(89, 10, false, true), "perplexity"),
      withEngine(sample(59, 0, false, false), "chatgpt"),
      withEngine(sample(59, 1, true, true), "gemini"),
    ];
    const engines: VisibilityEngine[] = ["chatgpt", "gemini"];
    const filtered = { ...scope, engines };
    const stats = summarize(answers, brand, filtered, brands[0]);

    expect(stats).toMatchObject({
      mentioned: 9,
      completed: 10,
      rate: 90,
      previousRate: 50,
      changePoints: 40,
    });
    expect(summarize(answers, brand, filtered, brands[1]).rate).toBe(10);
    expect(dailySeries(answers, brand, filtered, brands[0]).at(-1)?.rate).toBe(90);
    expect(promptRows(answers, brand, data, 30, engines)[0].stats).toEqual(stats);
    expect(promptRows(answers, brand, data, 30, engines)[0].engines.map((item) => item.id)).toEqual(
      engines,
    );

    const all = inScope(answers, scope);
    expect(inScope(answers, { ...scope, engines: [] })).toEqual(all);
    expect(inScope(answers, { ...scope, engines: ENGINES.map((engine) => engine.id) })).toEqual(
      all,
    );
    expect(inScope(answers, { ...scope, engines: ["gemini"] })).toHaveLength(1);
    expect(
      summarize(answers, brand, { ...scope, engines: ["google_ai_overviews"] }, brands[0]).rate,
    ).toBeNull();
  });

  it("normalizes multiple engine URL values while retiring the scalar filter", () => {
    expect(
      parseExplorationSearch({ engines: ["gemini", "invented", "chatgpt", "gemini"] }).engines,
    ).toEqual(["chatgpt", "gemini"]);
    expect(
      parseExplorationSearch({ engines: ENGINES.map((engine) => engine.id) }).engines,
    ).toHaveLength(4);
    expect(parseExplorationSearch({ engines: [] }).engines).toBeUndefined();
    expect(parseExplorationSearch({ engines: "chatgpt" }).engines).toBeUndefined();
    expect(parseExplorationSearch({ engine: "chatgpt" }).engines).toBeUndefined();
  });
});
