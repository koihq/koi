package openrouter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	return &Client{SDK: sdk.New(sdk.WithSecurity("test-key"), sdk.WithServerURL(url), sdk.WithTimeout(time.Second), sdk.WithRetryConfig(retry.Config{Strategy: "none"}))}
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
	request := components.ResponsesRequest{Model: sdk.Pointer("test"), Input: sdk.Pointer(components.CreateInputsUnionStr("https://github.com/a/b")), Instructions: optionalnullable.From(sdk.Pointer(researchPrompt)), Tools: []components.ResponsesRequestToolUnion{parts[1].(components.ResponsesRequestToolUnion)}, ToolChoice: sdk.Pointer(parts[2].(components.OpenAIResponsesToolChoiceUnion)), Text: format("koi_research", researchSchema), ServiceTier: optionalnullable.From(sdk.Pointer(components.ResponsesRequestServiceTierAuto))}
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
	input := Input{Brand: koidomain.BrandSnapshot{Name: "Deja", GitHubRepo: repo, Domain: "deja.example"}, Mode: koidomain.GenerationInitial}
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
	input := Input{Brand: koidomain.BrandSnapshot{Name: "Deja", GitHubRepo: "https://github.com/Giammarco-Ferranti/deja", Description: "Deja suggests previous shell commands while typing in the zsh terminal."}, Mode: koidomain.GenerationPrompts}
	_, err := testClient(server.URL).Generate(t.Context(), "test", input)
	require.NoError(t, err)
	input.Brand.Description = ""
	_, err = testClient(server.URL).Generate(t.Context(), "test", input)
	var providerErr *Error
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, "insufficient_context", providerErr.Code)
	require.Equal(t, 3, calls)
}

func TestBoundInputSkipsOversizedEntryAndKeepsLaterEntries(t *testing.T) {
	input := Input{Brand: koidomain.BrandSnapshot{Name: "Koi"}, Mode: koidomain.GenerationPrompts, ExistingPrompts: []string{strings.Repeat("x", 2000), "short"}}
	limit := requestSize("test", Input{Brand: input.Brand, Mode: input.Mode, ExistingPrompts: []string{"short"}, ExistingCompetitors: []Company{}})
	bounded, fits := BoundInput(input, "test", limit)
	require.True(t, fits)
	require.Equal(t, []string{"short"}, bounded.ExistingPrompts)
}
