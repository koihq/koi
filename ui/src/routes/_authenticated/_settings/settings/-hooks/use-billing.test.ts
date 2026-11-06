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
  usedPrompts: 0,
  trialAvailable: false,
};

describe("billing state", () => {
  it("selects the scheduled allowance before the current one", () => {
    expect(billingSelection(undefined, 50)).toBe(50);
    expect(billingSelection(status, 50)).toBe(25);
    expect(
      billingSelection({ ...status, subscription: { ...subscription, pendingQuantity: 15 } }, 50),
    ).toBe(15);
  });

  it("expires an unpaid plan start even though the quantity does not change", () => {
    const trial = (pendingUpgradeExpiresAt: Date) => ({
      ...status,
      subscription: {
        ...subscription,
        status: "trialing",
        pendingUpgradeQuantity: 25,
        pendingUpgradeExpiresAt,
      },
    });

    expect(billingUpgradeExpired(trial(new Date(500)), undefined, 1000)).toBe(true);
    expect(billingUpgradeExpired(trial(new Date(2000)), undefined, 1000)).toBe(false);
  });

  it("waits only while checkout or upgrade confirmation is outstanding", () => {
    expect(billingNeedsConfirmation(undefined, undefined, true)).toBe(true);
    expect(billingNeedsConfirmation(status, undefined, true)).toBe(false);
    expect(
      billingNeedsConfirmation(status, { quantity: 50, effective: "after_payment" }, false),
    ).toBe(true);
    expect(billingNeedsConfirmation(status, { quantity: 15, effective: "next_cycle" }, false)).toBe(
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
