package domain

import (
	"context"
	"slices"
)

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

// RecordOnboardingStep sends a setup step to analytics, so the funnel can show where people stop.
// It sends the step name only, nothing about the brand or its answers.
func RecordOnboardingStep(ctx context.Context, analytics Analytics, step string) error {
	if !slices.Contains(onboardingSteps, step) {
		return Validation(FieldError{Field: "step", Code: FieldInvalid})
	}

	trackIdentity(ctx, orNop(analytics), "onboarding_step", map[string]any{"step": step})

	return nil
}
