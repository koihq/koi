package koiapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_DBOk(t *testing.T) {
	store := NewMockStore(t)
	store.EXPECT().Ping(mock.Anything).Return(nil)
	h := NewHealthHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body HealthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, HealthResponse{Status: "ok", DB: "ok"}, body)
}

func TestHealthHandler_DBDown(t *testing.T) {
	store := NewMockStore(t)
	store.EXPECT().Ping(mock.Anything).Return(errors.New("no db"))
	h := NewHealthHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body HealthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, HealthResponse{Status: "ok", DB: "down"}, body)
}
