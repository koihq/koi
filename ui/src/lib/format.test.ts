import { describe, expect, it } from "vitest";

import { formatDateTime, formatDay, formatDayRange, formatPoints, formatRate } from "@/lib/format";

const now = new Date(2026, 10, 2, 12);

describe("formatDay", () => {
  it("omits the current year", () => {
    expect(formatDay(new Date(2026, 9, 4), "en-US", now)).toBe("Oct 4");
  });

  it("adds a different year", () => {
    expect(formatDay(new Date(2025, 11, 30), "en-US", now)).toBe("Dec 30, 2025");
  });
});

describe("formatDayRange", () => {
  it("joins two days with an en dash", () => {
    expect(formatDayRange(new Date(2026, 9, 4), new Date(2026, 10, 2), "en-US", now)).toBe(
      "Oct 4 – Nov 2",
    );
  });
});

describe("formatDateTime", () => {
  it("drops seconds", () => {
    expect(formatDateTime(new Date(2026, 10, 2, 8, 22, 54), "en-US", now)).toBe("Nov 2, 8:22 AM");
  });
});

describe("formatPoints", () => {
  it("signs differences with a real minus", () => {
    expect(formatPoints(-16, "en-US")).toBe("−16.0 pp");
    expect(formatPoints(4.25, "en-US")).toBe("+4.3 pp");
    expect(formatPoints(0, "en-US")).toBe("0.0 pp");
  });
});

describe("formatRate", () => {
  it("keeps one decimal", () => {
    expect(formatRate(52.8, "en-US")).toBe("52.8%");
  });
});
