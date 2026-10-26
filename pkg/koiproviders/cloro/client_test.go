package cloro

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koidomain"
)

func TestAsyncContracts(t *testing.T) {
	for _, engine := range koidomain.VisibilityEngines() {
		t.Run(string(engine), func(t *testing.T) {
			cell := koidomain.VisibilityResult{
				BaseEntity: koidomain.BaseEntity{ID: uuid.New()},
				Engine:     engine,
				PromptText: "Best analytics tools?",
			}
			taskID := uuid.NewString()
			var keys []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodPost {
					require.Equal(t, "/v1/async/task", r.URL.Path)
					var request struct {
						TaskType string         `json:"taskType"`
						Key      string         `json:"idempotencyKey"`
						Payload  map[string]any `json:"payload"`
						Webhook  any            `json:"webhook"`
						Priority any            `json:"priority"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
					keys = append(keys, request.Key)
					require.Equal(t, "US", request.Payload["country"])
					require.Nil(t, request.Webhook)
					require.Nil(t, request.Priority)
					if engine == koidomain.EngineGoogleAIOverviews {
						require.Equal(t, "GOOGLE", request.TaskType)
						require.Equal(t, cell.PromptText, request.Payload["query"])
						require.Equal(t, float64(1), request.Payload["pages"])
						require.Equal(t, map[string]any{"aioverview": map[string]any{"markdown": true}}, request.Payload["include"])
						require.Len(t, request.Payload, 4)
					} else {
						wantType := map[koidomain.EngineID]string{
							koidomain.EngineChatGPT:    "CHATGPT",
							koidomain.EnginePerplexity: "PERPLEXITY",
							koidomain.EngineGemini:     "GEMINI",
						}[engine]
						require.Equal(t, wantType, request.TaskType)
						require.Equal(t, cell.PromptText, request.Payload["prompt"])
						require.Equal(t, map[string]any{"markdown": true}, request.Payload["include"])
						if engine == koidomain.EngineChatGPT {
							require.Equal(t, false, request.Payload["disableWebSearch"])
						} else {
							require.Len(t, request.Payload, 3)
						}
					}
					fmt.Fprintf(w, `{"success":true,"task":{"id":%q,"status":"QUEUED"}}`, taskID)
					return
				}

				require.Equal(t, "/v1/async/task/"+taskID, r.URL.Path)
				w.Header().Set("X-Credits-Charged", "4")
				response := `{"text":"Koi is an option.","markdown":"1. [Acme](https://acme.dev)","entities":[{"type":"software","name":"Acme"}],"sources":[{"url":"https://koi.dev"},{"url":"https://koi.dev"}]}`
				if engine == koidomain.EngineGoogleAIOverviews {
					response = `{"aioverview":` + response + `,"organic":[{"url":"https://ignore.test"}]}`
				}
				fmt.Fprintf(w, `{"task":{"id":%q,"status":"COMPLETED"},"response":%s}`, taskID, response)
			}))
			defer server.Close()
			client := New("test-key")
			client.http.SetBaseURL(server.URL)

			for range 2 {
				id, err := client.Submit(t.Context(), cell, "US")
				require.NoError(t, err)
				require.Equal(t, taskID, id)
			}
			require.Equal(t, []string{"koi:" + cell.ID.String(), "koi:" + cell.ID.String()}, keys)
			answer, err := client.Poll(t.Context(), taskID, engine)
			require.NoError(t, err)
			require.Equal(t, "Koi is an option.", answer.Text)
			require.Equal(t, "1. [Acme](https://acme.dev)", answer.Markdown)
			require.Equal(t, []koidomain.NamedEntity{{Type: "software", Name: "Acme"}}, answer.Entities)
			require.Equal(t, []string{"https://koi.dev"}, answer.Sources)
			require.Equal(t, 4, *answer.CreditsCharged)
		})
	}
}

func TestTaskResponsesAndErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		status  int
		body    string
		engine  koidomain.EngineID
		code    string
		retry   bool
		pending bool
	}{
		{name: "queued", body: `{"task":{"id":"t","status":"QUEUED"}}`, pending: true},
		{name: "processing", body: `{"task":{"id":"t","status":"PROCESSING"}}`, pending: true},
		{name: "failed", body: `{"task":{"id":"t","status":"FAILED"},"response":{"error":{"code":"x"}}}`, code: "cloro_task_failed"},
		{name: "no overview", body: `{"task":{"id":"t","status":"COMPLETED"},"response":{"organic":[{"url":"https://koi.dev"}]}}`, engine: koidomain.EngineGoogleAIOverviews},
		{name: "sync envelope rejected", body: `{"task":{"id":"t","status":"COMPLETED"},"response":{"result":{"text":"Koi"}}}`, code: "cloro_invalid_response"},
		{name: "missing response", body: `{"task":{"id":"t","status":"COMPLETED"}}`, code: "cloro_invalid_response"},
		{name: "blank answer", body: `{"task":{"id":"t","status":"COMPLETED"},"response":{"text":" "}}`, code: "cloro_invalid_response"},
		{name: "conflict has no ID", status: 409, code: "cloro_submission_unknown"},
		{name: "auth", status: 401, code: "cloro_authentication"},
		{name: "credits", status: 403, code: "cloro_credits"},
		{name: "unsupported country", status: 400, code: "cloro_invalid_request"},
		{name: "rate limit", status: 429, code: "cloro_unavailable", retry: true},
		{name: "unavailable", status: 503, code: "cloro_request_failed", retry: true},
		{name: "expired", status: 404, code: "cloro_task_missing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if test.status != 0 {
					w.WriteHeader(test.status)
				}
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client := New("key")
			client.http.SetBaseURL(server.URL)
			engine := test.engine
			if engine == "" {
				engine = koidomain.EngineChatGPT
			}
			answer, err := client.Poll(t.Context(), "t", engine)
			if test.code != "" {
				require.Error(t, err)
				require.Equal(t, test.code, Failure(err).Code)
				require.Equal(t, test.retry, Failure(err).Retryable)
			} else if test.pending {
				require.NoError(t, err)
				require.Nil(t, answer)
			} else {
				require.NoError(t, err)
				require.Empty(t, answer.Text)
				require.Equal(t, []string{}, answer.Sources)
			}
		})
	}
}

func TestMalformedOptionalMetadataKeepsAnswer(t *testing.T) {
	for _, metadata := range []string{
		`"markdown":{},"entities":"wrong"`,
		`"markdown":null,"entities":[{"type":42,"name":"Bad"},{"type":"software","name":"Good"}]`,
	} {
		answer, err := normalize(koidomain.EngineChatGPT, json.RawMessage(`{"text":"Valid answer","sources":[],`+metadata+`}`))
		require.NoError(t, err)
		require.Equal(t, "Valid answer", answer.Text)
		require.Empty(t, answer.Markdown)
		for _, entity := range answer.Entities {
			require.Equal(t, "Good", entity.Name)
		}
	}
}
