import { describe, expect, it } from "vitest";

import { competitorFormSchema } from "@/routes/_authenticated/_settings/settings/competitors/-utils/competitor-form-schema";

describe("competitorFormSchema", () => {
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
