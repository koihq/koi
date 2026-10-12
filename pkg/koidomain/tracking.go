package koidomain

import "time"

type TrackingReadiness struct {
	CollectionEnabled   bool
	PromptsReviewed     bool
	CompetitorsReviewed bool
	HasPrompts          bool
	GenerationActive    bool
}

func TrackingBlockedReason(r TrackingReadiness) string {
	switch {
	case !r.CollectionEnabled:
		return "collection_unavailable"
	case !r.PromptsReviewed || !r.CompetitorsReviewed:
		return "reviews_required"
	case !r.HasPrompts:
		return "prompts_required"
	case r.GenerationActive:
		return "generation_active"
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
