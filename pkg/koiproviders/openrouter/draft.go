package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	sdk "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/models/sdkerrors"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"
	"github.com/OpenRouterTeam/go-sdk/retry"
	"github.com/koihq/koi/pkg/koidomain"
)

const draftPromptV1 = `Draft buyer questions and competing companies for the supplied brand. Use the brand snapshot as the source of context. Avoid repeating any existing prompt or competitor. Return JSON with exactly two arrays: prompts (question strings) and competitors (objects with name and domain). For initial mode, suggest both; for prompts mode, leave competitors empty; for competitors mode, leave prompts empty. Suggest useful, distinct entries only. A domain must be a bare hostname, not a URL. Do not assert that a logo or company identity has been verified.`

type Company = koidomain.DraftCompany
type Draft = koidomain.DraftOutput
type Input = koidomain.DraftInput

type Error struct {
	Code      string
	Retryable bool
	Status    int
	Err       error
}

func (e *Error) Error() string { return fmt.Sprintf("openrouter %s: %v", e.Code, e.Err) }
func (e *Error) Unwrap() error { return e.Err }

type Client struct {
	SDK *sdk.OpenRouter
}

func New(key string, timeout time.Duration) *Client {
	return &Client{SDK: sdk.New(
		sdk.WithSecurity(key), sdk.WithTimeout(timeout),
		sdk.WithRetryConfig(retry.Config{Strategy: "none"}),
	)}
}

// Context is bounded by serialized request bytes, not an arbitrary row count.
func BoundInput(input Input, model string, maxBytes int) (Input, bool) {
	bounded := Input{Brand: input.Brand, Mode: input.Mode, ExistingPrompts: []string{}, ExistingCompetitors: []Company{}}
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

var responseSchema = map[string]any{
	"type": "object", "additionalProperties": false,
	"required": []string{"prompts", "competitors"},
	"properties": map[string]any{
		"prompts": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"competitors": map[string]any{"type": "array", "items": map[string]any{
			"type": "object", "additionalProperties": false,
			"required":   []string{"name", "domain"},
			"properties": map[string]any{"name": map[string]any{"type": "string"}, "domain": map[string]any{"type": "string"}},
		}},
	},
}

func requestSize(model string, input Input) int { return len(mustJSON(chatRequest(model, input))) }

func chatRequest(model string, input Input) components.ChatRequest {
	return components.ChatRequest{
		Model: sdk.Pointer(model),
		Messages: []components.ChatMessages{
			components.CreateChatMessagesSystem(components.ChatSystemMessage{Content: components.CreateChatSystemMessageContentStr(draftPromptV1)}),
			components.CreateChatMessagesUser(components.ChatUserMessage{Content: components.CreateChatUserMessageContentStr(string(mustJSON(input)))}),
		},
		ResponseFormat: sdk.Pointer(components.CreateResponseFormatJSONSchema(components.ChatFormatJSONSchemaConfig{
			JSONSchema: components.ChatJSONSchemaConfig{Name: "koi_draft_v1", Strict: optionalnullable.From(sdk.Pointer(true)), Schema: responseSchema},
		})),
	}
}

func (c *Client) Generate(ctx context.Context, model string, input Input) (Draft, error) {
	resp, err := c.SDK.Chat.Send(ctx, chatRequest(model, input), nil)
	if err != nil {
		status := sdkStatus(err)
		return Draft{}, &Error{Code: statusCode(status), Retryable: status == 0 || status == 408 || status == 429 || status >= 500, Status: status, Err: err}
	}
	if resp == nil || resp.ChatResult == nil || len(resp.ChatResult.Choices) == 0 {
		return Draft{}, &Error{Code: "invalid_output", Retryable: true, Err: errors.New("missing or malformed response")}
	}
	message := resp.ChatResult.Choices[0].Message
	content, _ := message.Content.Get()
	refusal, _ := message.Refusal.Get()
	if content == nil || content.Str == nil || refusal != nil && *refusal != "" {
		return Draft{}, &Error{Code: "invalid_output", Retryable: true, Err: errors.New("missing or refused response content")}
	}
	var draft Draft
	dec := json.NewDecoder(strings.NewReader(*content.Str))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&draft); err != nil || draft.Prompts == nil || draft.Competitors == nil || dec.Decode(new(any)) != io.EOF {
		return Draft{}, &Error{Code: "invalid_output", Retryable: true, Err: errors.New("response does not match draft schema")}
	}
	return draft, nil
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
