package koiapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koidomain"
)

type stubStore struct {
	ping error
}

func (s stubStore) Atomic(context.Context, func(koidomain.Store) error) error {
	return nil
}

func (s stubStore) Ping(context.Context) error {
	return s.ping
}

func TestHealthHandler_DBOk(t *testing.T) {
	h := NewHealthHandler(stubStore{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body HealthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, HealthResponse{Status: "ok", DB: "ok"}, body)
}

func TestHealthHandler_DBDown(t *testing.T) {
	h := NewHealthHandler(stubStore{ping: errors.New("no db")})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body HealthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, HealthResponse{Status: "ok", DB: "down"}, body)
}
