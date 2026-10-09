package cloro

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/koihq/koi/pkg/koidomain"
)

type Client struct {
	http *resty.Client
}

func New(key string) *Client {
	return &Client{http: resty.New().SetBaseURL("https://api.cloro.dev").
		SetAuthToken(strings.TrimSpace(key)).SetTimeout(10 * time.Second)}
}

type Error struct {
	Code        string
	Retryable   bool
	AccountWide bool
}

func (e *Error) Error() string { return e.Code }

type Task struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type taskResponse struct {
	Task     Task            `json:"task"`
	Response json.RawMessage `json:"response"`
}

type Answer struct {
	Text    string
	Sources []string
}

func (c *Client) Submit(ctx context.Context, cell koidomain.VisibilityResult, country string) (string, error) {
	payload := map[string]any{"prompt": cell.PromptText, "country": country}
	var taskType string
	switch cell.Engine {
	case koidomain.EngineChatGPT:
		taskType = "CHATGPT"
		payload["disableWebSearch"] = false
	case koidomain.EnginePerplexity:
		taskType = "PERPLEXITY"
	case koidomain.EngineGemini:
		taskType = "GEMINI"
	case koidomain.EngineGoogleAIOverviews:
		taskType = "GOOGLE"
		payload = map[string]any{
			"query":   cell.PromptText,
			"country": country,
			"pages":   1,
			"include": map[string]any{
				"aioverview": map[string]any{},
			},
		}
	default:
		return "", &Error{Code: "cloro_engine_unsupported"}
	}

	resp, err := c.http.R().SetContext(ctx).SetBody(map[string]any{
		"taskType":       taskType,
		"idempotencyKey": "koi:" + cell.ID.String(),
		"payload":        payload,
	}).Post("/v1/async/task")
	if err != nil {
		return "", &Error{Code: "cloro_unavailable", Retryable: true}
	}
	if err := responseError(resp); err != nil {
		return "", err
	}

	var result taskResponse
	if json.Unmarshal(resp.Body(), &result) != nil || result.Task.ID == "" {
		// A successful but unreadable submission may already have spent credits.
		return "", &Error{Code: "cloro_unavailable", Retryable: true}
	}

	return result.Task.ID, nil
}

// Poll reads response directly: async responses do not have the sync result envelope.
func (c *Client) Poll(ctx context.Context, id string, engine koidomain.EngineID) (*Answer, error) {
	resp, err := c.http.R().SetContext(ctx).Get("/v1/async/task/" + url.PathEscape(id))
	if err != nil {
		return nil, &Error{Code: "cloro_unavailable", Retryable: true}
	}
	if err := responseError(resp); err != nil {
		return nil, err
	}

	var result taskResponse
	if json.Unmarshal(resp.Body(), &result) != nil || result.Task.ID != id {
		return nil, &Error{Code: "cloro_invalid_response"}
	}
	switch result.Task.Status {
	case "QUEUED", "PROCESSING":
		return nil, nil
	case "FAILED":
		return nil, &Error{Code: "cloro_task_failed"}
	case "COMPLETED":
		return normalize(engine, result.Response)
	default:
		return nil, &Error{Code: "cloro_invalid_response"}
	}
}

func responseError(resp *resty.Response) error {
	if resp.IsSuccess() {
		return nil
	}
	e := &Error{Code: "cloro_request_failed"}
	switch resp.StatusCode() {
	case 400, 422:
		e.Code = "cloro_invalid_request"
	case 401:
		e.Code, e.AccountWide = "cloro_authentication", true
	case 402, 403:
		e.Code, e.AccountWide = "cloro_credits", true
	case 404:
		e.Code = "cloro_task_missing"
	case 409:
		e.Code = "cloro_submission_unknown"
	case 408, 429:
		e.Code, e.Retryable = "cloro_unavailable", true
	default:
		e.Retryable = resp.StatusCode() >= 500
	}

	return e
}

type providerAnswer struct {
	Text    *string `json:"text"`
	Sources []struct {
		URL string `json:"url"`
	} `json:"sources"`
}

func normalize(engine koidomain.EngineID, raw json.RawMessage) (*Answer, error) {
	invalid := &Error{Code: "cloro_invalid_response"}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, invalid
	}

	if engine == koidomain.EngineGoogleAIOverviews {
		var google struct {
			Overview json.RawMessage `json:"aioverview"`
		}
		if json.Unmarshal(raw, &google) != nil {
			return nil, invalid
		}
		if len(google.Overview) == 0 || string(google.Overview) == "null" {
			return &Answer{Sources: []string{}}, nil
		}
		raw = google.Overview
	}

	var data providerAnswer
	if json.Unmarshal(raw, &data) != nil || data.Text == nil || strings.TrimSpace(*data.Text) == "" {
		return nil, invalid
	}
	answer := &Answer{Text: *data.Text, Sources: []string{}}
	seen := map[string]bool{}
	for _, source := range data.Sources {
		if source.URL != "" && !seen[source.URL] {
			seen[source.URL] = true
			answer.Sources = append(answer.Sources, source.URL)
		}
	}

	return answer, nil
}

func Failure(err error) *Error {
	var result *Error
	if errors.As(err, &result) {
		return result
	}

	return &Error{Code: "cloro_unavailable", Retryable: true}
}
