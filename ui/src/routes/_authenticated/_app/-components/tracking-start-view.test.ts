import { describe, expect, it } from "vitest";
import { trackingStartView } from "./tracking-start-view";

describe("trackingStartView", () => {
  it("offers the start once a prompt exists and no draft is running", () => {
    expect(trackingStartView({ collectionAvailable: true, drafting: false, prompts: 5 })).toBe(
      "ready",
    );
  });

  it("waits for a running draft even with manual prompts", () => {
    expect(trackingStartView({ collectionAvailable: true, drafting: true, prompts: 2 })).toBe(
      "drafting",
    );
  });

  it("asks for a prompt when a draft fails or finds nothing", () => {
    expect(trackingStartView({ collectionAvailable: true, drafting: false, prompts: 0 })).toBe(
      "empty",
    );
  });

  it("never offers the start without live collection", () => {
    expect(trackingStartView({ collectionAvailable: false, drafting: false, prompts: 5 })).toBe(
      "unavailable",
    );
  });
});
