import { describe, expect, it } from "vitest";

import { generationPollDelay } from "@/hooks/use-generation";

describe("generation polling", () => {
  it("spaces active polls, backs off after read errors, and checks for new runs while idle", () => {
    expect(generationPollDelay("queued", 0, 1200)).toBe(16_200);
    expect(generationPollDelay("running", 1, 1200)).toBe(31_200);
    expect(generationPollDelay("running", 4, 1200)).toBe(60_000);
    expect(generationPollDelay("succeeded", 0, 1200)).toBe(60_000);
    expect(generationPollDelay("failed", 2, 1200)).toBe(60_000);
  });
});
