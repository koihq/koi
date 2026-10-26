package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	url2 "net/url"
	"strings"
	"testing"
	"time"

	sdk "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"
	"github.com/OpenRouterTeam/go-sdk/retry"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/stretchr/testify/require"
)

func testClient(url string) *Client {
	serverURL, _ := url2.Parse(url)
	httpClient := &http.Client{
		Timeout: time.Second,
		Transport: localOnlyTransport{
			host: serverURL.Host,
			base: http.DefaultTransport,
		},
	}

	return &Client{
		SDK: sdk.New(sdk.WithSecurity("test-key"), sdk.WithServerURL(url), sdk.WithClient(httpClient), sdk.WithRetryConfig(retry.Config{
			Strategy: "none",
		})),
	}
}

type localOnlyTransport struct {
	host string
	base http.RoundTripper
}

func (t localOnlyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != t.host {
		return nil, errors.New("test client rejected non-local provider destination")
	}

	return t.base.RoundTrip(r)
}

func TestMarshalRequestParts(t *testing.T) {
	parts := []any{
		components.CreateInputsUnionStr("https://github.com/a/b"),
		components.CreateResponsesRequestToolUnionOpenrouterWebFetch(components.WebFetchServerTool{}),
		components.CreateOpenAIResponsesToolChoiceUnionOpenAIResponsesToolChoiceRequired(components.OpenAIResponsesToolChoiceRequiredRequired),
		optionalnullable.From(sdk.Pointer(researchPrompt)),
		format("koi_research", researchSchema),
	}

	for i, part := range parts {
		_, err := json.Marshal(part)
		require.NoErrorf(t, err, "part %d", i)
	}

	request := components.ResponsesRequest{
		Model:        sdk.Pointer("test"),
		Input:        sdk.Pointer(components.CreateInputsUnionStr("https://github.com/a/b")),
		Instructions: optionalnullable.From(sdk.Pointer(researchPrompt)),
		Tools:        []components.ResponsesRequestToolUnion{parts[1].(components.ResponsesRequestToolUnion)},
		ToolChoice:   sdk.Pointer(parts[2].(components.OpenAIResponsesToolChoiceUnion)),
		Text:         format("koi_research", researchSchema),
		ServiceTier:  optionalnullable.From(sdk.Pointer(components.ResponsesRequestServiceTierAuto)),
	}
	_, err := json.Marshal(request)
	require.NoError(t, err)
}

func result(content, tool, url string) string {
	quoted, _ := json.Marshal(content)
	output := `[]`
	if tool == "openrouter:web_fetch" {
		output = fmt.Sprintf(`[{"type":"openrouter:web_fetch","status":"completed","url":%q,"content":"repository content"}]`, url)
	}

	if tool == "openrouter:web_search" {
		output = `[{"type":"openrouter:web_search","status":"completed"}]`
	}

	return fmt.Sprintf(`{"id":"test","object":"response","created_at":1,"model":"test","status":"completed","output":%s,"output_text":%s,"tool_choice":"auto","tools":[],"parallel_tool_calls":false}`, output, quoted)
}

func TestGenerateResearchesDejaAndSearchesAlternatives(t *testing.T) {
	const repo = "https://github.com/Giammarco-Ferranti/deja"
	input := koidomain.GenerationInput{
		Brand: koidomain.BrandSnapshot{
			Name:       "Deja",
			GitHubRepo: repo,
			Domain:     "deja.example",
		},
		Mode: koidomain.GenerationInitial,
	}
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		require.Equal(t, "/responses", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "required", body["tool_choice"])
		require.Equal(t, "json_schema", body["text"].(map[string]any)["format"].(map[string]any)["type"])
		if calls == 1 {
			require.Equal(t, repo, body["input"])
			require.Equal(t, "openrouter:web_fetch", body["tools"].([]any)[0].(map[string]any)["type"])
			_, _ = fmt.Fprint(w, result("```json\n"+`{"source_url":"https://github.com/Giammarco-Ferranti/deja","product":"Deja","category":"zsh autosuggestion tool","audience":"terminal users","functions":["suggest shell commands from history"]}`+"\n```", "openrouter:web_fetch", repo))

			return
		}

		require.Contains(t, body["input"], "zsh autosuggestion tool")
		require.Equal(t, "high", body["reasoning"].(map[string]any)["effort"])
		require.Equal(t, "openrouter:web_search", body["tools"].([]any)[0].(map[string]any)["type"])
		require.Equal(t, float64(3), body["tools"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["max_uses"])
		_, _ = fmt.Fprint(w, result("```json\n"+`{"prompts":["What are the best zsh command autosuggestion tools?","Which shell history suggestion plugins work with zsh?"],"competitors":[{"name":"zsh-autosuggestions","domain":"","github_repo":"https://github.com/zsh-users/zsh-autosuggestions"}]}`+"\n```", "openrouter:web_search", ""))
	}))

	defer server.Close()
	draft, err := testClient(server.URL).Generate(t.Context(), koidomain.GenerationModelSlug, input)
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Len(t, draft.Prompts, 2)
	require.NotContains(t, strings.ToLower(strings.Join(draft.Prompts, " ")), "personal memory")
	require.Equal(t, "https://github.com/zsh-users/zsh-autosuggestions", draft.Competitors[0].GitHubRepo)
}

func TestGenerateFetchFallback(t *testing.T) {
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls%2 == 1 {
			w.WriteHeader(503)
			_, _ = fmt.Fprint(w, `{"error":{"message":"unavailable"}}`)

			return
		}

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Contains(t, body["input"], "zsh terminal")
		_, _ = fmt.Fprint(w, result(`{"prompts":[],"competitors":[]}`, "", ""))
	}))

	defer server.Close()
	input := koidomain.GenerationInput{
		Brand: koidomain.BrandSnapshot{
			Name:        "Deja",
			GitHubRepo:  "https://github.com/Giammarco-Ferranti/deja",
			Description: "Deja suggests previous shell commands while typing in the zsh terminal.",
		},
		Mode: koidomain.GenerationPrompts,
	}
	_, err := testClient(server.URL).Generate(t.Context(), "test", input)
	require.NoError(t, err)
	input.Brand.Description = ""
	_, err = testClient(server.URL).Generate(t.Context(), "test", input)
	var providerErr *Error
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, "provider_unavailable", providerErr.Code)
	require.Equal(t, 503, providerErr.Status)
	require.True(t, providerErr.Retryable)
	require.Equal(t, 3, calls)
}

func TestResearchPreservesProviderFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		code   string
		retry  bool
	}{
		{"expired key", 401, "provider_auth", false},
		{"forbidden", 403, "provider_auth", false},
		{"payment", 402, "provider_rejected", false},
		{"invalid request", 400, "provider_rejected", false},
		{"rate limited", 429, "provider_rate_limited", true},
		{"unavailable", 503, "provider_unavailable", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprint(w, `{"error":{"message":"API key expired."}}`)
			}))
			defer server.Close()

			input := koidomain.GenerationInput{
				Brand: koidomain.BrandSnapshot{
					Name:       "Deja",
					GitHubRepo: "https://github.com/Giammarco-Ferranti/deja",
				},
				Mode: koidomain.GenerationInitial,
			}
			_, err := testClient(server.URL).Generate(t.Context(), "test", input)
			var providerErr *Error
			require.ErrorAs(t, err, &providerErr)
			require.Equal(t, tc.code, providerErr.Code)
			require.Equal(t, tc.status, providerErr.Status)
			require.Equal(t, tc.retry, providerErr.Retryable)
			require.Equal(t, 1, calls)

			if !tc.retry {
				input.Brand.Description = "Deja suggests previous shell commands while typing in the zsh terminal."
				_, err = testClient(server.URL).Generate(t.Context(), "test", input)
				require.ErrorAs(t, err, &providerErr)
				require.Equal(t, tc.code, providerErr.Code)
				require.Equal(t, tc.status, providerErr.Status)
				require.False(t, providerErr.Retryable)
				require.Equal(t, 2, calls, "terminal errors must stop before drafting even with fallback context")
			}
		})
	}
}

func TestBoundInputSkipsOversizedEntryAndKeepsLaterEntries(t *testing.T) {
	input := koidomain.GenerationInput{
		Brand: koidomain.BrandSnapshot{
			Name: "Koi",
		},
		Mode:            koidomain.GenerationPrompts,
		PromptTarget:    5,
		ExistingPrompts: []string{strings.Repeat("x", 2000), "short"},
	}
	limit := requestSize("test", koidomain.GenerationInput{
		Brand:               input.Brand,
		Mode:                input.Mode,
		PromptTarget:        input.PromptTarget,
		ExistingPrompts:     []string{"short"},
		ExistingCompetitors: []koidomain.CompetitorIdentity{},
	})
	bounded, fits := BoundInput(input, "test", limit)
	require.True(t, fits)
	require.Equal(t, []string{"short"}, bounded.ExistingPrompts)
	require.Equal(t, 5, bounded.PromptTarget)
}

func TestGenerationFixturesAndOutputValidation(t *testing.T) {
	for _, tc := range []struct {
		name, site, category, output, wantCode string
		mode                                   koidomain.GenerationMode
		wantPrompts                            int
	}{
		{"Plausible Analytics", "plausible.io", "privacy-friendly website analytics", `{"prompts":["Which privacy-friendly analytics tools work for websites?"],"competitors":[]}`, "", koidomain.GenerationPrompts, 1},
		{"sparse fictional brand", "acme.example", "", `{"prompts":[],"competitors":[]}`, "insufficient_context", koidomain.GenerationInitial, 0},
		{"empty arrays", "plausible.io", "website analytics", `{"prompts":[],"competitors":[]}`, "", koidomain.GenerationPrompts, 0},
		{"malformed JSON", "plausible.io", "website analytics", `{"prompts":[`, "invalid_output", koidomain.GenerationPrompts, 0},
		{"extra field", "plausible.io", "website analytics", `{"prompts":[],"competitors":[],"secret":"x"}`, "invalid_output", koidomain.GenerationPrompts, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "test-key", strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
				if calls == 1 {
					require.Equal(t, "https://"+tc.site, body["input"])
					require.Equal(t, "openrouter:web_fetch", body["tools"].([]any)[0].(map[string]any)["type"])
					_, _ = fmt.Fprint(w, result(fmt.Sprintf(`{"source_url":"https://%s","product":"%s","category":"%s","audience":"site owners","functions":["measure website traffic"]}`, tc.site, tc.name, tc.category), "openrouter:web_fetch", "https://"+tc.site))

					return
				}

				require.Contains(t, body["input"], tc.category)
				require.Contains(t, body["input"], `"mode":"prompts"`)
				_, _ = fmt.Fprint(w, result(tc.output, "", ""))
			}))

			defer server.Close()

			draft, err := testClient(server.URL).Generate(t.Context(), "test", koidomain.GenerationInput{
				Brand: koidomain.BrandSnapshot{
					Name:   tc.name,
					Domain: tc.site,
				},
				Mode: tc.mode,
			})
			if tc.wantCode != "" {
				var providerErr *Error
				require.ErrorAs(t, err, &providerErr)
				require.Equal(t, tc.wantCode, providerErr.Code)
			} else {
				require.NoError(t, err)
				require.Len(t, draft.Prompts, tc.wantPrompts)
			}

			if tc.wantCode == "insufficient_context" {
				require.Equal(t, 1, calls)
			} else {
				require.Equal(t, 2, calls)
			}
		})
	}
}

func TestProviderErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		code   string
		retry  bool
	}{
		{"timeout", 408, "provider_unavailable", true},
		{"rate limited", 429, "provider_rate_limited", true},
		{"server error", 503, "provider_unavailable", true},
		{"auth", 401, "provider_auth", false},
		{"payment", 402, "provider_rejected", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprint(w, `{"error":{"message":"provider detail must stay private"}}`)
			}))

			defer server.Close()

			_, err := testClient(server.URL).suggest(context.Background(), "test", koidomain.GenerationInput{
				Brand: koidomain.BrandSnapshot{
					Name:   "Plausible",
					Domain: "plausible.io",
				},
				Mode: koidomain.GenerationPrompts,
			}, researchSummary{Product: "Plausible", Category: "website analytics"})

			var providerErr *Error
			require.ErrorAs(t, err, &providerErr)
			require.Equal(t, tc.code, providerErr.Code)
			require.Equal(t, tc.retry, providerErr.Retryable)
		})
	}
}
