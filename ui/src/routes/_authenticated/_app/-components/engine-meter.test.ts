import { describe, expect, it } from "vitest";

import { filledTicks } from "@/routes/_authenticated/_app/-components/engine-meter";

describe("filledTicks", () => {
  it("fills one tick per 20 points, rounding up so presence always shows", () => {
    expect(filledTicks(null)).toBe(0);
    expect(filledTicks(0)).toBe(0);
    expect(filledTicks(1)).toBe(1);
    expect(filledTicks(13.3)).toBe(1);
    expect(filledTicks(60)).toBe(3);
    expect(filledTicks(100)).toBe(5);
  });
});
