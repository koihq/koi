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

func TestGenerationBudgetLimitsOnlyTheInitialTrialDraft(t *testing.T) {
	now := time.Date(2026, time.October, 20, 12, 0, 0, 0, time.UTC)
	end := now.Add(24 * time.Hour)
	trial := BillingSubscription{Quantity: 100, Status: "trialing", TrialEnd: &end}
	paid := BillingSubscription{Quantity: 15, Status: "active", CurrentPeriodEnd: &end}

	for _, test := range []struct {
		name         string
		subscription BillingSubscription
		mode         GenerationMode
		stored       int
		want         int
	}{
		{"trial initial draft", trial, GenerationInitial, 0, 5},
		{"trial initial draft with little room", trial, GenerationInitial, 7, 3},
		{"trial initial draft at the limit", trial, GenerationInitial, 10, 0},
		{"trial drafting more prompts", trial, GenerationPrompts, 5, 5},
		{"paid initial draft", paid, GenerationInitial, 0, 15},
		{"paid over its allowance", paid, GenerationPrompts, 20, 0},
		{"no subscription", BillingSubscription{}, GenerationInitial, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.subscription.GenerationBudget(now, test.mode, test.stored); got != test.want {
				t.Fatalf("GenerationBudget = %d, want %d", got, test.want)
			}
		})
	}
}
