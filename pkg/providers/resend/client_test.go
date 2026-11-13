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

type sentEmail struct {
	key     string
	from    string
	to      []string
	subject string
	text    string
	html    string
}

// fakeResend returns a client whose requests land in the returned email.
func fakeResend(t *testing.T, publicURL string) (*Client, *sentEmail) {
	t.Helper()

	sent := &sentEmail{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/emails", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var body struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			Text    string   `json:"text"`
			Html    string   `json:"html"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		*sent = sentEmail{
			key:     r.Header.Get("Idempotency-Key"),
			from:    body.From,
			to:      body.To,
			subject: body.Subject,
			text:    body.Text,
			html:    body.Html,
		}
		_, _ = w.Write([]byte(`{"id":"email-id"}`))
	}))
	t.Cleanup(server.Close)

	client := resend.NewClient("test-key")
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL

	return &Client{client: client, from: "Koi <hello@mail.getkoi.co>", publicURL: publicURL}, sent
}

func TestEmails(t *testing.T) {
	userID := uuid.New()
	runID := uuid.New()

	for name, tc := range map[string]struct {
		send    func(*Client) error
		key     string
		subject string
		text    []string
		html    []string
	}{
		"welcome": {
			send:    func(c *Client) error { return c.SendWelcome(t.Context(), userID, "you@example.com") },
			key:     "welcome-user/" + userID.String(),
			subject: "Welcome to Koi",
			text:    []string{"Set up your brand: https://app.getkoi.co/setup"},
			html:    []string{`href="https://app.getkoi.co/setup"`, "See how AI talks about your brand."},
		},
		"first check ready": {
			send: func(c *Client) error {
				return c.SendFirstCheckReady(t.Context(), runID, "you@example.com", "Acme & <Co>")
			},
			key:     "first-check-run/" + runID.String(),
			subject: "We asked AI about Acme & <Co>",
			text:    []string{"Your first check for Acme & <Co> is ready.", "See what they said: https://app.getkoi.co/"},
			// The brand is the founder's own text, so the markup escapes it.
			html: []string{"Acme &amp; &lt;Co&gt;", `href="https://app.getkoi.co/"`},
		},
		// A reset carries a new token on every retry, so a replayed request must not be deduplicated.
		"password reset": {
			send:    func(c *Client) error { return c.SendPasswordReset(t.Context(), "you@example.com", "secret-token") },
			subject: "Reset your Koi password",
			text:    []string{"https://app.getkoi.co/reset-password#secret-token"},
			html:    []string{`href="https://app.getkoi.co/reset-password#secret-token"`},
		},
		// A second reset must produce a second notice, so there is no idempotency key either.
		"password changed": {
			send:    func(c *Client) error { return c.SendPasswordChanged(t.Context(), userID, "you@example.com") },
			subject: "Your Koi password was reset",
			text:    []string{"We signed you out everywhere", "If this was not you"},
			html:    []string{"Your password was reset", "reset your password again"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			client, sent := fakeResend(t, "https://app.getkoi.co/")

			require.NoError(t, tc.send(client))

			require.Equal(t, tc.key, sent.key)
			require.Equal(t, "Koi <hello@mail.getkoi.co>", sent.from)
			require.Equal(t, []string{"you@example.com"}, sent.to)
			require.Equal(t, tc.subject, sent.subject)
			for _, want := range tc.text {
				require.Contains(t, sent.text, want)
			}
			for _, want := range tc.html {
				require.Contains(t, sent.html, want)
			}
		})
	}
}

func TestWelcomeContentWithoutPublicURL(t *testing.T) {
	text, markup := welcomeContent("")

	require.Contains(t, text, "sign in and add your brand")
	require.Contains(t, markup, "sign in and add your brand")
	require.NotContains(t, markup, "href=")
}
