package stub

import (
	"context"
	"fmt"

	"github.com/koihq/koi/pkg/domain"
)

// Collect produces sample data only; live visibility collection will use Cloro.
func Collect(ctx context.Context, brand domain.BrandSnapshot, result domain.VisibilityResult) (domain.VisibilityResult, error) {
	if err := ctx.Err(); err != nil {
		return result, err
	}

	enginePosition := 0

	for i, engine := range domain.VisibilityEngines() {
		if engine == result.Engine {
			enginePosition = i
			break
		}
	}

	outcomes := []domain.Outcome{domain.OutcomeCited, domain.OutcomeMentioned, domain.OutcomeMiss, domain.OutcomeError}
	outcome := outcomes[(result.PromptPosition+enginePosition)%len(outcomes)]
	result.Outcome = &outcome
	result.Sources = []string{}
	result.ErrorCode = nil

	switch outcome {
	case domain.OutcomeCited:
		result.RawAnswer = fmt.Sprintf("Sample answer for %q: %s is an option. No engine was contacted.", result.PromptText, brand.Name)
		url := brand.GitHubRepo
		if brand.Domain != "" {
			url = "https://" + brand.Domain
		}
		result.Sources = []string{url}
	case domain.OutcomeMentioned:
		result.RawAnswer = fmt.Sprintf("Sample answer for %q: consider %s. No engine was contacted.", result.PromptText, brand.Name)
	case domain.OutcomeMiss:
		result.RawAnswer = fmt.Sprintf("Sample answer for %q: other products are recommended. No engine was contacted.", result.PromptText)
	case domain.OutcomeError:
		result.RawAnswer = ""
		code := "stub_error"
		result.ErrorCode = &code
	}

	return result, nil
}
