import { describe, expect, it } from "vitest";

import { setupProgress } from "./setup-steps";

describe("setup progress", () => {
  it("gives both paths the same four steps on Koi Cloud", () => {
    const cloud = { canChoose: true, cloud: true };

    expect(setupProgress("choice", cloud)).toEqual({ current: 1, total: 4 });
    expect(setupProgress("website", cloud)).toEqual({ current: 2, total: 4 });
    expect(setupProgress("agent_connect", cloud)).toEqual({ current: 2, total: 4 });
    expect(setupProgress("confirm", cloud)).toEqual({ current: 3, total: 4 });
    expect(setupProgress("agent_prompt", cloud)).toEqual({ current: 3, total: 4 });
    expect(setupProgress("first_look", cloud)).toEqual({ current: 4, total: 4 });
  });

  it("counts only the steps a self-hosted instance has", () => {
    expect(setupProgress("confirm", { canChoose: true, cloud: false })).toEqual({
      current: 3,
      total: 3,
    });
    expect(setupProgress("website", { canChoose: false, cloud: false })).toEqual({
      current: 1,
      total: 2,
    });
    expect(setupProgress("confirm", { canChoose: false, cloud: false })).toEqual({
      current: 2,
      total: 2,
    });
  });
});
