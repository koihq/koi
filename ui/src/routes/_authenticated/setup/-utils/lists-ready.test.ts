import { describe, expect, it } from "vitest";

import { listsReady } from "@/routes/_authenticated/setup/-utils/lists-ready";

describe("listsReady", () => {
  it("requires two non-empty lists with no unconfirmed items", () => {
    const empty = { totalItems: 0 };
    const populated = { totalItems: 2 };
    const unconfirmed = { totalItems: 1 };
    expect(listsReady(empty, populated, empty, empty)).toBe(false);
    expect(listsReady(populated, empty, empty, empty)).toBe(false);
    expect(listsReady(populated, populated, unconfirmed, empty)).toBe(false);
    expect(listsReady(populated, populated, empty, unconfirmed)).toBe(false);
    expect(listsReady(populated, populated, empty, empty)).toBe(true);
  });
});
