import { describe, expect, it } from "vitest";

import type { VisibilityResult } from "@/api/generated";
import {
  answerPassage,
  firstLookPlan,
  firstLookRows,
  firstLookScore,
  firstLookVerdict,
  keepVerdict,
} from "./first-look";

function answer(overrides: Partial<VisibilityResult>): VisibilityResult {
  return {
    id: "result",
    runId: "run",
    promptId: "prompt",
    promptText: "Best payment tools?",
    promptPosition: 0,
    engine: "chatgpt",
    collector: "cloro",
    fetchedAt: new Date(0),
    outcome: "miss",
    rawAnswer: "An answer",
    sources: [],
    errorCode: null,
    appearances: [],
    ...overrides,
  } as VisibilityResult;
}

const rival = { competitorId: null, name: "Stripe", confirmed: false, evidence: [] };

describe("first look verdict", () => {
  it("waits for a real answer", () => {
    expect(firstLookVerdict([])).toBeNull();
    expect(firstLookVerdict([answer({ outcome: null })])).toBeNull();
    expect(firstLookVerdict([answer({ outcome: "error" })])).toBeNull();
  });

  it("leads with the competitor recommended instead of the brand", () => {
    const verdict = firstLookVerdict([
      answer({ outcome: "cited", engine: "chatgpt" }),
      answer({ outcome: "miss", engine: "perplexity", appearances: [rival] }),
    ]);

    expect(verdict).toMatchObject({ kind: "rival", engine: "perplexity", competitor: "Stripe" });
  });

  it("does not claim a win over a rival when the brand also appeared", () => {
    const verdict = firstLookVerdict([answer({ outcome: "mentioned", appearances: [rival] })]);

    expect(verdict?.kind).toBe("mentioned");
  });

  it("prefers a citation to a mention, and a mention to an absence", () => {
    expect(
      firstLookVerdict([answer({ outcome: "miss" }), answer({ outcome: "mentioned" })])?.kind,
    ).toBe("mentioned");
    expect(
      firstLookVerdict([answer({ outcome: "mentioned" }), answer({ outcome: "cited" })])?.kind,
    ).toBe("cited");
    expect(firstLookVerdict([answer({ outcome: "miss" })])?.kind).toBe("absent");
  });
});

describe("first look plan", () => {
  it("counts only answers that left the brand out", () => {
    const steps = firstLookPlan(
      [
        answer({
          outcome: "miss",
          appearances: [rival],
          sources: ["https://www.g2.com/a", "https://acme.com/x"],
        }),
        answer({ outcome: "miss", appearances: [rival], sources: ["https://g2.com/b"] }),
        answer({ outcome: "cited", appearances: [{ ...rival, name: "Adyen" }] }),
        answer({ outcome: "error", appearances: [{ ...rival, name: "Square" }] }),
      ],
      "acme.com",
    );

    expect(steps).toEqual([
      { kind: "rival", competitor: "Stripe", count: 2, total: 2 },
      { kind: "source", source: "g2.com", count: 2 },
      { kind: "prompt", prompt: "Best payment tools?", count: 2 },
    ]);
  });

  it("invents nothing when the brand appears everywhere", () => {
    expect(firstLookPlan([answer({ outcome: "cited" })], "acme.com")).toEqual([]);
  });

  it("omits a step it has no evidence for", () => {
    const steps = firstLookPlan([answer({ outcome: "miss", sources: ["not a url"] })], "acme.com");

    expect(steps.map((step) => step.kind)).toEqual(["prompt"]);
  });
});

describe("first look rows", () => {
  it("groups answers by question in the order they are asked", () => {
    const rows = firstLookRows([
      answer({ promptId: "a", promptText: "First?", engine: "chatgpt" }),
      answer({ promptId: "a", promptText: "First?", engine: "gemini", outcome: null }),
      answer({ promptId: "b", promptText: "Second?" }),
    ]);

    expect(rows.map((row) => [row.text, row.answers.length])).toEqual([
      ["First?", 2],
      ["Second?", 1],
    ]);
  });
});

describe("first look headline", () => {
  const mentioned = firstLookVerdict([answer({ id: "a", outcome: "mentioned" })]);
  const cited = firstLookVerdict([answer({ id: "b", outcome: "cited" })]);
  const lost = firstLookVerdict([answer({ id: "c", outcome: "miss", appearances: [rival] })]);
  const otherLoss = firstLookVerdict([
    answer({ id: "d", outcome: "miss", appearances: [{ ...rival, name: "Adyen" }] }),
  ]);

  it("changes only for a stronger finding", () => {
    expect(keepVerdict(null, mentioned)).toBe(mentioned);
    expect(keepVerdict(mentioned, cited)).toBe(cited);
    expect(keepVerdict(cited, lost)).toBe(lost);
    expect(keepVerdict(lost, cited)).toBe(lost);
    expect(keepVerdict(lost, otherLoss)).toBe(lost);
    expect(keepVerdict(lost, null)).toBe(lost);
  });
});

describe("first look score", () => {
  it("counts real answers only", () => {
    expect(
      firstLookScore([
        answer({ outcome: "cited" }),
        answer({ outcome: "mentioned" }),
        answer({ outcome: "miss" }),
        answer({ outcome: "error" }),
        answer({ outcome: null }),
      ]),
    ).toEqual({ seen: 2, total: 3 });
  });
});

describe("answer passage", () => {
  const text = "Opening line.\n\nGetCito is one option.\nElmo tracks answers.\nAcme does too.";

  it("leads with the paragraph naming a rival", () => {
    expect(answerPassage(text, "Acme", ["Elmo"])).toEqual({
      passage: "Elmo tracks answers.",
      trimmed: true,
    });
  });

  it("falls back to the brand, then to the opening", () => {
    expect(answerPassage(text, "Acme", ["Missing"]).passage).toBe("Acme does too.");
    expect(answerPassage(text, "Nope", []).passage).toBe("Opening line.");
  });

  it("does not match inside a longer word or offer more of a single paragraph", () => {
    expect(answerPassage("Elmonet is here.", "Nope", ["Elmo"])).toEqual({
      passage: "Elmonet is here.",
      trimmed: false,
    });
  });
});
