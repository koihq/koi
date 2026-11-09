package posthogprovider

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTrackSkipsExcludedAccounts(t *testing.T) {
	var mu sync.Mutex
	var events []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body io.Reader = r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(r.Body)
			require.NoError(t, err)
			body = reader
		}

		var batch struct {
			Batch []map[string]any `json:"batch"`
		}
		_ = json.NewDecoder(body).Decode(&batch)

		mu.Lock()
		events = append(events, batch.Batch...)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	team := uuid.New()
	customer := uuid.New()
	organization := uuid.New()
	lookups := 0
	lookup := func(_ context.Context, userID uuid.UUID) (string, error) {
		lookups++
		if userID == team {
			return "Team@Example.com", nil
		}

		return "customer@example.com", nil
	}

	client, err := New("phc_test", server.URL, []string{" team@example.com "}, lookup)
	require.NoError(t, err)

	client.Track(team, organization, "user_signed_up", nil)
	client.Track(team, organization, "mcp_tool_called", nil)
	client.Track(customer, organization, "user_signed_up", map[string]any{"step": "website"})
	require.NoError(t, client.Close())

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, events, 1)
	require.Equal(t, customer.String(), events[0]["distinct_id"])
	require.Equal(t, "user_signed_up", events[0]["event"])

	properties := events[0]["properties"].(map[string]any)
	require.Equal(t, organization.String(), properties["organization_id"])
	require.Equal(t, "website", properties["step"])
	require.Equal(t, 2, lookups, "each user is looked up once")
}

func TestTrackWithoutExclusionsNeverLooksUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New("phc_test", server.URL, []string{"", " "}, func(context.Context, uuid.UUID) (string, error) {
		t.Fatal("lookup without exclusions")

		return "", nil
	})
	require.NoError(t, err)

	client.Track(uuid.New(), uuid.Nil, "user_signed_up", nil)
	require.NoError(t, client.Close())
}
