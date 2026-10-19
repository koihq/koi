import { describe, expect, it, vi } from "vitest";
import { collectPages } from "@/lib/collect-pages";

describe("collectPages", () => {
  it("collects every backend page in order", async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: ["first", "second"], hasNext: true })
      .mockResolvedValueOnce({ items: ["third"], hasNext: false });

    await expect(collectPages<string>(fetchPage)).resolves.toEqual(["first", "second", "third"]);
    expect(fetchPage.mock.calls).toEqual([[1], [2]]);
  });

  it("returns an empty list and propagates request failures", async () => {
    await expect(collectPages(async () => ({ items: [], hasNext: false }))).resolves.toEqual([]);

    const failure = new Error("page unavailable");
    await expect(collectPages(() => Promise.reject(failure))).rejects.toBe(failure);
  });
});
