import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const posthog = vi.hoisted(() => ({
  __loaded: false,
  init: vi.fn(),
  identify: vi.fn(),
  opt_out_capturing: vi.fn(),
  opt_in_capturing: vi.fn(),
  startSessionRecording: vi.fn(),
  reset: vi.fn(),
}));

vi.mock("posthog-js", () => ({ default: posthog }));

import { createAnalytics } from "@/analytics/analyticsProvider";
import { noopAnalytics } from "@/analytics/noop.adapter";

const config = {
  key: "phc_test",
  apiHost: "/ingest",
  uiHost: "https://eu.posthog.com",
  excludedEmails: ["Team@Example.com"],
};

const settle = () => new Promise((resolve) => setTimeout(resolve, 0));

describe("createAnalytics", () => {
  beforeEach(() => {
    vi.stubGlobal("document", { cookie: "" });
    vi.stubGlobal("window", { location: { hostname: "app.getkoi.co", protocol: "https:" } });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("sends nothing without a configuration", () => {
    expect(createAnalytics(null)).toBe(noopAnalytics);
  });

  it("waits for consent before capturing", async () => {
    createAnalytics(config).start();
    await settle();

    expect(posthog.init).toHaveBeenCalledWith(
      "phc_test",
      expect.objectContaining({
        api_host: "/ingest",
        opt_out_capturing_by_default: true,
        cookieless_mode: "on_reject",
        disable_session_recording: true,
      }),
    );
  });

  it("opts the team's own accounts out instead of identifying them", async () => {
    const analytics = createAnalytics(config);

    analytics.identify({ id: "team", email: "team@example.com" });
    analytics.identify({ id: "customer", email: "customer@example.com" });
    await settle();

    expect(posthog.opt_out_capturing).toHaveBeenCalledOnce();
    expect(posthog.identify).toHaveBeenCalledExactlyOnceWith("customer", {
      email: "customer@example.com",
    });
  });

  it("records a session only after consent", async () => {
    const analytics = createAnalytics(config);

    analytics.startReplay();
    await settle();
    expect(posthog.startSessionRecording).not.toHaveBeenCalled();

    analytics.setConsent("granted");
    analytics.startReplay();
    await settle();

    expect(document.cookie).toContain("koi_consent=granted");
    expect(document.cookie).toContain("Domain=.getkoi.co");
    expect(posthog.opt_in_capturing).toHaveBeenCalledOnce();
    expect(posthog.startSessionRecording).toHaveBeenCalledOnce();
  });
});
