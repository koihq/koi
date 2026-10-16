import { describe, expect, it } from "vitest";
import type { Competitor } from "@/api/competitors.api";
import type { Prompt } from "@/api/prompts.api";
import { mockChangePoints, mockRate, mockSeries } from "@/lib/mock-visibility";

const confirmed: Prompt = {
  id: "prompt-1",
  brandId: "brand-1",
  text: "Which tools are recommended?",
  source: "manual",
  confirmed: true,
};

const draft: Prompt = {
  ...confirmed,
  id: "prompt-2",
  text: "A draft awaiting review?",
  confirmed: false,
};

const competitor: Competitor = {
  id: "competitor-1",
  brandId: "brand-1",
  name: "Other",
  domain: "other.example",
  githubRepo: "",
  source: "manual",
  confirmed: true,
  aliases: [],
  additionalUrls: [],
};

describe("temporary visibility values", () => {
  it("excludes unconfirmed prompts and remains stable across reads", () => {
    const scope = { days: 30 as const, engines: [] };
    const expected = mockSeries("brand-1", [confirmed], scope);

    expect(mockSeries("brand-1", [confirmed, draft], scope)).toEqual(expected);
    expect(mockRate("brand-1", [confirmed, draft], scope)).toBe(
      expected.reduce((total, point) => total + point.rate, 0) / expected.length,
    );
    expect(mockSeries("brand-1", [draft], scope)).toEqual([]);
    expect(mockRate("brand-1", [draft], scope)).toBeNull();
  });

  it("uses the same scope for headline, engine filters, and competitor comparison", () => {
    const scope = { days: 7 as const, engines: ["chatgpt" as const] };
    const series = mockSeries("brand-1", [confirmed], scope);

    expect(series).toHaveLength(7);
    expect(mockRate("brand-1", [confirmed], scope)).toBeGreaterThan(0);
    expect(mockRate("brand-1", [confirmed], scope, competitor)).not.toBe(
      mockRate("brand-1", [confirmed], scope),
    );
    expect(Number.isFinite(mockChangePoints("brand-1", [confirmed], scope))).toBe(true);
  });
});
