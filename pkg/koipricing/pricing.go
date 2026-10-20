package koipricing

import (
	_ "embed"
	"encoding/json"
	"errors"
)

//go:embed catalog.json
var catalogJSON []byte

type Point struct {
	Prompts int `json:"prompts"`
	Euros   int `json:"euros"`
}

type Catalog struct {
	Currency string  `json:"currency"`
	Options  []Point `json:"options"`
}

var ErrPromptRange = errors.New("unsupported prompt allowance")

func Current() Catalog {
	var catalog Catalog
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil {
		panic(err)
	}
	return catalog
}

// Quote returns the monthly amount for a selectable prompt allowance.
func (c Catalog) Quote(prompts int) (int64, error) {
	for _, option := range c.Options {
		if prompts == option.Prompts {
			return int64(option.Euros) * 100, nil
		}
	}
	return 0, ErrPromptRange
}
