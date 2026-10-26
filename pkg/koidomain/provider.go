package koidomain

// CollectedAnswer is the provider-independent answer consumed by visibility workers.
type CollectedAnswer struct {
	Text           string
	Markdown       string
	Entities       []NamedEntity
	Sources        []string
	CreditsCharged *int
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
