import { describe, expect, it } from "vitest";
import type { VisibilityRunStatus } from "@/api/generated";
import { answerMessageKey, getAnswerState, isCollecting } from "./answer-state";

describe("saved answer states", () => {
  const run = { state: "succeeded", engines: ["chatgpt"] } satisfies Pick<
    VisibilityRunStatus,
    "state" | "engines"
  >;

  it("distinguishes an omitted engine from an unfinished and an uncollected answer", () => {
    expect(getAnswerState(run, "gemini")).toBe("not-requested");
    expect(getAnswerState(run, "chatgpt")).toBe("not-collected");
    expect(getAnswerState({ ...run, state: "failed" }, "chatgpt")).toBe("not-collected");
    for (const state of ["queued", "running"] as const) {
      expect(getAnswerState({ ...run, state }, "chatgpt")).toBe("pending");
      expect(isCollecting({ state })).toBe(true);
    }
    expect(isCollecting(run)).toBe(false);
    expect(isCollecting()).toBe(false);
  });

  it("preserves saved outcomes and prioritizes collection errors", () => {
    for (const outcome of ["cited", "mentioned", "miss", "error"] as const) {
      expect(getAnswerState(run, "chatgpt", { outcome, errorCode: null })).toBe(outcome);
    }
    expect(getAnswerState(run, "chatgpt", { outcome: "miss", errorCode: "upstream_error" })).toBe(
      "error",
    );
    expect(getAnswerState(run, "gemini", { outcome: "cited", errorCode: null })).toBe("cited");
  });

  it("never calls an empty terminal answer pending", () => {
    for (const state of ["cited", "mentioned", "miss"] as const) {
      expect(answerMessageKey(state)).toBe("empty");
    }
    expect(answerMessageKey("error")).toBe("error");
    expect(answerMessageKey("not-collected")).toBe("not-collected");
  });
});
