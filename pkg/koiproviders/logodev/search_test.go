package logodev

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koidomain"
)

func TestSearchMapsProviderResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		require.Equal(t, "acme labs", r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"Acme","domain":"acme.com"},{"name":"Bad","domain":"acme.com/path"},{"name":"","domain":"empty.com"}]`))
	}))

	defer server.Close()
	searcher := NewSearcher(" secret ")
	searcher.url = server.URL

	results, err := searcher.Search(t.Context(), "acme labs")
	require.NoError(t, err)
	require.Equal(t, []koidomain.CompanySearchResult{
		{
			Name:   "Acme",
			Domain: "acme.com",
		},
	}, results)
}

func TestSearchFailureAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))

	defer server.Close()
	searcher := NewSearcher("secret")
	searcher.url = server.URL
	_, err := searcher.Search(t.Context(), "acme")
	require.ErrorContains(t, err, "502")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = searcher.Search(ctx, "acme")
	require.ErrorIs(t, err, context.Canceled)
}

func TestSearchRejectsMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{`))
	}))

	defer server.Close()
	searcher := NewSearcher("secret")
	searcher.url = server.URL
	_, err := searcher.Search(t.Context(), "acme")
	require.Error(t, err)
}
