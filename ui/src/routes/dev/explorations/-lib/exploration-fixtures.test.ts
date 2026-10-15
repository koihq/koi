import { describe, expect, it } from "vitest";
import {
  answerResult,
  collectionAnswers,
  confirmItem,
  createPreviewBrand,
  finishReview,
  pageOf,
  parseExplorationSearch,
  resultForState,
} from "./exploration-fixtures";
import { HOME_BRANDS, type HomeBrand, homeEvidence, mentionRate } from "./home-evidence";

describe("MVP exploration", () => {
  it("accepts only supported destinations, brands and explicit fixture states", () => {
    expect(parseExplorationSearch({ view: "prompts", brand: "tally", state: "review" })).toEqual({
      view: "prompts",
      brand: "tally",
      state: "review",
    });
    expect(parseExplorationSearch({ view: "brief", brand: "other", state: ["error"] })).toEqual({
      view: undefined,
      brand: undefined,
      state: undefined,
    });
  });

  it("returns the standard empty page and clamps after deleting the final page", () => {
    expect(pageOf([], 4, 20)).toEqual({
      items: [],
      currentPage: 1,
      totalPages: 0,
      totalItems: 0,
      hasPrev: false,
      hasNext: false,
    });
    const items = Array.from({ length: 21 }, (_, index) => index);
    expect(pageOf(items, 2, 20).items).toEqual([20]);
    expect(pageOf(items.slice(0, 20), 2, 20)).toMatchObject({
      currentPage: 1,
      totalPages: 1,
      hasNext: false,
      hasPrev: false,
    });
  });

  it("preserves baseline mention rates when adapting evidence for the shared answer panel", () => {
    for (const brand of Object.keys(HOME_BRANDS) as HomeBrand[]) {
      const answers = homeEvidence(brand).filter((answer) => answer.day >= 60);
      const results = answers.map((answer) => answerResult(answer, brand));
      const adaptedRate =
        (results.filter((result) => result.outcome === "mentioned").length / results.length) * 100;

      expect(adaptedRate).toBe(mentionRate(answers, HOME_BRANDS[brand].name));
      expect(results.map((result) => result.promptText)).toEqual(
        answers.map((answer) => answer.prompt),
      );
      expect(
        results.every((result) => result.collector === "stub" && result.sources.length === 0),
      ).toBe(true);
    }
  });

  it("confirms reviewed drafts but leaves observed competitors for individual identity review", () => {
    const data = createPreviewBrand("deja", "review");
    const observed = data.competitors.find((item) => item.source === "observed");
    if (!observed) throw new Error("Missing observed fixture");

    expect(confirmItem(data, "competitors", observed.id).competitors.at(-1)?.confirmed).toBe(false);
    const reviewed = finishReview(data, "competitors");
    expect(reviewed.reviewed).toEqual({ prompts: false, competitors: true });
    expect(reviewed.competitors.at(-1)?.confirmed).toBe(false);
    expect(data.reviewed.competitors).toBe(false);

    const identified = {
      ...data,
      competitors: data.competitors.map((item) =>
        item.id === observed.id ? { ...item, domain: "fishshell.com" } : item,
      ),
    };
    expect(confirmItem(identified, "competitors", observed.id).competitors.at(-1)?.confirmed).toBe(
      true,
    );
    expect(identified.competitors.at(-1)?.confirmed).toBe(false);
    expect(finishReview(reviewed, "prompts").reviewed).toEqual({
      prompts: true,
      competitors: true,
    });
  });

  it("keeps collections and answer snapshots independent of current tracking lists", () => {
    const data = createPreviewBrand("deja");
    const answers = collectionAnswers("deja", 89);
    const before = answers.map((answer) => answerResult(answer, "deja"));
    data.prompts[0].text = "A different question";
    data.prompts.splice(1);
    data.competitors.length = 0;

    expect(collectionAnswers("deja", 89).map((answer) => answerResult(answer, "deja"))).toEqual(
      before,
    );
    expect(createPreviewBrand("deja").prompts[0].text).not.toBe("A different question");
    expect(collectionAnswers("deja", 89, "empty")).toEqual([]);
  });

  it("distinguishes failed, pending and successful answers in partial collections", () => {
    const answers = collectionAnswers("deja", 89);
    const results = answers.map((answer, index) =>
      resultForState(answer, "deja", index, "partial"),
    );
    expect(results[0]).toMatchObject({
      outcome: null,
      fetchedAt: null,
      rawAnswer: "",
      appearances: [],
    });
    expect(results[1]).toMatchObject({ outcome: "error", rawAnswer: "", appearances: [] });
    expect(["mentioned", "miss"]).toContain(results[2].outcome);
    expect(answers[0].prompt).toBe(results[0].promptText);
  });
});
