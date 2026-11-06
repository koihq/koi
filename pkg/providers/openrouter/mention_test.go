package openrouter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/domain"
)

func mentionResponse(output, usage string) string {
	quoted, _ := json.Marshal(output)

	return fmt.Sprintf(`{"id":"test","object":"response","created_at":1,"model":"test","status":"completed","output":[],"output_text":%s,"tool_choice":"auto","tools":[],"parallel_tool_calls":false%s}`, quoted, usage)
}

const mentionUsage = `,"usage":{"input_tokens":800,"input_tokens_details":{"cached_tokens":0},"output_tokens":150,"output_tokens_details":{"reasoning_tokens":140},"total_tokens":950,"cost":0.0002}`

func mentionInput() domain.MentionInput {
	return domain.MentionInput{
		Brand: domain.BrandSnapshot{
			Name:        "Koi",
			GitHubRepo:  "https://github.com/koihq/koi",
			Country:     "US",
			Description: "An AEO tool.",
		},
		Prompt: "What is Koi?",
		Answer: "I couldn't reliably identify Koi.",
	}
}

func TestJudgeMentionRequestAndVerdict(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, mentionResponse(`{"verdict":"no_info"}`, mentionUsage))
	}))
	defer server.Close()

	judgment, err := testClient(server.URL).JudgeMention(t.Context(), mentionInput())
	require.NoError(t, err)

	require.Equal(t, domain.MentionVerdictNoInfo, judgment.Verdict)
	require.NotNil(t, judgment.CostUSD)
	require.InDelta(t, 0.0002, *judgment.CostUSD, 1e-9)

	require.Equal(t, domain.MentionModelSlug, body["model"])
	require.Equal(t, mentionPrompt, body["instructions"])
	require.Equal(t, "low", body["reasoning"].(map[string]any)["effort"])
	require.EqualValues(t, maxMentionOutputTokens, body["max_output_tokens"])
	require.Empty(t, body["tools"])

	input := body["input"].(string)
	require.Contains(t, input, `"github_repo":"https://github.com/koihq/koi"`)
	require.Contains(t, input, `"description":"An AEO tool."`)
	require.NotContains(t, input, "country")
	require.Contains(t, input, "PROMPT:\nWhat is Koi?")
	require.Contains(t, input, "ANSWER:\nI couldn't reliably identify Koi.")

	schema := body["text"].(map[string]any)["format"].(map[string]any)["schema"].(map[string]any)
	verdict := schema["properties"].(map[string]any)["verdict"].(map[string]any)
	require.ElementsMatch(t, []any{"mention", "no_info", "other_company", "other_meaning"}, verdict["enum"])
}

func TestJudgeMentionWithoutReportedCost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, mentionResponse(`{"verdict":"mention"}`, ""))
	}))
	defer server.Close()

	judgment, err := testClient(server.URL).JudgeMention(t.Context(), mentionInput())
	require.NoError(t, err)

	require.Equal(t, domain.MentionVerdictMention, judgment.Verdict)
	require.Nil(t, judgment.CostUSD)
}

func TestJudgeMentionRejectsInvalidOutput(t *testing.T) {
	for _, output := range []string{
		`{"verdict":"maybe"}`,
		`{"verdict":""}`,
		`{"verdict":"mention","reason":"extra"}`,
		`not json`,
	} {
		t.Run(output, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, mentionResponse(output, mentionUsage))
			}))
			defer server.Close()

			_, err := testClient(server.URL).JudgeMention(t.Context(), mentionInput())

			var providerErr *Error
			require.ErrorAs(t, err, &providerErr)
			require.Equal(t, "invalid_output", providerErr.Code)
		})
	}
}

func TestJudgeMentionClassifiesProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprint(w, `{"error":{"message":"slow down"}}`)
	}))
	defer server.Close()

	_, err := testClient(server.URL).JudgeMention(t.Context(), mentionInput())

	var providerErr *Error
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, "provider_rate_limited", providerErr.Code)
}

func TestBoundAnswerCutsOnCharacterBoundary(t *testing.T) {
	require.Equal(t, "short", boundAnswer("short"))

	bounded := boundAnswer(strings.Repeat("a", maxMentionAnswerBytes-1) + "è and more")

	require.LessOrEqual(t, len(bounded), maxMentionAnswerBytes)
	require.True(t, utf8.ValidString(bounded))
}
