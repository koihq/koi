import { describe, expect, it } from "vitest";

import { markAnswer } from "@/lib/answer-marks";

describe("markAnswer", () => {
  it("marks the brand and rivals on word edges", () => {
    expect(markAnswer("Koi and Peec AI beat Koifish.", "Koi", ["Peec AI"])).toEqual([
      { text: "Koi", mark: "brand" },
      { text: " and " },
      { text: "Peec AI", mark: "rival" },
      { text: " beat Koifish." },
    ]);
  });

  it("matches case-insensitively and prefers the longest name", () => {
    expect(markAnswer("try peec ai pro", "Koi", ["Peec", "Peec AI"])).toEqual([
      { text: "try " },
      { text: "peec ai", mark: "rival" },
      { text: " pro" },
    ]);
  });

  it("escapes names with regex characters", () => {
    expect(markAnswer("Use A.I. (beta) now", "A.I. (beta)", [])).toEqual([
      { text: "Use " },
      { text: "A.I. (beta)", mark: "brand" },
      { text: " now" },
    ]);
  });

  it("returns the text whole when nothing matches", () => {
    expect(markAnswer("Nothing here", "Koi", ["Peec AI"])).toEqual([{ text: "Nothing here" }]);
  });
});
