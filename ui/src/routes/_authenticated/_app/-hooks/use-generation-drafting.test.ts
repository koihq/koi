import { describe, expect, it } from "vitest";

import { generationPollDelay } from "@/routes/_authenticated/_app/-hooks/use-generation-drafting";

describe("generation polling", () => {
  it("spaces active polls, backs off after read errors, and stops at a terminal state", () => {
    expect(generationPollDelay("queued", 0, 1200)).toBe(16_200);
    expect(generationPollDelay("running", 1, 1200)).toBe(31_200);
    expect(generationPollDelay("running", 4, 1200)).toBe(60_000);
    expect(generationPollDelay("succeeded", 0, 1200)).toBe(false);
    expect(generationPollDelay("failed", 2, 1200)).toBe(false);
  });
});
