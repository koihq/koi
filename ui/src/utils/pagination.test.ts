import { describe, expect, it } from "vitest";

import { lastPage } from "@/utils/pagination";

describe("lastPage", () => {
  it("keeps empty lists on page one and calculates the last populated page", () => {
    expect(lastPage(0, 10)).toBe(1);
    expect(lastPage(10, 10)).toBe(1);
    expect(lastPage(11, 10)).toBe(2);
  });
});
