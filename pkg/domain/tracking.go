package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TrackingReadiness struct {
	CollectionEnabled bool
	Confirmed         bool
	HasPrompts        bool
}

// TrackingCommander confirms a brand's lists once and starts its daily collection.
type TrackingCommander interface {
	Start(context.Context, uuid.UUID) error
}

type trackingCommander struct{ store Store }

func NewTrackingCommander(store Store) TrackingCommander {
	return &trackingCommander{store: store}
}

func (c *trackingCommander) Start(ctx context.Context, brandID uuid.UUID) error {
	return c.store.VisibilityRepo().StartTracking(ctx, brandID)
}

func TrackingBlockedReason(r TrackingReadiness) string {
	switch {
	case !r.CollectionEnabled:
		return "collection_unavailable"
	case !r.Confirmed:
		return "start_required"
	case !r.HasPrompts:
		return "prompts_required"
	default:
		return ""
	}
}

// NextVisibilitySchedule skips missed slots without moving the UTC anchor.
func NextVisibilitySchedule(anchor, now time.Time) (slot, next time.Time) {
	anchor = anchor.UTC()
	elapsed := max(time.Duration(0), now.Sub(anchor))
	slot = anchor.Add(elapsed / (24 * time.Hour) * (24 * time.Hour))
	return slot, slot.Add(24 * time.Hour)
}
