package koiapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/alexedwards/scs/v2"
	"github.com/stretchr/testify/require"
)

func TestSPAFallback(t *testing.T) {
	files := http.FS(fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<html>Koi</html>"),
		},
		"assets/app.js": &fstest.MapFile{
			Data: []byte("app();"),
		},
	})
	handler := spaHandler(files)
	for _, tc := range []struct {
		method, path string
		status       int
		body         string
	}{
		{http.MethodGet, "/", 200, "<html>Koi</html>"},
		{http.MethodGet, "/prompts", 200, "<html>Koi</html>"},
		{http.MethodGet, "/settings/brand", 200, "<html>Koi</html>"},
		{http.MethodHead, "/mentions", 200, ""},
		{http.MethodGet, "/assets/app.js", 200, "app();"},
		{http.MethodGet, "/assets/missing.js", 404, ""},
		{http.MethodGet, "/assets/missing", 404, ""},
		{http.MethodGet, "/api/missing", 404, ""},
		{http.MethodGet, "/health/missing", 404, ""},
		{http.MethodPost, "/prompts", 404, ""},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code)
			if tc.status == 200 {
				require.Equal(t, tc.body, response.Body.String())
			}
			require.Equal(t, tc.path, request.URL.Path)
		})
	}
}

func TestEmbeddedSPARouting(t *testing.T) {
	handler := NewServer(NewMockStore(t), scs.New(), nil, false)
	for _, route := range []string{"/prompts", "/competitors", "/mentions", "/settings/brand"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
		require.Equal(t, http.StatusOK, response.Code)
		require.Contains(t, response.Header().Get("Content-Type"), "text/html")
		require.Contains(t, response.Body.String(), "<html")
	}

	for _, route := range []string{"/api/v1/missing", "/assets/missing.js"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
		require.Equal(t, http.StatusNotFound, response.Code)
	}
}
