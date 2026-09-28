import { describe, expect, it } from "vitest";

import { confirmedFromStatus, tableSearchParams } from "@/hooks/use-table-state";

describe("table search state", () => {
  it("parses supported values and falls back for invalid values", () => {
    expect(tableSearchParams.page.parse("2")).toBe(2);
    expect(tableSearchParams.pageSize.parse("20")).toBe(20);
    expect(tableSearchParams.pageSize.parse("25")).toBeNull();
    expect(tableSearchParams.status.parse("confirmed")).toBe("confirmed");
    expect(tableSearchParams.status.parse("pending")).toBeNull();
  });

  it("maps status to the API confirmed filter", () => {
    expect(confirmedFromStatus("all")).toBeUndefined();
    expect(confirmedFromStatus("confirmed")).toBe(true);
    expect(confirmedFromStatus("unconfirmed")).toBe(false);
  });
});
