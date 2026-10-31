package koiapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koidomain"
)

type staticCompanySearcher struct{}

func (staticCompanySearcher) Search(context.Context, string) ([]koidomain.CompanySearchResult, error) {
	return nil, nil
}

func TestCapabilitiesReflectOptionalSearch(t *testing.T) {
	for _, tc := range []struct {
		name            string
		searcher        koidomain.CompanySearcher
		searchEnabled   bool
		draftingEnabled bool
		logoKey         string
		expectedLogoKey string
	}{
		{
			name: "unconfigured",
		},
		{
			name:          "search configured",
			searcher:      staticCompanySearcher{},
			searchEnabled: true,
		},
		{
			name:            "drafting configured",
			draftingEnabled: true,
		},
		{
			name:            "publishable logo key",
			logoKey:         "pk_test",
			expectedLogoKey: "pk_test",
		},
		{
			name:    "secret logo key is not exposed",
			logoKey: "sk_test",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VITE_LOGODEV_PUBLISHABLE_KEY", tc.logoKey)

			sess := scs.New()
			sess.Cookie.Name = "koi_session"
			app := NewServer(sessionStore(t), sess, tc.searcher, tc.draftingEnabled)

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
			resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/capabilities", "")
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			resp.Body.Close()
			resp = doJSON(t, client, http.MethodGet, ts.URL+"/test-session", "")
			require.Equal(t, http.StatusNoContent, resp.StatusCode)
			resp.Body.Close()
			resp = doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/capabilities", "")
			var body struct {
				CompetitorSearch      bool   `json:"competitorSearch"`
				AIDrafting            bool   `json:"aiDrafting"`
				LogoDevPublishableKey string `json:"logoDevPublishableKey"`
				MCPURL                string `json:"mcpUrl"`
			}
			readJSON(t, resp, &body)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, tc.searchEnabled, body.CompetitorSearch)
			require.Equal(t, tc.draftingEnabled, body.AIDrafting)
			require.Equal(t, tc.expectedLogoKey, body.LogoDevPublishableKey)
			require.Equal(t, oauthOrigin()+"/mcp", body.MCPURL)
		})
	}
}
