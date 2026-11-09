package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/auth"
)

type trackedEvent struct {
	userID         uuid.UUID
	organizationID uuid.UUID
	name           string
	props          map[string]any
}

type trackedEvents struct {
	events []trackedEvent
}

func (a *trackedEvents) Track(userID, organizationID uuid.UUID, event string, props map[string]any) {
	a.events = append(a.events, trackedEvent{userID, organizationID, event, props})
}

func TestRecordOnboardingStepSendsTheStepOnly(t *testing.T) {
	identity := auth.Identity{UserID: uuid.New(), OrganizationID: uuid.New(), Role: RoleOwner}
	ctx := auth.WithIdentity(t.Context(), identity)
	analytics := &trackedEvents{}

	require.NoError(t, RecordOnboardingStep(ctx, analytics, "website"))

	err := RecordOnboardingStep(ctx, analytics, "unknown")
	require.ErrorIs(t, err, ErrValidation)

	require.Equal(t, []trackedEvent{{
		userID:         identity.UserID,
		organizationID: identity.OrganizationID,
		name:           "onboarding_step",
		props:          map[string]any{"step": "website"},
	}}, analytics.events)
}

func TestRecordOnboardingStepWithoutAnalytics(t *testing.T) {
	ctx := auth.WithIdentity(t.Context(), auth.Identity{UserID: uuid.New(), OrganizationID: uuid.New()})

	require.NoError(t, RecordOnboardingStep(ctx, nil, "website"))
}
