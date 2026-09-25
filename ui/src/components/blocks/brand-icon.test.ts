import { describe, expect, it } from "vitest";

import { brandInitial, logoDevURL } from "@/components/blocks/brand-icon";

describe("brand icon", () => {
  it("builds a Logo.dev CDN URL only with a publishable key", () => {
    expect(logoDevURL("acme.com", undefined)).toBeNull();
    expect(logoDevURL("acme.com", " public key ")).toBe(
      "https://img.logo.dev/acme.com?token=public%20key&size=64&format=webp",
    );
  });

  it("provides a local lettermark fallback", () => {
    expect(brandInitial(" Acme ", "acme.com")).toBe("A");
    expect(brandInitial("", "beta.test")).toBe("B");
    expect(brandInitial("", "")).toBe("?");
  });
});
