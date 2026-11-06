package domain

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
	if got := subscription.PromptLimit(now); got != TrialPromptLimit {
		t.Fatalf("trial should grant only %d prompts, got %d", TrialPromptLimit, got)
	}
}

func TestGenerationBudgetCapsTheDraftAndLeavesRoom(t *testing.T) {
	for _, test := range []struct {
		name   string
		limit  int
		stored int
		want   int
	}{
		{"first look", FirstLookPromptLimit, 0, InitialDraftPromptLimit},
		{"trial", TrialPromptLimit, 0, InitialDraftPromptLimit},
		{"trial with little room", TrialPromptLimit, 7, 3},
		{"trial at its limit", TrialPromptLimit, 10, 0},
		{"paid allowance above the cap", 100, 0, InitialDraftPromptLimit},
		{"paid over its allowance", 100, 120, 0},
		{"no allowance", 0, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := GenerationBudget(test.limit, test.stored); got != test.want {
				t.Fatalf("GenerationBudget = %d, want %d", got, test.want)
			}
		})
	}
}

func TestEffectivePromptLimitGrantsTheFirstLookOnlyBeforeAnyTrial(t *testing.T) {
	now := time.Date(2026, time.October, 20, 12, 0, 0, 0, time.UTC)
	end := now.Add(24 * time.Hour)
	used := now.Add(-time.Hour)

	for _, test := range []struct {
		name        string
		trialUsedAt *time.Time
		sub         BillingSubscription
		firstLook   bool
		want        int
	}{
		{"new organization", nil, BillingSubscription{}, true, FirstLookPromptLimit},
		{"trial used, no subscription left", &used, BillingSubscription{}, false, 0},
		{"canceled", &used, BillingSubscription{Status: "canceled", Quantity: 25}, false, 0},
		{"canceled without a marked trial", nil, BillingSubscription{Status: "canceled", Quantity: 25}, false, 0},
		{"trial", &used, BillingSubscription{Status: "trialing", TrialEnd: &end}, false, TrialPromptLimit},
		{"active", &used, BillingSubscription{Status: "active", Quantity: 25, CurrentPeriodEnd: &end}, false, 25},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := IsFirstLook(test.sub, test.trialUsedAt); got != test.firstLook {
				t.Fatalf("IsFirstLook = %v, want %v", got, test.firstLook)
			}
			if got := EffectivePromptLimit(test.sub, test.trialUsedAt, now); got != test.want {
				t.Fatalf("EffectivePromptLimit = %d, want %d", got, test.want)
			}
		})
	}
}
