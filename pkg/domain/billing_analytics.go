package domain

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/pricing"
)

// billingEvent is a subscription change worth counting in the conversion and churn funnels.
type billingEvent struct {
	name  string
	props map[string]any
}

// billingTransitions compares the stored subscription with the one Stripe returned. previous is nil
// for a subscription Koi has not stored yet.
func billingTransitions(previous *BillingSubscription, next BillingSnapshot, now time.Time) []billingEvent {
	before := BillingSubscription{}
	if previous != nil && previous.StripeSubscriptionID == next.ID {
		before = *previous
	}

	tenure := map[string]any{"tenure_days": tenureDays(before, now)}
	var events []billingEvent

	if next.Status == "trialing" && before.Status != "trialing" {
		events = append(events, billingEvent{"trial_started", map[string]any{"quantity": next.Quantity}})
	}

	if next.Status == "active" && before.Status != "active" && before.Status != "past_due" {
		props := map[string]any{
			"quantity":   next.Quantity,
			"from_trial": before.Status == "trialing",
		}
		if amount, err := pricing.Current().Quote(next.Quantity); err == nil {
			props["amount_cents"] = amount
		}

		events = append(events, billingEvent{"subscription_activated", props})
	}

	if next.Status == "past_due" && before.Status != "past_due" {
		events = append(events, billingEvent{"payment_failed", nil})
	}

	if before.Status == "active" && next.Status == "active" && next.Quantity != before.Quantity {
		events = append(events, billingEvent{"allowance_changed", map[string]any{
			"from_quantity": before.Quantity,
			"to_quantity":   next.Quantity,
		}})
	}

	if next.CancelAtPeriodEnd && !before.CancelAtPeriodEnd && !billingEnded(next.Status) {
		events = append(events, billingEvent{"subscription_cancel_scheduled", tenure})
	}

	if !next.CancelAtPeriodEnd && before.CancelAtPeriodEnd && !billingEnded(next.Status) {
		events = append(events, billingEvent{"subscription_reactivated", nil})
	}

	if billingEnded(next.Status) && before.Status != "" && !billingEnded(before.Status) {
		events = append(events, billingEvent{"subscription_ended", map[string]any{
			"reason":      endReason(before.Status),
			"tenure_days": tenureDays(before, now),
		}})
	}

	return events
}

func billingEnded(status string) bool {
	return status == "canceled" || status == "incomplete_expired"
}

func endReason(previousStatus string) string {
	switch previousStatus {
	case "trialing":
		return "trial_expired"
	case "past_due", "unpaid", "incomplete":
		return "payment_failed"
	default:
		return "voluntary"
	}
}

func tenureDays(subscription BillingSubscription, now time.Time) int {
	if subscription.CreatedAt.IsZero() {
		return 0
	}

	return int(math.Floor(now.Sub(subscription.CreatedAt).Hours() / 24))
}

type billingEventsKey struct{}

type pendingBillingEvents struct {
	organizationID uuid.UUID
	events         []billingEvent
}

// atomic runs fn in a transaction and sends the billing events it recorded only after the
// transaction commits, so a rollback or a retried webhook never counts a change twice.
func (s *billingService) atomic(ctx context.Context, fn func(ctx context.Context, store Store) error) error {
	pending := &pendingBillingEvents{}
	ctx = context.WithValue(ctx, billingEventsKey{}, pending)

	err := s.store.Atomic(ctx, func(store Store) error {
		return fn(ctx, store)
	})
	if err != nil || len(pending.events) == 0 {
		return err
	}

	s.sendBillingEvents(ctx, pending)

	return nil
}

func recordBillingEvents(ctx context.Context, organizationID uuid.UUID, events []billingEvent) {
	pending, ok := ctx.Value(billingEventsKey{}).(*pendingBillingEvents)
	if !ok || len(events) == 0 {
		return
	}

	pending.organizationID = organizationID
	pending.events = append(pending.events, events...)
}

// sendBillingEvents attributes billing to the organization's owner, who signed up and pays.
func (s *billingService) sendBillingEvents(ctx context.Context, pending *pendingBillingEvents) {
	ownerID, err := s.store.BillingRepo().OwnerUserID(ctx, pending.organizationID)
	if err != nil {
		slog.Warn("billing analytics skipped", "organization_id", pending.organizationID, "err", err)

		return
	}

	for _, event := range pending.events {
		s.analytics.Track(ownerID, pending.organizationID, event.name, event.props)
	}
}
