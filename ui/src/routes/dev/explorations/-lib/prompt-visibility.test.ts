import { describe, expect, it } from "vitest";
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
  rateFor,
  summarize,
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
    const rows = promptRows(records, brand, data, 30, "chatgpt");
    const selectedScope = { ...scope, engine: "chatgpt" };
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

  it("does not fabricate observations for new prompts or competitors, and exposes tied leaders", () => {
    const edited = createPreviewBrand(brand);
    edited.prompts.push({ ...edited.prompts[0], id: "new", text: "New question" });
    edited.competitors.push({ ...edited.competitors[0], id: "new-brand", name: "New brand" });
    const rows = promptRows([sample(89, 0, true, true)], brand, edited, 30);

    expect(rows[0].leaders.map((item) => item.id)).toEqual([brands[0].id, brands[1].id]);
    expect(rows.find((row) => row.prompt.id === "new")?.stats.rate).toBeNull();
    const newBrand = comparedBrands(brand, edited).find((item) => item.id === "new-brand");
    if (!newBrand) throw new Error("Missing new competitor");
    expect(summarize(records, brand, scope, newBrand).rate).toBeNull();
    expect(promptRows([sample(89, 0, false, false)], brand, data, 30)[0].leaders).toEqual([]);
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
      engine: "invented",
      page: -1,
      pageSize: "50",
      status: "past",
      sort: "rate",
      descending: "true",
      answerId: [],
    });
    expect(parsed).toMatchObject({
      days: 90,
      pageSize: 50,
      status: "past",
      sort: "rate",
      descending: true,
    });
    expect(parsed.engine).toBeUndefined();
    expect(parsed.page).toBeUndefined();
    expect(parsed.answerId).toBeUndefined();
    expect(inScope(buildEvidence("deja"), scope)).toEqual([]);
    expect(buildEvidence(brand, "empty")).toEqual([]);
    expect(buildEvidence(brand, "review")).toEqual([]);
    expect(
      buildEvidence(brand, "partial").some(
        (record) => record.stopped && record.result.outcome === null,
      ),
    ).toBe(true);
  });
});
