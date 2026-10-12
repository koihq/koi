package koidomain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTrackingReadinessPrecedence(t *testing.T) {
	r := TrackingReadiness{}
	require.Equal(t, "collection_unavailable", TrackingBlockedReason(r))
	r.CollectionEnabled = true
	require.Equal(t, "reviews_required", TrackingBlockedReason(r))
	r.PromptsReviewed, r.CompetitorsReviewed = true, true
	require.Equal(t, "prompts_required", TrackingBlockedReason(r))
	r.HasPrompts, r.GenerationActive = true, true
	require.Equal(t, "generation_active", TrackingBlockedReason(r))
	r.GenerationActive = false
	require.Empty(t, TrackingBlockedReason(r))
}

func TestVisibilityScheduleKeepsUTCAnchor(t *testing.T) {
	anchor := time.Date(2026, 10, 24, 23, 30, 0, 0, time.FixedZone("local", 2*60*60))
	for _, elapsed := range []time.Duration{0, 23 * time.Hour, 24 * time.Hour, 6*24*time.Hour + time.Hour} {
		slot, next := NextVisibilitySchedule(anchor, anchor.Add(elapsed))
		require.Equal(t, anchor.UTC().Add(elapsed/(24*time.Hour)*(24*time.Hour)), slot)
		require.Equal(t, time.UTC, slot.Location())
		require.Equal(t, slot.Add(24*time.Hour), next)
	}
}
