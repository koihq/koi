import { describe, expect, it } from "vitest";

import { consentDomain, readConsent } from "@/analytics/consent";

describe("readConsent", () => {
  it("reads the shared choice among other cookies", () => {
    expect(readConsent("theme=dark; koi_consent=granted; other=1")).toBe("granted");
    expect(readConsent("koi_consent=denied")).toBe("denied");
  });

  it("treats a missing or unknown value as no choice yet", () => {
    expect(readConsent("")).toBeNull();
    expect(readConsent("koi_consent=maybe")).toBeNull();
    expect(readConsent("not_koi_consent=granted")).toBeNull();
  });
});

describe("consentDomain", () => {
  it("shares the cookie between the landing and the app", () => {
    expect(consentDomain("app.getkoi.co")).toBe("; Domain=.getkoi.co");
    expect(consentDomain("getkoi.co")).toBe("; Domain=.getkoi.co");
  });

  it("keeps a host-only cookie on local addresses", () => {
    expect(consentDomain("localhost")).toBe("");
    expect(consentDomain("127.0.0.1")).toBe("");
  });
});
