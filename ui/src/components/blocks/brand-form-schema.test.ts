import { describe, expect, it } from "vitest";

import { brandFormSchema } from "@/components/blocks/brand-form-schema";

const valid = {
  name: "Koi",
  domain: "koi.dev",
  githubRepo: "",
  country: "IT",
  description: "AI visibility",
};

function issuePaths(input: typeof valid): string[] {
  const result = brandFormSchema.safeParse(input);
  expect(result.success).toBe(false);

  return result.error?.issues.map((issue) => String(issue.path[0])) ?? [];
}

describe("brandFormSchema", () => {
  it("trims and normalizes a valid request", () => {
    expect(
      brandFormSchema.parse({
        name: " Koi ",
        domain: " koi.dev ",
        githubRepo: " https://github.com/koihq/koi ",
        country: " it ",
        description: " AI visibility ",
      }),
    ).toEqual({
      name: "Koi",
      domain: "koi.dev",
      githubRepo: "https://github.com/koihq/koi",
      country: "IT",
      description: "AI visibility",
    });
  });

  it("requires at least one identifier", () => {
    expect(issuePaths({ ...valid, domain: "", githubRepo: "" })).toEqual(["domain", "githubRepo"]);
  });

  it.each([
    "https://koi.dev",
    "koi..dev",
    "-koi.dev",
    "127.0.0.1",
  ])("rejects invalid domain %s", (domain) => {
    expect(issuePaths({ ...valid, domain })).toContain("domain");
  });

  it.each([
    "http://github.com/koihq/koi",
    "https://example.com/koihq/koi",
    "https://github.com/koihq",
    "https://github.com/koihq/koi/issues",
    "https://github.com/koihq/koi?tab=readme",
    "https://github.com:443/koihq/koi",
    "https://github.com/koihq/%6boi",
  ])("rejects invalid GitHub repository %s", (githubRepo) => {
    expect(issuePaths({ ...valid, domain: "", githubRepo })).toContain("githubRepo");
  });

  it("requires a name", () => {
    expect(issuePaths({ ...valid, name: "  " })).toContain("name");
  });

  it("requires a two-letter country", () => {
    expect(issuePaths({ ...valid, country: "" })).toContain("country");
  });
});
