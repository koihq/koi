package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckHealth(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    int
		body    string
		wantErr bool
	}{
		{name: "healthy", code: http.StatusOK, body: `{"status":"ok","db":"ok"}`},
		{name: "database down", code: http.StatusOK, body: `{"status":"ok","db":"down"}`, wantErr: true},
		{name: "invalid response", code: http.StatusOK, body: `broken`, wantErr: true},
		{name: "http error", code: http.StatusServiceUnavailable, body: `{"status":"ok","db":"ok"}`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			err := checkHealth(server.URL)
			if (err != nil) != tc.wantErr {
				t.Fatalf("checkHealth() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
