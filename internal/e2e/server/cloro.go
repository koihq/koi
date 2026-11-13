package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// Every engine gives the same answer: it names the journey's brand, cites its site and names a
// rival, so a run produces cited results and a competitor appearance without calling Cloro.
const fakeAnswer = "For startups, Acme is a strong choice and Rival is a common alternative."

var fakeSources = []string{"https://acme.example/pricing", "https://rival.example"}

// startFakeCloro serves the parts of the Cloro API that Koi calls: synchronous monitors and
// asynchronous tasks that complete on the first poll.
func startFakeCloro() *httptest.Server {
	var mu sync.Mutex
	tasks := map[string]string{}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/monitor/{engine}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"success": true,
			"result":  answerFor(r.PathValue("engine") == "google"),
		})
	})

	mux.HandleFunc("POST /v1/async/task", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			TaskType       string `json:"taskType"`
			IdempotencyKey string `json:"idempotencyKey"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.IdempotencyKey == "" {
			http.Error(w, `{"error":{"code":"INVALID_REQUEST"}}`, http.StatusBadRequest)
			return
		}

		// The idempotency key names the result, so a retried submission returns the same task.
		id := "task-" + strings.TrimPrefix(body.IdempotencyKey, "koi:")

		mu.Lock()
		tasks[id] = body.TaskType
		mu.Unlock()

		writeJSON(w, map[string]any{"task": map[string]string{"id": id, "status": "QUEUED"}})
	})

	mux.HandleFunc("GET /v1/async/task/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		mu.Lock()
		taskType, ok := tasks[id]
		mu.Unlock()

		if !ok {
			http.Error(w, `{"error":{"code":"NOT_FOUND"}}`, http.StatusNotFound)
			return
		}

		writeJSON(w, map[string]any{
			"task":     map[string]string{"id": id, "status": "COMPLETED"},
			"response": answerFor(taskType == "GOOGLE"),
		})
	})

	return httptest.NewServer(mux)
}

func answerFor(overview bool) any {
	sources := make([]map[string]string, 0, len(fakeSources))
	for _, source := range fakeSources {
		sources = append(sources, map[string]string{"url": source})
	}

	answer := map[string]any{
		"text":     fakeAnswer,
		"markdown": fakeAnswer,
		"sources":  sources,
	}
	if overview {
		return map[string]any{"aioverview": answer}
	}

	return answer
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Credits-Charged", "0")

	_ = json.NewEncoder(w).Encode(body)
}
