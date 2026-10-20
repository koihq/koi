package koidomain

import (
	"testing"
	"time"
)

func TestBillingEntitlementEndsAfterPaidPeriodAndOnPaymentFailure(t *testing.T) {
	now := time.Date(2026, time.October, 20, 12, 0, 0, 0, time.UTC)
	periodEnd := now.Add(24 * time.Hour)
	subscription := BillingSubscription{
		Quantity:          25,
		Status:            "active",
		CurrentPeriodEnd:  &periodEnd,
		CancelAtPeriodEnd: true,
	}
	if got := subscription.PromptLimit(now); got != 25 {
		t.Fatalf("canceled-at-period-end subscription should retain 25 prompts, got %d", got)
	}
	if got := subscription.PromptLimit(periodEnd); got != 0 {
		t.Fatalf("expired subscription should grant no prompts, got %d", got)
	}

	subscription.Status = "past_due"
	if got := subscription.PromptLimit(now); got != 0 {
		t.Fatalf("failed payment should grant no prompts, got %d", got)
	}

	subscription.Status = "trialing"
	subscription.TrialEnd = &periodEnd
	if got := subscription.PromptLimit(now); got != 5 {
		t.Fatalf("trial should grant only 5 prompts, got %d", got)
	}
}
