import { describe, expect, it } from "vitest";

import type { VisibilityResult } from "@/api/generated";
import {
  answerPassage,
  displaySite,
  firstLookPlan,
  firstLookRows,
  firstLookScore,
  firstLookVerdict,
  firstLookWait,
  keepVerdict,
  recognitionGap,
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
    mentionVerdict: null,
    ...overrides,
  } as VisibilityResult;
}

const verdictOf = (results: VisibilityResult[]) => firstLookVerdict(results, "Acme");

const rival = { competitorId: null, name: "Stripe", confirmed: false, evidence: [] };

describe("first look verdict", () => {
  it("waits for a real answer", () => {
    expect(verdictOf([])).toBeNull();
    expect(verdictOf([answer({ outcome: null })])).toBeNull();
    expect(verdictOf([answer({ outcome: "error" })])).toBeNull();
  });

  it("leads with the competitor recommended instead of the brand", () => {
    const verdict = verdictOf([
      answer({ outcome: "cited", engine: "chatgpt" }),
      answer({ outcome: "miss", engine: "google_ai_mode", appearances: [rival] }),
    ]);

    expect(verdict).toMatchObject({
      kind: "rival",
      engine: "google_ai_mode",
      competitor: "Stripe",
    });
  });

  it("does not claim a win over a rival when the brand also appeared", () => {
    const verdict = verdictOf([answer({ outcome: "mentioned", appearances: [rival] })]);

    expect(verdict?.kind).toBe("mentioned");
  });

  it("prefers a citation to a mention, and a mention to an absence", () => {
    expect(verdictOf([answer({ outcome: "miss" }), answer({ outcome: "mentioned" })])?.kind).toBe(
      "mentioned",
    );
    expect(verdictOf([answer({ outcome: "mentioned" }), answer({ outcome: "cited" })])?.kind).toBe(
      "cited",
    );
    expect(verdictOf([answer({ outcome: "miss" })])?.kind).toBe("absent");
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

describe("recognition gap", () => {
  const named = { promptText: "What is Acme?" };

  it("leads with the engine not knowing the brand, ahead of a rival", () => {
    const verdict = verdictOf([
      answer({ id: "lost", outcome: "miss", appearances: [rival] }),
      answer({ ...named, id: "gap", engine: "gemini", mentionVerdict: "no_info" }),
    ]);

    expect(verdict).toMatchObject({ kind: "unknown", engine: "gemini" });
    expect(verdict?.result.id).toBe("gap");
    expect(verdict?.competitor).toBeUndefined();
  });

  it("names who the engine suggests when it doesn't know the brand", () => {
    const verdict = verdictOf([
      answer({ ...named, mentionVerdict: "no_info", appearances: [rival] }),
    ]);

    expect(verdict).toMatchObject({ kind: "unknown", competitor: "Stripe" });
  });

  it("prefers not knowing over confusing, and never names a namesake as a rival", () => {
    const confused = answer({
      ...named,
      id: "confused",
      mentionVerdict: "other_company",
      appearances: [{ ...rival, name: "Acme Security" }],
    });

    expect(verdictOf([confused])).toMatchObject({ kind: "confused", competitor: undefined });
    expect(verdictOf([confused, answer({ ...named, mentionVerdict: "no_info" })])?.kind).toBe(
      "unknown",
    );
  });

  it("ignores namesakes when the question didn't name the brand", () => {
    const insurer = answer({
      promptText: "Cheapest car insurance?",
      mentionVerdict: "other_company",
    });

    expect(recognitionGap(insurer, "Acme")).toBeNull();
    expect(verdictOf([insurer])?.kind).toBe("absent");
  });

  it("does not treat another meaning or a real mention as a gap", () => {
    expect(recognitionGap({ ...named, mentionVerdict: "other_meaning" }, "Acme")).toBeNull();
    expect(recognitionGap({ ...named, mentionVerdict: "mention" }, "Acme")).toBeNull();
    expect(recognitionGap({ ...named, mentionVerdict: null }, "Acme")).toBeNull();
  });

  it("outranks every other headline once it arrives", () => {
    const lost = verdictOf([answer({ outcome: "miss", appearances: [rival] })]);
    const confused = verdictOf([answer({ ...named, mentionVerdict: "other_company" })]);
    const unknown = verdictOf([answer({ ...named, mentionVerdict: "no_info" })]);

    expect(keepVerdict(lost, confused)).toBe(confused);
    expect(keepVerdict(confused, unknown)).toBe(unknown);
    expect(keepVerdict(unknown, lost)).toBe(unknown);
  });

  it("counts these answers as not seeing the brand", () => {
    expect(firstLookScore([answer({ ...named, mentionVerdict: "no_info" })])).toEqual({
      seen: 0,
      total: 1,
    });
  });
});

describe("first look headline", () => {
  const mentioned = verdictOf([answer({ id: "a", outcome: "mentioned" })]);
  const cited = verdictOf([answer({ id: "b", outcome: "cited" })]);
  const lost = verdictOf([answer({ id: "c", outcome: "miss", appearances: [rival] })]);
  const otherLoss = verdictOf([
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

describe("firstLookWait", () => {
  const summary = { product: "Koi", category: "AEO tool", audience: "", functions: [] };

  it("follows what the draft has saved", () => {
    expect(firstLookWait({ state: "queued" })).toEqual({
      read: "active",
      draft: "pending",
      ask: "pending",
    });
    expect(firstLookWait({ state: "running" })).toEqual({
      read: "active",
      draft: "pending",
      ask: "pending",
    });
    expect(firstLookWait({ state: "running", summary })).toEqual({
      read: "done",
      draft: "active",
      ask: "pending",
    });
    expect(firstLookWait({ state: "succeeded", summary })).toEqual({
      read: "done",
      draft: "done",
      ask: "active",
    });
  });

  it("does not wait on a summary that never came once the draft is done", () => {
    expect(firstLookWait({ state: "succeeded" }).read).toBe("done");
  });

  it("only asks the engines without a live draft", () => {
    const askOnly = { read: null, draft: null, ask: "active" };

    expect(firstLookWait(undefined)).toEqual(askOnly);
    expect(firstLookWait({ state: "failed" })).toEqual(askOnly);
  });
});

describe("displaySite", () => {
  it("drops the protocol and trailing slash", () => {
    expect(displaySite("https://github.com/koihq/koi/")).toBe("github.com/koihq/koi");
    expect(displaySite("koi.dev")).toBe("koi.dev");
  });
});
