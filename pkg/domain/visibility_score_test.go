package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestScoreVisibility(t *testing.T) {
	brand := BrandSnapshot{Name: "Koi", Domain: "www.koi.dev", GitHubRepo: "https://github.com/koihq/koi"}
	for _, test := range []struct {
		name    string
		answer  string
		sources []string
		want    Outcome
	}{
		{name: "name in answer", answer: "Try KOI for tracking.", want: OutcomeMentioned},
		{name: "substring", answer: "koifish and akoi", want: OutcomeMiss},
		{name: "Unicode boundary", answer: "éKoi魚", want: OutcomeMiss},
		{name: "domain in answer only", answer: "See koi.dev", want: OutcomeMentioned},
		{name: "citation wins", answer: "Koi", sources: []string{"https://docs.KOI.dev/start"}, want: OutcomeCited},
		{name: "source only", sources: []string{"https://www.koi.dev"}, want: OutcomeCited},
		{name: "lookalike", sources: []string{"https://koi.dev.evil.test", "https://notkoi.dev"}, want: OutcomeMiss},
		{name: "userinfo", sources: []string{"https://koi.dev@evil.test/"}, want: OutcomeMiss},
		{name: "unsafe scheme", sources: []string{"javascript://koi.dev/x"}, want: OutcomeMiss},
		{name: "own repository", sources: []string{"https://github.com/koihq/koi/issues/1"}, want: OutcomeCited},
		{name: "other repository", sources: []string{"https://github.com/koihq/koi-other", "https://github.com/other/koi"}, want: OutcomeMiss},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, ScoreVisibility(brand, test.answer, test.sources))
		})
	}

	require.Equal(t, OutcomeMentioned, ScoreVisibility(BrandSnapshot{Name: "Café   Studio"}, "Try CAFÉ\nStudio.", nil))
	require.Equal(t, OutcomeMiss, ScoreVisibility(BrandSnapshot{}, "", nil))
}

func TestContainsWord(t *testing.T) {
	for _, test := range []struct {
		name string
		text string
		find string
		want bool
	}{
		{name: "start", text: "Koi tracks answers", find: "Koi", want: true},
		{name: "end", text: "answers tracked by Koi", find: "Koi", want: true},
		{name: "whole text", text: "Koi", find: "Koi", want: true},
		{name: "later occurrence after rejected prefix", text: "koikoi then koi", find: "koi", want: true},
		{name: "only inside words", text: "koikoi akoi koi_x", find: "koi", want: false},
		{name: "punctuation edges", text: "Try Peec AI. It works", find: "Peec AI", want: true},
		{name: "Unicode folding", text: "Visit STRASSE today", find: "Straße", want: true},
		{name: "non-word name edge", text: "Prefer C++17 here", find: "C++", want: true},
		{name: "collapsed whitespace", text: "Peec\n\t AI tracks", find: "Peec  AI", want: true},
		{name: "longer than text", text: "Koi", find: "Koi Analytics", want: false},
		{name: "empty name", text: "Koi", find: " ", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, containsWord(foldText(test.text), foldText(test.find)))
		})
	}
}

func TestMentionVerdictMatchesOutcome(t *testing.T) {
	now := time.Now()
	base := VisibilityResult{
		BaseEntity: BaseEntity{
			ID: uuid.New(),
		},
		BrandID:      uuid.New(),
		RunCreatedAt: now,
		RunID:        uuid.New(),
		PromptID:     uuid.New(),
		PromptText:   "What is Koi?",
		Engine:       EngineChatGPT,
		Collector:    CollectorCloro,
		FetchedAt:    &now,
		Sources:      []string{},
	}

	for _, test := range []struct {
		verdict MentionVerdict
		outcome Outcome
		valid   bool
	}{
		{verdict: MentionVerdictMention, outcome: OutcomeMentioned, valid: true},
		{verdict: MentionVerdictNoInfo, outcome: OutcomeMiss, valid: true},
		{verdict: MentionVerdictOtherCompany, outcome: OutcomeMiss, valid: true},
		{verdict: MentionVerdictOtherMeaning, outcome: OutcomeMiss, valid: true},
		{verdict: MentionVerdictMention, outcome: OutcomeMiss},
		{verdict: MentionVerdictNoInfo, outcome: OutcomeMentioned},
		{verdict: MentionVerdictNoInfo, outcome: OutcomeCited},
		{verdict: "maybe", outcome: OutcomeMiss},
	} {
		t.Run(string(test.verdict)+"/"+string(test.outcome), func(t *testing.T) {
			result := base
			result.Outcome = &test.outcome
			result.MentionVerdict = &test.verdict

			err := result.Validate()
			if test.valid {
				require.NoError(t, err)
				require.Equal(t, test.outcome, test.verdict.Outcome())

				return
			}

			require.ErrorIs(t, err, ErrValidation)
		})
	}
}
