package koipricing

import (
	"errors"
	"testing"
)

func TestCurrentCatalogQuotes(t *testing.T) {
	catalog := Current()
	if catalog.Version != "cloud-eur-monthly-v1" || catalog.Currency != "eur" {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}

	for _, sample := range []struct{ prompts, cents int }{
		{5, 900}, {6, 1000}, {7, 1100}, {8, 1300}, {25, 2900},
		{501, 32000}, {750, 44900}, {1000, 54900},
	} {
		got, err := catalog.Quote(sample.prompts)
		if err != nil || got != int64(sample.cents) {
			t.Fatalf("quote(%d) = %d, %v; want %d", sample.prompts, got, err, sample.cents)
		}
	}
	for _, prompts := range []int{4, 1001} {
		if _, err := catalog.Quote(prompts); !errors.Is(err, ErrPromptRange) {
			t.Fatalf("quote(%d): %v", prompts, err)
		}
	}

	for prompts := 5; prompts <= 1000; prompts++ {
		if amount, err := catalog.Quote(prompts); err != nil || amount < 900 || amount > 54900 {
			t.Fatalf("quote(%d) = %d, %v", prompts, amount, err)
		}
	}
}
