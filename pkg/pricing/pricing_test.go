package pricing

import (
	"errors"
	"testing"
)

func TestCurrentCatalogQuotes(t *testing.T) {
	catalog := Current()
	if catalog.Currency != "eur" || len(catalog.Options) != 10 {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}

	for _, sample := range []struct{ prompts, cents int }{
		{15, 2900}, {25, 4500}, {50, 7500}, {75, 9900}, {100, 12900},
		{150, 17900}, {200, 22900}, {250, 26900}, {350, 34900}, {500, 44900},
	} {
		got, err := catalog.Quote(sample.prompts)
		if err != nil || got != int64(sample.cents) {
			t.Fatalf("quote(%d) = %d, %v; want %d", sample.prompts, got, err, sample.cents)
		}
	}
	for _, prompts := range []int{4, 5, 6, 10, 26, 351, 501, 750, 1000} {
		if _, err := catalog.Quote(prompts); !errors.Is(err, ErrPromptRange) {
			t.Fatalf("quote(%d): %v", prompts, err)
		}
	}
}
