import { describe, expect, it } from "vitest";

import { brandNameFromSite, countryFromLocale, parseBrandSite, siteFromEmail } from "./first-brand";

describe("first brand setup", () => {
  it("reads a website or a repository from one field", () => {
    expect(parseBrandSite(" https://www.Acme.com/pricing ")).toEqual({
      domain: "acme.com",
      githubRepo: "",
    });
    expect(parseBrandSite("acme.io")).toEqual({ domain: "acme.io", githubRepo: "" });
    expect(parseBrandSite("github.com/koihq/koi/")).toEqual({
      domain: "",
      githubRepo: "https://github.com/koihq/koi",
    });
    expect(parseBrandSite("")).toEqual({ domain: "", githubRepo: "" });
  });

  it("suggests a brand name from the site", () => {
    expect(brandNameFromSite("acme.com")).toBe("Acme");
    expect(brandNameFromSite("app.acme.com")).toBe("Acme");
    expect(brandNameFromSite("acme.co.uk")).toBe("Acme");
    expect(brandNameFromSite("https://github.com/koihq/koi")).toBe("Koi");
    expect(brandNameFromSite("")).toBe("");
  });

  it("suggests a site only from a work email", () => {
    expect(siteFromEmail("founder@Deja.dev")).toBe("deja.dev");
    expect(siteFromEmail("someone@gmail.com")).toBe("");
    expect(siteFromEmail("someone@libero.it")).toBe("");
    expect(siteFromEmail("not-an-email")).toBe("");
  });

  it("prefills the country only when the locale names one", () => {
    expect(countryFromLocale("it-IT")).toBe("IT");
    expect(countryFromLocale("en")).toBe("");
  });
});
