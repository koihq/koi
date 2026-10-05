package koiapp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koidomain"
)

func visibilityHTTP(t *testing.T, client *http.Client, method, url, body string, expected int, output any) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	require.NoError(t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, expected, resp.StatusCode, "response: %s", data)
	if output != nil {
		require.NoError(t, json.Unmarshal(data, output))
	}
}

func TestVisibilityHTTPWithoutProviderKeysSurvivesRestart(t *testing.T) {
	cfg := Config{
		DatabaseURL: testdb.URL(t), SessionSecret: "test-secret",
		WorkerConcurrency: 2, ProviderTimeout: 45 * time.Second, DBMaxOpenConns: 4,
	}
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: jar, Timeout: 2 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	t.Cleanup(client.CloseIdleConnections)
	cancel, done, address := startTestApp(t, &App{cfg: cfg}, client)
	t.Cleanup(cancel)
	base := "http://" + address + "/api/v1"
	visibilityHTTP(t, client, http.MethodPost, base+"/auth/signup",
		`{"email":"visibility-`+uuid.NewString()+`@example.com","password":"password123"}`, 201, nil)
	var capabilities map[string]bool
	visibilityHTTP(t, client, http.MethodGet, base+"/capabilities", "", 200, &capabilities)
	require.False(t, capabilities["aiDrafting"])
	var brand koidomain.Brand
	visibilityHTTP(t, client, http.MethodPost, base+"/brands",
		`{"name":"Koi","domain":"koi.dev","country":"IT"}`, 201, &brand)
	brandURL := base + "/brands/" + brand.ID.String()
	visibilityHTTP(t, client, http.MethodPost, brandURL+"/runs", "", 409, nil)
	for _, text := range []string{"Best analytics for startups?", "Alternatives to Example Analytics?"} {
		body, err := json.Marshal(map[string]string{"text": text})
		require.NoError(t, err)
		visibilityHTTP(t, client, http.MethodPost, brandURL+"/prompts", string(body), 201, nil)
	}
	var accepted struct {
		ID        uuid.UUID `json:"id"`
		StatusURL string    `json:"status_url"`
	}
	visibilityHTTP(t, client, http.MethodPost, brandURL+"/runs", "", 202, &accepted)
	var status struct {
		State     koidomain.VisibilityState `json:"state"`
		Completed int                       `json:"completed_results"`
		Total     int                       `json:"total_results"`
	}
	require.Eventually(t, func() bool {
		visibilityHTTP(t, client, http.MethodGet, "http://"+address+accepted.StatusURL, "", 200, &status)
		return status.State == koidomain.VisibilitySucceeded
	}, 10*time.Second, 50*time.Millisecond)
	require.Equal(t, 8, status.Completed)
	require.Equal(t, 8, status.Total)
	var results koidomain.Page[koidomain.VisibilityResult]
	resultsPath := accepted.StatusURL + "/results?page=1&pageSize=40"
	visibilityHTTP(t, client, http.MethodGet, "http://"+address+resultsPath, "", 200, &results)
	require.Len(t, results.Items, 8)
	for _, result := range results.Items {
		require.NoError(t, result.Validate())
		require.Equal(t, koidomain.CollectorStub, result.Collector)
	}
	otherJar, err := cookiejar.New(nil)
	require.NoError(t, err)
	other := &http.Client{Jar: otherJar, Timeout: 2 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	t.Cleanup(other.CloseIdleConnections)
	visibilityHTTP(t, other, http.MethodPost, base+"/auth/signup",
		`{"email":"visibility-`+uuid.NewString()+`@example.com","password":"password123"}`, 201, nil)
	for _, path := range []string{brandURL + "/runs", "http://" + address + accepted.StatusURL, "http://" + address + resultsPath} {
		visibilityHTTP(t, other, http.MethodGet, path, "", 404, nil)
	}
	visibilityHTTP(t, other, http.MethodPost, brandURL+"/runs", "", 404, nil)
	visibilityHTTP(t, http.DefaultClient, http.MethodGet, "http://"+address+accepted.StatusURL, "", 401, nil)
	cancel()
	require.NoError(t, waitApp(t, done))
	// A new app instance reads the same run and results with the persisted cookie session.
	cancel, done, address = startTestApp(t, &App{cfg: cfg}, client)
	t.Cleanup(cancel)
	var reloaded koidomain.Page[koidomain.VisibilityResult]
	visibilityHTTP(t, client, http.MethodGet, "http://"+address+resultsPath, "", 200, &reloaded)
	require.Equal(t, results, reloaded)
	visibilityHTTP(t, client, http.MethodGet, "http://"+address+accepted.StatusURL, "", 200, &status)
	require.Equal(t, koidomain.VisibilitySucceeded, status.State)
	cancel()
	require.NoError(t, waitApp(t, done))
}
