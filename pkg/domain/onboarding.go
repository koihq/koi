package domain

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// OnboardingEvent records the first time an organization reached a setup step, so the funnel can
// show where people stop. It holds no content from the brand or its answers.
type OnboardingEvent struct {
	BaseEntity
	OrganizationID uuid.UUID     `json:"-" gorm:"type:uuid;not null;uniqueIndex:idx_onboarding_events_step,priority:1"`
	Organization   *Organization `json:"-" gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE"`
	Step           string        `json:"-" gorm:"type:varchar(32);not null;uniqueIndex:idx_onboarding_events_step,priority:2"`
}

func (OnboardingEvent) TableName() string { return "onboarding_events" }

var onboardingSteps = []string{
	"choice",
	"website",
	"confirm",
	"agent_connect",
	"agent_prompt",
	"first_look",
	"first_answer",
	"trial_clicked",
	"agent_handoff",
}

type OnboardingRepository interface {
	RecordStep(ctx context.Context, step string) error
}

func RecordOnboardingStep(ctx context.Context, store Store, step string) error {
	if !slices.Contains(onboardingSteps, step) {
		return Validation(FieldError{Field: "step", Code: FieldInvalid})
	}

	return store.OnboardingRepo().RecordStep(ctx, step)
}
