package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/auth"
)

func TestIngestProxyKeepsCredentialsBehind(t *testing.T) {
	var forwarded *http.Request
	var body string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded = r
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)

		http.SetCookie(w, &http.Cookie{Name: "upstream", Value: "1"})
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	proxy := newIngestProxy(upstream.URL, upstream.URL)
	req := httptest.NewRequest(http.MethodPost, "/ingest/e/?compression=gzip-js", strings.NewReader("events"))
	req.Header.Set("Cookie", "koi_session=secret")
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Forwarded-For", "203.0.113.7")

	res := httptest.NewRecorder()
	proxy.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, "/e/", forwarded.URL.Path)
	require.Equal(t, "compression=gzip-js", forwarded.URL.RawQuery)
	require.Equal(t, "events", body)
	require.Empty(t, forwarded.Header.Get("Cookie"))
	require.Empty(t, forwarded.Header.Get("Authorization"))
	require.True(t, strings.HasPrefix(forwarded.Header.Get("X-Forwarded-For"), "203.0.113.7"))
	require.Empty(t, res.Header().Values("Set-Cookie"))
}

func TestIngestProxySendsScriptsToTheAssetHost(t *testing.T) {
	ingest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer ingest.Close()

	assets := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	defer assets.Close()

	proxy := newIngestProxy(ingest.URL, assets.URL)
	for _, path := range []string{"/static/recorder.js", "/array/phc_key/config.js"} {
		res := httptest.NewRecorder()
		proxy.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/ingest"+path, nil))

		require.Equal(t, http.StatusOK, res.Code)
		require.Equal(t, path, res.Body.String())
	}
}

func TestCapabilitiesAndIngestFollowBrowserAnalytics(t *testing.T) {
	for _, tc := range []struct {
		name    string
		browser *BrowserAnalytics
		want    *BrowserAnalyticsRes
	}{
		{name: "off"},
		{
			name: "on",
			browser: &BrowserAnalytics{
				Key:            "phc_test",
				IngestHost:     "https://eu.i.posthog.com",
				AssetsHost:     "https://eu-assets.i.posthog.com",
				UIHost:         "https://eu.posthog.com",
				ExcludedEmails: []string{"team@example.com"},
			},
			want: &BrowserAnalyticsRes{
				Key:            "phc_test",
				APIHost:        "/ingest",
				UIHost:         "https://eu.posthog.com",
				ExcludedEmails: []string{"team@example.com"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sess := scs.New()
			sess.Cookie.Name = "koi_session"
			app := NewServer(sessionStore(t), sess, ServerOptions{Browser: tc.browser})

			testSession := sess.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sess.Put(r.Context(), "user_id", uuid.NewString())
				w.WriteHeader(http.StatusNoContent)
			}))
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/test-session" {
					testSession.ServeHTTP(w, r)

					return
				}

				app.ServeHTTP(w, r)
			}))
			defer ts.Close()

			client := newCookieClient(t)
			resp := doJSON(t, client, http.MethodGet, ts.URL+"/test-session", "")
			resp.Body.Close()

			resp = doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/capabilities", "")
			var body struct {
				Analytics *BrowserAnalyticsRes `json:"analytics"`
			}
			readJSON(t, resp, &body)
			require.Equal(t, tc.want, body.Analytics)

			if tc.browser == nil {
				resp = doJSON(t, client, http.MethodPost, ts.URL+"/ingest/e/", "{}")
				resp.Body.Close()
				require.NotEqual(t, http.StatusOK, resp.StatusCode, "no proxy without analytics")
			}
		})
	}
}

func TestMCPToolCallsAreCountedOnSuccessOnly(t *testing.T) {
	identity := auth.Identity{UserID: uuid.New(), OrganizationID: uuid.New()}
	ctx := auth.WithIdentity(context.Background(), identity)
	call := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "prompts_list", Arguments: json.RawMessage(`{}`)}}

	for _, tc := range []struct {
		name   string
		result mcp.Result
		err    error
		want   int
	}{
		{name: "success", result: &mcp.CallToolResult{}, want: 1},
		{name: "tool error", result: &mcp.CallToolResult{IsError: true}},
		{name: "protocol error", err: errors.New("boom")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			analytics := &countingAnalytics{}
			handler := trackMCPToolCalls(analytics)(func(context.Context, string, mcp.Request) (mcp.Result, error) {
				return tc.result, tc.err
			})

			_, _ = handler(ctx, "tools/call", call)
			_, _ = handler(ctx, "tools/list", call)

			require.Len(t, analytics.events, tc.want)
			if tc.want == 1 {
				require.Equal(t, "mcp_tool_called", analytics.events[0])
				require.Equal(t, identity.UserID, analytics.userID)
				require.Equal(t, "prompts_list", analytics.props["tool"])
			}
		})
	}
}

type countingAnalytics struct {
	events []string
	userID uuid.UUID
	props  map[string]any
}

func (a *countingAnalytics) Track(userID, _ uuid.UUID, event string, props map[string]any) {
	a.events = append(a.events, event)
	a.userID = userID
	a.props = props
}
