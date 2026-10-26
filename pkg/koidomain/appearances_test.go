package koidomain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAppearancesMatchAliasesAndSpecificURLs(t *testing.T) {
	miss := OutcomeMiss
	competitors := []Competitor{
		{
			BaseEntity: BaseEntity{
				ID: uuid.New(),
			},
			Name: "Acme Analytics", Domain: "acme.dev", Confirmed: true,
			Aliases:        []string{"Acme", "Straße"},
			AdditionalURLs: []string{"https://acme.tools", "https://github.com/acme/analytics"},
		},
		{
			BaseEntity: BaseEntity{
				ID: uuid.New(),
			},
			Name: "Docs", Domain: "docs.acme.dev", Confirmed: true,
		},
	}
	result := VisibilityResult{
		Outcome: &miss, RawAnswer: "STRASSE and ACME work here; Acmeish does not.",
		Sources: []string{
			"https://acme.tools/features", "https://www.acme.dev/about", "https://acme.tools/features",
			"https://docs.acme.dev/page", "https://github.com/acme/analytics/blob/main/README.md",
			"https://acme.dev.evil.test", "https://github.com/acme/analytics-fork", "javascript:alert(1)",
		},
	}
	appearances := MatchAppearances(result, competitors, BrandSnapshot{Name: "Koi"})
	require.Len(t, appearances, 2)
	require.Equal(t, competitors[0].ID, *appearances[0].CompetitorID)
	require.True(t, appearances[0].Confirmed)
	require.Len(t, appearances[0].Evidence, 5) // Two aliases and three distinct official citations.
	require.Equal(t, []AppearanceEvidence{{Kind: "citation", Value: "https://docs.acme.dev/page"}}, appearances[1].Evidence)
	require.Equal(t, OutcomeMiss, *result.Outcome)

	result.RawAnswer, result.Sources = "Acmeish", nil
	require.Empty(t, MatchAppearances(result, competitors, BrandSnapshot{}))
}

func TestAmbiguousAliasesNeedURLIdentity(t *testing.T) {
	competitors := []Competitor{
		{Name: "First", Domain: "first.dev", Aliases: []string{"Shared"}},
		{Name: "Second", Domain: "second.dev", Aliases: []string{"Shared"}},
	}
	index, ambiguous := ResolveObservation(AppearanceObservation{Name: " SHARED "}, competitors)
	require.Equal(t, -1, index)
	require.True(t, ambiguous)
	index, ambiguous = ResolveObservation(AppearanceObservation{Name: "Shared", URLs: []string{"https://second.dev"}}, competitors)
	require.Equal(t, 1, index)
	require.False(t, ambiguous)
}

func TestConservativeObservationExtraction(t *testing.T) {
	result := VisibilityResult{
		NamedEntities: []NamedEntity{
			{Type: "software", Name: "Alpha"}, {Type: "company", Name: " ALPHA "},
			{Type: "person", Name: "A person"}, {Type: "product", Name: "Name only"},
		},
		AnswerMarkdown: "1. **[Alpha](https://alpha.dev/)**\n" +
			"## [Beta](https://github.com/owner/beta)\n" +
			"| [Gamma](https://gamma.dev) |\n" +
			"Read [An article](https://news.dev/article).\n" +
			"- [News](https://news.dev/article)\n" +
			"- [1](https://citations.dev)\n" +
			"- [Unsafe](https://user:pass@unsafe.dev)\n" +
			"- [Profile](https://github.com/owner)\n",
	}
	observations := ExtractObservations(result)
	require.Len(t, observations, 4)
	require.Equal(t, "alpha", observations[0].Key)
	require.Equal(t, []string{"https://alpha.dev"}, observations[0].URLs)
	require.Len(t, observations[0].Evidence, 4)
	require.Empty(t, observations[1].URLs)
	require.Equal(t, []string{"https://github.com/owner/beta"}, observations[2].URLs)
	require.True(t, ObservationIsBrand(observations[0], BrandSnapshot{Domain: "alpha.dev"}))

	miss := OutcomeMiss
	result.Outcome, result.Observations = &miss, observations
	appearances := MatchAppearances(result, nil, BrandSnapshot{Name: "Alpha"})
	require.Len(t, appearances, 3)
	require.Nil(t, appearances[0].CompetitorID)
	require.False(t, appearances[0].Confirmed)
	errorOutcome := OutcomeError
	result.Outcome = &errorOutcome
	require.Empty(t, MatchAppearances(result, nil, BrandSnapshot{}))
}

func TestObservedIdentityValidationAndNormalization(t *testing.T) {
	competitor := Competitor{
		BrandID: uuid.New(), Name: " Name only ", Source: CompetitorSourceObserved,
		Aliases:        []string{" Alias ", "ALIAS"},
		AdditionalURLs: []string{"https://www.example.dev/docs?q=1", "http://example.dev", "https://github.com/Owner/Repo/"},
	}
	competitor.Normalize()
	require.NoError(t, competitor.Validate())
	require.Equal(t, []string{"Alias"}, competitor.Aliases)
	require.Equal(t, []string{"https://example.dev", "https://github.com/owner/repo"}, competitor.AdditionalURLs)
	competitor.Confirmed = true
	require.ErrorIs(t, competitor.Validate(), ErrValidation)
	competitor.Domain = "example.dev"
	require.NoError(t, competitor.Validate())
	competitor.AdditionalURLs = []string{"javascript:alert(1)"}
	require.ErrorIs(t, competitor.Validate(), ErrValidation)
}
