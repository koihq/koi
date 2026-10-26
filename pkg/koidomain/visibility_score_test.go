package koidomain

import (
	"testing"

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
