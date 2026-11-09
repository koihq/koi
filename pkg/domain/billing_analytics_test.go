package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBillingTransitions(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	stored := func(status string, quantity int, cancel bool) *BillingSubscription {
		subscription := &BillingSubscription{
			StripeSubscriptionID: "sub_1",
			Status:               status,
			Quantity:             quantity,
			CancelAtPeriodEnd:    cancel,
		}
		subscription.CreatedAt = now.Add(-10 * 24 * time.Hour)

		return subscription
	}
	snapshot := func(status string, quantity int, cancel bool) BillingSnapshot {
		return BillingSnapshot{ID: "sub_1", Status: status, Quantity: quantity, CancelAtPeriodEnd: cancel}
	}

	tests := []struct {
		name     string
		previous *BillingSubscription
		next     BillingSnapshot
		want     []string
	}{
		{"checkout with trial", nil, snapshot("trialing", 25, false), []string{"trial_started"}},
		{"checkout without trial", nil, snapshot("active", 25, false), []string{"subscription_activated"}},
		{"trial converts", stored("trialing", 25, false), snapshot("active", 25, false), []string{"subscription_activated"}},
		{"trial expires", stored("trialing", 25, false), snapshot("canceled", 25, false), []string{"subscription_ended"}},
		{"payment fails", stored("active", 25, false), snapshot("past_due", 25, false), []string{"payment_failed"}},
		{"payment recovers", stored("past_due", 25, false), snapshot("active", 25, false), nil},
		{"unpaid ends", stored("past_due", 25, false), snapshot("canceled", 25, false), []string{"subscription_ended"}},
		{"upgrade", stored("active", 25, false), snapshot("active", 50, false), []string{"allowance_changed"}},
		{"trial allowance change", stored("trialing", 25, false), snapshot("trialing", 50, false), nil},
		{"cancel scheduled", stored("active", 25, false), snapshot("active", 25, true), []string{"subscription_cancel_scheduled"}},
		{"cancel undone", stored("active", 25, true), snapshot("active", 25, false), []string{"subscription_reactivated"}},
		{"scheduled cancel ends", stored("active", 25, true), snapshot("canceled", 25, true), []string{"subscription_ended"}},
		{"no change", stored("active", 25, false), snapshot("active", 25, false), nil},
		{"already ended", stored("canceled", 25, false), snapshot("canceled", 25, false), nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var names []string
			for _, event := range billingTransitions(test.previous, test.next, now) {
				names = append(names, event.name)
			}

			require.Equal(t, test.want, names)
		})
	}
}

func TestBillingTransitionsDescribeTheChange(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	previous := &BillingSubscription{StripeSubscriptionID: "sub_1", Status: "trialing", Quantity: 25}
	previous.CreatedAt = now.Add(-7 * 24 * time.Hour)

	ended := billingTransitions(previous, BillingSnapshot{ID: "sub_1", Status: "canceled", Quantity: 25}, now)
	require.Equal(t, map[string]any{"reason": "trial_expired", "tenure_days": 7}, ended[0].props)

	activated := billingTransitions(previous, BillingSnapshot{ID: "sub_1", Status: "active", Quantity: 25}, now)
	require.Equal(t, true, activated[0].props["from_trial"])
	require.Equal(t, int64(4500), activated[0].props["amount_cents"])

	// A new subscription after a canceled one starts from nothing.
	previous.Status = "canceled"
	restarted := billingTransitions(previous, BillingSnapshot{ID: "sub_2", Status: "trialing", Quantity: 25}, now)
	require.Equal(t, "trial_started", restarted[0].name)
}
