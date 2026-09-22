import { describe, expect, it } from "vitest";

import {
  brandfetchLogoURL,
  brandInitial,
  competitorFormSchema,
  listsReady,
  promptFormSchema,
} from "@/components/blocks/visibility-lists";

describe("visibility list forms", () => {
  it("validates and trims prompts", () => {
    expect(promptFormSchema.parse({ text: "  What is Koi?  " })).toEqual({ text: "What is Koi?" });
    expect(promptFormSchema.safeParse({ text: " " }).success).toBe(false);
  });

  it("normalizes competitor domains and aliases", () => {
    expect(
      competitorFormSchema.parse({
        name: " Acme ",
        domain: " ACME.COM ",
        aliases: " Acme AI, , acme ai, Acme Labs ",
      }),
    ).toEqual({ name: "Acme", domain: "acme.com", aliases: ["Acme AI", "Acme Labs"] });
    expect(
      competitorFormSchema.safeParse({ name: "Acme", domain: "https://acme.com", aliases: "" })
        .success,
    ).toBe(false);
  });
});

describe("list readiness", () => {
  it("requires two non-empty fully confirmed lists", () => {
    expect(listsReady([], [{ confirmed: true }])).toBe(false);
    expect(listsReady([{ confirmed: true }], [])).toBe(false);
    expect(listsReady([{ confirmed: false }], [{ confirmed: true }])).toBe(false);
    expect(listsReady([{ confirmed: true }], [{ confirmed: false }])).toBe(false);
    expect(listsReady([{ confirmed: true }], [{ confirmed: true }])).toBe(true);
  });
});

describe("brand icon", () => {
  it("builds a direct Brandfetch CDN URL only with a public client ID", () => {
    expect(brandfetchLogoURL("acme.com", undefined)).toBeNull();
    expect(brandfetchLogoURL("acme.com", " public id ")).toBe(
      "https://cdn.brandfetch.io/domain/acme.com/w/64/h/64/fallback/404/type/icon?c=public%20id",
    );
  });

  it("provides a local lettermark fallback", () => {
    expect(brandInitial(" Acme ", "acme.com")).toBe("A");
    expect(brandInitial("", "beta.test")).toBe("B");
    expect(brandInitial("", "")).toBe("?");
  });
});
