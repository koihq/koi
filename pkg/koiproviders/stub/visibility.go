package stub

import (
	"context"
	"fmt"

	"github.com/koihq/koi/pkg/koidomain"
)

// Collect produces sample data only; live visibility collection will use Cloro.
func Collect(ctx context.Context, brand koidomain.BrandSnapshot, result koidomain.VisibilityResult) (koidomain.VisibilityResult, error) {
	if err := ctx.Err(); err != nil {
		return result, err
	}

	enginePosition := 0

	for i, engine := range koidomain.VisibilityEngines() {
		if engine == result.Engine {
			enginePosition = i
			break
		}
	}

	outcomes := []koidomain.Outcome{koidomain.OutcomeCited, koidomain.OutcomeMentioned, koidomain.OutcomeMiss, koidomain.OutcomeError}
	outcome := outcomes[(result.PromptPosition+enginePosition)%len(outcomes)]
	result.Outcome = &outcome
	result.Sources = []string{}
	result.ErrorCode = nil

	switch outcome {
	case koidomain.OutcomeCited:
		result.RawAnswer = fmt.Sprintf("Sample answer for %q: %s is an option. No engine was contacted.", result.PromptText, brand.Name)
		url := brand.GitHubRepo
		if brand.Domain != "" {
			url = "https://" + brand.Domain
		}
		result.Sources = []string{url}
	case koidomain.OutcomeMentioned:
		result.RawAnswer = fmt.Sprintf("Sample answer for %q: consider %s. No engine was contacted.", result.PromptText, brand.Name)
	case koidomain.OutcomeMiss:
		result.RawAnswer = fmt.Sprintf("Sample answer for %q: other products are recommended. No engine was contacted.", result.PromptText)
	case koidomain.OutcomeError:
		result.RawAnswer = ""
		code := "stub_error"
		result.ErrorCode = &code
	}

	return result, nil
}
