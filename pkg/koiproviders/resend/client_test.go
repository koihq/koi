package resendprovider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/resend/resend-go/v4"
	"github.com/stretchr/testify/require"
)

func TestSendWelcome(t *testing.T) {
	userID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/emails", r.URL.Path)
		require.Equal(t, "welcome-user/"+userID.String(), r.Header.Get("Idempotency-Key"))
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var body struct {
			From string   `json:"from"`
			To   []string `json:"to"`
			Text string   `json:"text"`
			Html string   `json:"html"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "Koi <hello@mail.getkoi.co>", body.From)
		require.Equal(t, []string{"you@example.com"}, body.To)
		require.NotEmpty(t, body.Text)
		require.Contains(t, body.Text, "Set up your brand: https://app.getkoi.co/setup")
		require.Contains(t, body.Html, `href="https://app.getkoi.co/setup"`)
		require.Contains(t, body.Html, "See how AI talks about your brand.")
		_, _ = w.Write([]byte(`{"id":"email-id"}`))
	}))
	defer server.Close()

	client := resend.NewClient("test-key")
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL

	sender := &Client{client: client, from: "Koi <hello@mail.getkoi.co>", publicURL: "https://app.getkoi.co"}
	require.NoError(t, sender.SendWelcome(t.Context(), userID, "you@example.com"))
}

func TestWelcomeContentWithoutPublicURL(t *testing.T) {
	text, markup := welcomeContent("")

	require.Contains(t, text, "sign in and add your brand")
	require.Contains(t, markup, "sign in and add your brand")
	require.NotContains(t, markup, "href=")
}
