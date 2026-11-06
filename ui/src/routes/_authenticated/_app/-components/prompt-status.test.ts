import { describe, expect, it } from "vitest";

import { canApply, type PromptBulkKind } from "./prompt-status";

const kinds: PromptBulkKind[] = ["confirm", "stop", "resume", "delete"];

function applicable(confirmed: boolean, inactive: boolean, hasResults: boolean) {
  return kinds.filter((kind) => canApply(kind, { confirmed, inactive, hasResults }));
}

describe("canApply", () => {
  it("offers one way out of the list for each prompt", () => {
    expect(applicable(false, false, false)).toEqual(["confirm", "delete"]);
    expect(applicable(true, false, true)).toEqual(["stop"]);
    expect(applicable(true, false, false)).toEqual(["delete"]);
    expect(applicable(true, true, true)).toEqual(["resume"]);
    expect(applicable(true, true, false)).toEqual(["resume", "delete"]);
  });
});
