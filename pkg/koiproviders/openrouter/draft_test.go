package openrouter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/retry"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/stretchr/testify/require"
)

func testClient(url string) *Client {
	return &Client{SDK: sdk.New(sdk.WithSecurity("test-key"), sdk.WithServerURL(url), sdk.WithTimeout(time.Second), sdk.WithRetryConfig(retry.Config{Strategy: "none"}))}
}

func TestGenerateUsesSchemaAndAllFittingEntries(t *testing.T) {
	input := Input{Brand: koidomain.BrandSnapshot{Name: "Koi", Domain: "koi.dev", Country: "IT"}, Mode: koidomain.GenerationInitial}
	for i := range 120 {
		input.ExistingPrompts = append(input.ExistingPrompts, fmt.Sprintf("Question %d?", i))
	}
	bounded, fits := BoundInput(input, koidomain.GenerationModelSlug, 128<<10)
	require.True(t, fits)
	require.Len(t, bounded.ExistingPrompts, 120)
	require.LessOrEqual(t, requestSize(koidomain.GenerationModelSlug, bounded), 128<<10)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		var body struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
			ResponseFormat struct {
				Type       string `json:"type"`
				JSONSchema struct {
					Name   string `json:"name"`
					Strict bool   `json:"strict"`
				} `json:"json_schema"`
			} `json:"response_format"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "json_schema", body.ResponseFormat.Type)
		require.Equal(t, "koi_draft_v1", body.ResponseFormat.JSONSchema.Name)
		require.True(t, body.ResponseFormat.JSONSchema.Strict)
		var sent Input
		require.NoError(t, json.Unmarshal([]byte(body.Messages[1].Content), &sent))
		require.Len(t, sent.ExistingPrompts, 120)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":"test","object":"chat.completion","created":1,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"{\"prompts\":[\"Which tool?\"],\"competitors\":[{\"name\":\"Atlas\",\"domain\":\"atlas.example\"}]}"},"finish_reason":"stop"}]}`)
	}))
	defer server.Close()
	draft, err := testClient(server.URL).Generate(t.Context(), koidomain.GenerationModelSlug, bounded)
	require.NoError(t, err)
	require.Equal(t, []string{"Which tool?"}, draft.Prompts)
	require.Equal(t, []Company{{Name: "Atlas", Domain: "atlas.example"}}, draft.Competitors)
	require.EqualValues(t, 1, calls.Load())
}

func TestGenerateRejectsMalformedOutputWithoutAnotherCall(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":"test","object":"chat.completion","created":1,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"{\"prompts\":[],\"competitors\":{}}"},"finish_reason":"stop"}]}`)
	}))
	defer server.Close()
	_, err := testClient(server.URL).Generate(t.Context(), "test", Input{Brand: koidomain.BrandSnapshot{Name: "Koi"}, Mode: koidomain.GenerationInitial})
	var providerErr *Error
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, "invalid_output", providerErr.Code)
	require.EqualValues(t, 1, calls.Load())
}

func TestGenerateDoesNotRetrySDKFailures(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":{"message":"temporary"}}`)
	}))
	defer server.Close()
	_, err := testClient(server.URL).Generate(t.Context(), "test", Input{Brand: koidomain.BrandSnapshot{Name: "Koi"}, Mode: koidomain.GenerationInitial})
	var providerErr *Error
	require.ErrorAs(t, err, &providerErr)
	require.True(t, providerErr.Retryable)
	require.Equal(t, 500, providerErr.Status)
	require.EqualValues(t, 1, calls.Load())
}

func TestBoundInputSkipsOversizedEntryAndKeepsLaterEntries(t *testing.T) {
	input := Input{Brand: koidomain.BrandSnapshot{Name: "Koi"}, Mode: koidomain.GenerationPrompts,
		ExistingPrompts: []string{strings.Repeat("x", 2000), "short"}}
	limit := requestSize("test", Input{Brand: input.Brand, Mode: input.Mode, ExistingPrompts: []string{"short"}, ExistingCompetitors: []Company{}})
	bounded, fits := BoundInput(input, "test", limit)
	require.True(t, fits)
	require.Equal(t, []string{"short"}, bounded.ExistingPrompts)
}
