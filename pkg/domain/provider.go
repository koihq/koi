package domain

import "context"

// CollectedAnswer is the provider-independent answer consumed by visibility workers.
type CollectedAnswer struct {
	Text           string
	Markdown       string
	Entities       []NamedEntity
	Sources        []string
	CreditsCharged *int
}

// MentionInput is what the judge reads to decide whether an answer is about the brand.
type MentionInput struct {
	Brand  BrandSnapshot
	Prompt string
	Answer string
}

type MentionJudgment struct {
	Verdict MentionVerdict
	CostUSD *float64
}

// MentionJudge is optional: without one, a named brand stays a mention by name match alone.
type MentionJudge interface {
	JudgeMention(context.Context, MentionInput) (MentionJudgment, error)
}

type ProviderFailure struct {
	Code        string
	Retryable   bool
	AccountWide bool
	Status      int
}

// ProviderError classifies integration failures without exposing vendor client types.
type ProviderError interface {
	error
	Failure() ProviderFailure
}
