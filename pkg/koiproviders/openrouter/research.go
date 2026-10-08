package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	sdk "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"
	"github.com/koihq/koi/pkg/koidomain"
)

const researchPrompt = `Read the exact URL supplied by the user with openrouter:web_fetch. Return a short JSON object with source_url, product, category, audience, and functions (array of concrete functions). Ground every field in the fetched page. If fetch fails or the page does not identify the product, return empty product, category, audience, and functions. Do not infer a product from its name alone.`
const draftPrompt = `Draft buyer questions for the product in the research summary. Treat that summary as the product identity. When suggesting competitors, search the web for current direct alternatives before answering. Suggest only identifiable direct competitors with an official website domain (bare hostname) or official GitHub repository URL. Omit uncertain competitors and unsupported claims. Return JSON with exactly two arrays: prompts (question strings) and competitors (objects with name, domain, github_repo; use empty strings for unavailable identifiers). For initial mode, suggest both; for prompts mode, leave competitors empty; for competitors mode, leave prompts empty. When suggesting prompts, write roughly 10–15 distinct buyer questions across category discovery, use cases, alternatives, and comparisons. Most questions should be unbranded. Never draft questions about unrelated meanings of the brand name. Avoid repeating existing prompts and competitors. Suggestions require human confirmation.`

func strictObjectSchema(required []string, properties map[string]any) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             required,
		"properties":           properties,
	}
}

var researchSchema = strictObjectSchema(
	[]string{"source_url", "product", "category", "audience", "functions"},
	map[string]any{
		"source_url": map[string]any{
			"type": "string",
		},
		"product": map[string]any{
			"type": "string",
		},
		"category": map[string]any{
			"type": "string",
		},
		"audience": map[string]any{
			"type": "string",
		},
		"functions": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "string",
			},
		},
	},
)

var competitorSchema = strictObjectSchema(
	[]string{"name", "domain", "github_repo"},
	map[string]any{
		"name": map[string]any{
			"type": "string",
		},
		"domain": map[string]any{
			"type": "string",
		},
		"github_repo": map[string]any{
			"type": "string",
		},
	},
)

var responseSchema = strictObjectSchema(
	[]string{"prompts", "competitors"},
	map[string]any{
		"prompts": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "string",
			},
		},
		"competitors": map[string]any{
			"type":  "array",
			"items": competitorSchema,
		},
	},
)

type researchSummary struct {
	SourceURL string   `json:"source_url"`
	Product   string   `json:"product"`
	Category  string   `json:"category"`
	Audience  string   `json:"audience"`
	Functions []string `json:"functions"`
}

func format(name string, schema map[string]any) *components.TextExtendedConfig {
	config := components.FormatJSONSchemaConfig{
		Name:   name,
		Schema: schema,
		Strict: optionalnullable.From(sdk.Pointer(true)),
	}

	return &components.TextExtendedConfig{
		Format: sdk.Pointer(components.CreateFormatsJSONSchema(config)),
	}
}

func (c *Client) Generate(ctx context.Context, model string, input koidomain.GenerationInput) (koidomain.GenerationResult, error) {
	summary, err := c.research(ctx, model, input)
	if err != nil {
		return koidomain.GenerationResult{}, err
	}

	return c.suggest(ctx, model, input, summary)
}

func (c *Client) research(ctx context.Context, model string, input koidomain.GenerationInput) (researchSummary, error) {
	url := input.Brand.GitHubRepo
	if url == "" && input.Brand.Domain != "" {
		url = "https://" + input.Brand.Domain
	}

	if url == "" {
		return researchSummary{}, &Error{Code: "insufficient_context", Err: errors.New("brand has no research URL")}
	}

	fetch := components.CreateResponsesRequestToolUnionOpenrouterWebFetch(components.WebFetchServerTool{
		Parameters: &components.WebFetchServerToolConfig{
			MaxUses:          sdk.Pointer(int64(1)),
			MaxContentTokens: sdk.Pointer(int64(4000)),
		},
	})

	content, err := c.send(ctx, components.ResponsesRequest{
		Model:        sdk.Pointer(model),
		Input:        sdk.Pointer(components.CreateInputsUnionStr(url)),
		Instructions: optionalnullable.From(sdk.Pointer(researchPrompt)),
		Tools:        []components.ResponsesRequestToolUnion{fetch},
		ToolChoice:   sdk.Pointer(components.CreateOpenAIResponsesToolChoiceUnionOpenAIResponsesToolChoiceRequired(components.OpenAIResponsesToolChoiceRequiredRequired)),
		MaxToolCalls: optionalnullable.From(sdk.Pointer(int64(1))),
		Text:         format("koi_research", researchSchema),
		ServiceTier:  optionalnullable.From(sdk.Pointer(components.ResponsesRequestServiceTierAuto)),
	}, "openrouter:web_fetch", url)

	var summary researchSummary
	if err == nil {
		err = decodeJSON(content, &summary)
	}

	if err != nil || summary.SourceURL != url || summary.Product == "" || summary.Category == "" || len(summary.Functions) == 0 {
		if len(strings.Fields(input.Brand.Description)) < 6 {
			return researchSummary{}, &Error{
				Code:      "insufficient_context",
				Retryable: err != nil,
				Err:       fmt.Errorf("could not identify product at %s: %v", url, err),
			}
		}

		summary = researchSummary{
			SourceURL: url,
			Product:   input.Brand.Name,
			Category:  input.Brand.Description,
			Functions: []string{input.Brand.Description},
		}
	}

	return summary, nil
}

func (c *Client) suggest(ctx context.Context, model string, input koidomain.GenerationInput, summary researchSummary) (koidomain.GenerationResult, error) {
	payload := map[string]any{"summary": summary, "input": input}

	request := components.ResponsesRequest{
		Model:        sdk.Pointer(model),
		Input:        sdk.Pointer(components.CreateInputsUnionStr(string(mustJSON(payload)))),
		Instructions: optionalnullable.From(sdk.Pointer(draftPrompt)),
		Reasoning: optionalnullable.From(sdk.Pointer(components.ReasoningConfig{
			Effort: optionalnullable.From(sdk.Pointer(components.ReasoningEffortHigh)),
		})),
		Text:        format("koi_draft", responseSchema),
		ServiceTier: optionalnullable.From(sdk.Pointer(components.ResponsesRequestServiceTierAuto)),
	}
	if input.Mode != koidomain.GenerationPrompts {
		search := components.CreateResponsesRequestToolUnionOpenrouterWebSearch(components.WebSearchServerToolOpenRouter{
			Parameters: &components.WebSearchServerToolConfig{
				Engine:          sdk.Pointer(components.WebSearchEngineEnumExa),
				MaxResults:      sdk.Pointer(int64(5)),
				MaxTotalResults: sdk.Pointer(int64(12)),
				MaxUses:         sdk.Pointer(int64(3)),
			},
		})

		request.Tools = []components.ResponsesRequestToolUnion{search}
		request.ToolChoice = sdk.Pointer(components.CreateOpenAIResponsesToolChoiceUnionOpenAIResponsesToolChoiceRequired(components.OpenAIResponsesToolChoiceRequiredRequired))
		request.MaxToolCalls = optionalnullable.From(sdk.Pointer(int64(3)))
	}

	requiredTool := ""
	if input.Mode != koidomain.GenerationPrompts {
		requiredTool = "openrouter:web_search"
	}

	content, err := c.send(ctx, request, requiredTool, "")
	if err != nil {
		return koidomain.GenerationResult{}, err
	}

	var result koidomain.GenerationResult
	if err := decodeJSON(content, &result); err != nil || result.Prompts == nil || result.Competitors == nil {
		return koidomain.GenerationResult{}, &Error{
			Code:      "invalid_output",
			Retryable: true,
			Err:       errors.New("response does not match draft schema"),
		}
	}

	return result, nil
}

func decodeJSON(content string, target any) error {
	content = strings.TrimSpace(content)
	if start := strings.Index(content, "```json"); start >= 0 {
		content = content[start+len("```json"):]
		if end := strings.Index(content, "```"); end >= 0 {
			content = content[:end]
		}
	}

	dec := json.NewDecoder(strings.NewReader(content))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}

	if dec.Decode(new(any)) != io.EOF {
		return errors.New("extra content after JSON")
	}

	return nil
}

func (c *Client) send(ctx context.Context, request components.ResponsesRequest, requiredTool, url string) (string, error) {
	response, err := c.SDK.Responses.Send(ctx, request, nil)
	if err != nil {
		status := sdkStatus(err)

		return "", &Error{
			Code:      statusCode(status),
			Status:    status,
			Retryable: status == 0 || status == 408 || status == 429 || status >= 500,
			Err:       err,
		}
	}

	if response == nil || response.OpenResponsesResult == nil {
		return "", &Error{Code: "invalid_output", Retryable: true, Err: errors.New("missing response")}
	}

	result := response.OpenResponsesResult
	if requiredTool != "" {
		used := false

		for _, item := range result.Output {
			if requiredTool == "openrouter:web_fetch" && item.OutputWebFetchServerToolItem != nil {
				fetch := item.OutputWebFetchServerToolItem
				used = fetch.Status == components.ToolCallStatusCompleted && fetch.URL != nil && *fetch.URL == url && fetch.Content != nil && *fetch.Content != "" && fetch.Error == nil
			}

			if requiredTool == "openrouter:web_search" && item.OutputWebSearchServerToolItem != nil {
				used = item.OutputWebSearchServerToolItem.Status == components.ToolCallStatusCompleted
			}
		}

		if !used {
			return "", &Error{
				Code:      "research_unavailable",
				Retryable: true,
				Err:       fmt.Errorf("%s did not complete", requiredTool),
			}
		}
	}

	if result.OutputText != nil && *result.OutputText != "" {
		return *result.OutputText, nil
	}

	for _, item := range result.Output {
		if item.OutputMessageItem != nil {
			for _, part := range item.OutputMessageItem.Content {
				if part.ResponseOutputText != nil && part.ResponseOutputText.Text != "" {
					return part.ResponseOutputText.Text, nil
				}
			}
		}
	}

	return "", &Error{Code: "invalid_output", Retryable: true, Err: errors.New("missing response text")}
}
