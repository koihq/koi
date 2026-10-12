import { describe, expect, it } from "vitest";

import type { VisibilityResult } from "@/api/runs.api";
import {
  groupVisibilityResults,
  safeSourceURL,
  visibilityPollDelay,
} from "@/routes/_authenticated/_app/mentions/-lib/visibility";

describe("visibility matrix", () => {
  it("groups by historical identity, orders prompts and keeps pending/error cells", () => {
    const result = {
      runId: "run",
      promptId: "first",
      promptText: "Original question?",
      promptPosition: 0,
      engine: "chatgpt",
      collector: "stub",
      outcome: null,
      fetchedAt: null,
      rawAnswer: "",
      appearances: [],
      sources: [],
      errorCode: null,
    } satisfies Omit<VisibilityResult, "id">;

    const grouped = groupVisibilityResults([
      { ...result, id: "third", promptId: "second", promptPosition: 1 },
      { ...result, id: "second", engine: "perplexity", outcome: "error", errorCode: "stub_error" },
      { ...result, id: "first" },
    ]);

    expect(grouped.map((row) => row.promptID)).toEqual(["first", "second"]);
    expect(grouped[0].results.get("chatgpt")?.outcome).toBeNull();
    expect(grouped[0].results.get("perplexity")?.outcome).toBe("error");
  });
  it("only makes HTTP sources clickable", () => {
    expect(safeSourceURL("https://koi.dev/docs")).toBe("https://koi.dev/docs");
    expect(safeSourceURL("http://example.com")).toBe("http://example.com/");

    for (const value of ["javascript:alert(1)", "data:text/html,bad", "/relative", "not a URL"]) {
      expect(safeSourceURL(value)).toBeUndefined();
    }
  });
});

describe("visibility polling", () => {
  it("polls active runs, backs off after errors, and stops on completion", () => {
    expect(visibilityPollDelay("queued")).toBe(1000);
    expect(visibilityPollDelay("running", 2)).toBe(4000);
    expect(visibilityPollDelay("running", 10)).toBe(30_000);
    expect(visibilityPollDelay("succeeded")).toBe(false);
    expect(visibilityPollDelay("failed")).toBe(false);
    expect(visibilityPollDelay(undefined)).toBe(false);
  });
});
