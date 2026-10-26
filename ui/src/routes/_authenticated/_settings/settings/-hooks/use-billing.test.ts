import { describe, expect, it } from "vitest";

import type { BillingStatus, BillingSubscription } from "@/api/generated";
import { billingNeedsConfirmation, billingSelection, billingUpgradeExpired } from "./use-billing";

const subscription: BillingSubscription = {
  organizationId: "org_1",
  quantity: 25,
  status: "active",
  cancelAtPeriodEnd: false,
};
const status: BillingStatus = {
  subscription,
  promptLimit: 25,
  trialAvailable: false,
};

describe("billing state", () => {
  it("selects the scheduled allowance before the current one", () => {
    expect(billingSelection(undefined, 5)).toBe(5);
    expect(billingSelection(status, 5)).toBe(25);
    expect(
      billingSelection({ ...status, subscription: { ...subscription, pendingQuantity: 10 } }, 5),
    ).toBe(10);
  });

  it("waits only while checkout or upgrade confirmation is outstanding", () => {
    expect(billingNeedsConfirmation(undefined, undefined, true)).toBe(true);
    expect(billingNeedsConfirmation(status, undefined, true)).toBe(false);
    expect(
      billingNeedsConfirmation(status, { quantity: 50, effective: "after_payment" }, false),
    ).toBe(true);
    expect(billingNeedsConfirmation(status, { quantity: 5, effective: "next_cycle" }, false)).toBe(
      false,
    );
    expect(
      billingNeedsConfirmation(status, { quantity: 25, effective: "after_payment" }, false),
    ).toBe(false);
  });

  it("expires an unapplied upgrade without hiding a completed one", () => {
    const change = { quantity: 50, effective: "after_payment" as const, expiresAt: new Date(1000) };
    expect(billingUpgradeExpired(status, change, 2000)).toBe(true);
    expect(billingUpgradeExpired(status, change, 500)).toBe(false);
    expect(
      billingUpgradeExpired(
        { ...status, subscription: { ...subscription, quantity: 50 } },
        change,
        2000,
      ),
    ).toBe(false);
    expect(billingNeedsConfirmation(status, change, false)).toBe(false);
  });
});
