import { describe, expect, it } from "vitest";

import { brandfetchLogoURL, brandInitial } from "@/components/blocks/brand-icon";

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
