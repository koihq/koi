package openrouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sdk "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/sdkerrors"
	"github.com/OpenRouterTeam/go-sdk/retry"
	"github.com/koihq/koi/pkg/koidomain"
)

type Error struct {
	Code      string
	Retryable bool
	Status    int
	Err       error
}

func (e *Error) Error() string { return fmt.Sprintf("openrouter %s: %v", e.Code, e.Err) }

func (e *Error) Unwrap() error { return e.Err }

type Client struct{ SDK *sdk.OpenRouter }

func New(key string, timeout time.Duration) *Client {
	return &Client{
		SDK: sdk.New(sdk.WithSecurity(key), sdk.WithTimeout(timeout), sdk.WithRetryConfig(retry.Config{
			Strategy: "none",
		})),
	}
}

// Context is bounded by serialized request bytes, not an arbitrary row count.
func BoundInput(input koidomain.GenerationInput, model string, maxBytes int) (koidomain.GenerationInput, bool) {
	bounded := koidomain.GenerationInput{
		Brand:               input.Brand,
		Mode:                input.Mode,
		PromptTarget:        input.PromptTarget,
		ExistingPrompts:     []string{},
		ExistingCompetitors: []koidomain.CompetitorIdentity{},
	}
	if requestSize(model, bounded) > maxBytes {
		return bounded, false
	}

	for _, prompt := range input.ExistingPrompts {
		bounded.ExistingPrompts = append(bounded.ExistingPrompts, prompt)
		if requestSize(model, bounded) > maxBytes {
			bounded.ExistingPrompts = bounded.ExistingPrompts[:len(bounded.ExistingPrompts)-1]
		}
	}

	for _, company := range input.ExistingCompetitors {
		bounded.ExistingCompetitors = append(bounded.ExistingCompetitors, company)
		if requestSize(model, bounded) > maxBytes {
			bounded.ExistingCompetitors = bounded.ExistingCompetitors[:len(bounded.ExistingCompetitors)-1]
		}
	}

	return bounded, true
}

func mustJSON(value any) []byte { b, _ := json.Marshal(value); return b }

func requestSize(model string, input koidomain.GenerationInput) int {
	return len(mustJSON(map[string]any{
		"model":  model,
		"input":  input,
		"prompt": draftInstructions(input.PromptTarget),
		"schema": responseSchema,
	}))
}

func sdkStatus(err error) int {
	var apiErr *sdkerrors.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}

	var badRequest *sdkerrors.BadRequestResponseError
	if errors.As(err, &badRequest) {
		return 400
	}

	var unauthorized *sdkerrors.UnauthorizedResponseError
	if errors.As(err, &unauthorized) {
		return 401
	}

	var payment *sdkerrors.PaymentRequiredResponseError
	if errors.As(err, &payment) {
		return 402
	}

	var forbidden *sdkerrors.ForbiddenResponseError
	if errors.As(err, &forbidden) {
		return 403
	}

	var notFound *sdkerrors.NotFoundResponseError
	if errors.As(err, &notFound) {
		return 404
	}

	var timeout *sdkerrors.RequestTimeoutResponseError
	if errors.As(err, &timeout) {
		return 408
	}

	var tooLarge *sdkerrors.PayloadTooLargeResponseError
	if errors.As(err, &tooLarge) {
		return 413
	}

	var invalid *sdkerrors.UnprocessableEntityResponseError
	if errors.As(err, &invalid) {
		return 422
	}

	var rateLimit *sdkerrors.TooManyRequestsResponseError
	if errors.As(err, &rateLimit) {
		return 429
	}

	var server *sdkerrors.InternalServerResponseError
	if errors.As(err, &server) {
		return 500
	}

	var gateway *sdkerrors.BadGatewayResponseError
	if errors.As(err, &gateway) {
		return 502
	}

	var unavailable *sdkerrors.ServiceUnavailableResponseError
	if errors.As(err, &unavailable) {
		return 503
	}

	var networkTimeout *sdkerrors.EdgeNetworkTimeoutResponseError
	if errors.As(err, &networkTimeout) {
		return 524
	}

	var overloaded *sdkerrors.ProviderOverloadedResponseError
	if errors.As(err, &overloaded) {
		return 529
	}

	return 0
}

func statusCode(status int) string {
	if status == 0 {
		return "provider_unavailable"
	}

	if status == 401 || status == 403 {
		return "provider_auth"
	}

	if status == 429 {
		return "provider_rate_limited"
	}

	if status >= 500 || status == 408 {
		return "provider_unavailable"
	}

	return "provider_rejected"
}
