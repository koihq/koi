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
	Version  string  `json:"version"`
	Currency string  `json:"currency"`
	Anchors  []Point `json:"anchors"`
}

var ErrPromptRange = errors.New("prompt allowance must be between 5 and 1000")

func Current() Catalog {
	var catalog Catalog
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil {
		panic(err)
	}
	return catalog
}

// Quote interpolates in integer euros, rounding a half euro upward.
func (c Catalog) Quote(prompts int) (int64, error) {
	if len(c.Anchors) < 2 || prompts < c.Anchors[0].Prompts || prompts > c.Anchors[len(c.Anchors)-1].Prompts {
		return 0, ErrPromptRange
	}

	for i := 1; i < len(c.Anchors); i++ {
		low, high := c.Anchors[i-1], c.Anchors[i]
		if prompts > high.Prompts {
			continue
		}

		width := int64(high.Prompts - low.Prompts)
		rise := int64(high.Euros - low.Euros)
		offset := int64(prompts - low.Prompts)
		euros := int64(low.Euros) + (2*rise*offset+width)/(2*width)

		return euros * 100, nil
	}

	panic("unreachable price interval")
}
