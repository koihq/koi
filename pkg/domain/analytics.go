package domain

import (
	"context"

	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/auth"
)

// Analytics sends product events to an analytics service. It never fails the caller: an
// implementation queues the event and drops it on error. Events carry ids and counts, never prompt
// text, answers or competitor names.
type Analytics interface {
	Track(userID, organizationID uuid.UUID, event string, props map[string]any)
}

// NopAnalytics is used when no analytics service is configured, as on a self-hosted instance.
type NopAnalytics struct{}

func (NopAnalytics) Track(uuid.UUID, uuid.UUID, string, map[string]any) {}

// trackIdentity sends an event for the user making the request. Requests without an identity, such
// as webhooks, send nothing.
func trackIdentity(ctx context.Context, analytics Analytics, event string, props map[string]any) {
	identity, ok := auth.FromContext(ctx)
	if !ok {
		return
	}

	analytics.Track(identity.UserID, identity.OrganizationID, event, props)
}

func orNop(analytics Analytics) Analytics {
	if analytics == nil {
		return NopAnalytics{}
	}

	return analytics
}
