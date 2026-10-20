package koipricing

import (
	"errors"
	"testing"
)

func TestCurrentCatalogQuotes(t *testing.T) {
	catalog := Current()
	if catalog.Currency != "eur" || len(catalog.Options) != 12 {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}

	for _, sample := range []struct{ prompts, cents int }{
		{5, 900}, {10, 1500}, {15, 2000}, {25, 2900},
		{50, 4900}, {75, 6900}, {100, 8900}, {150, 11900},
		{200, 14900}, {250, 17900}, {350, 23500}, {500, 31900},
	} {
		got, err := catalog.Quote(sample.prompts)
		if err != nil || got != int64(sample.cents) {
			t.Fatalf("quote(%d) = %d, %v; want %d", sample.prompts, got, err, sample.cents)
		}
	}
	for _, prompts := range []int{4, 6, 26, 351, 501, 750, 1000} {
		if _, err := catalog.Quote(prompts); !errors.Is(err, ErrPromptRange) {
			t.Fatalf("quote(%d): %v", prompts, err)
		}
	}
}
