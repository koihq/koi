package koiapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
)

type bearerTransport struct{ token string }

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(clone)
}
func mcpSession(t *testing.T, endpoint, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "integration-test", Version: "1"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: &http.Client{Transport: bearerTransport{token: token}}, DisableStandaloneSSE: true}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}
func TestMCPOAuthTokenScopeIsolationAndRevocation(t *testing.T) {
	db, err := koidatabase.Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, koidatabase.Migrate(db))
	store := koidatabase.NewStore(db)
	service := koidomain.NewCredentialService(store)
	server := httptest.NewServer(NewServer(store, scs.New(), nil, false))
	t.Cleanup(server.Close)
	first, second := newCookieClient(t), newCookieClient(t)
	tokens := make([]string, 2)
	ids := make([]uuid.UUID, 2)
	var firstUserID uuid.UUID
	for i, client := range []*http.Client{first, second} {
		response := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/signup", `{"email":"`+uuid.NewString()+`@example.com","password":"password123"}`)
		require.Equal(t, 201, response.StatusCode)
		var account AuthRes
		readJSON(t, response, &account)
		if i == 0 {
			firstUserID = uuid.MustParse(account.ID)
			response = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/credentials", `{"name":"Old path"}`)
			require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
			require.NoError(t, response.Body.Close())
		}

		issued, err := service.IssueTokens(t.Context(), uuid.MustParse(account.ID), "test-client", "Test agent", "koi:read koi:write", oauthOrigin()+"/mcp", uuid.Nil)
		require.NoError(t, err)
		tokens[i] = issued.AccessToken

		response = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/credentials", "")
		require.Equal(t, 200, response.StatusCode)
		var listed struct {
			Items []struct {
				ID uuid.UUID `json:"id"`
			} `json:"items"`
		}
		readJSON(t, response, &listed)
		require.Len(t, listed.Items, 1)
		ids[i] = listed.Items[0].ID
	}
	endpoint := server.URL + "/mcp"
	legacyToken, err := koidomain.RandomCredential()
	require.NoError(t, err)
	require.NoError(t, store.CredentialRepo().CreateCredential(t.Context(), &koidomain.Credential{
		UserID:    firstUserID,
		Name:      "Old personal token",
		TokenHash: koidomain.HashCredential(legacyToken),
		Kind:      "personal",
	}))
	legacyClient := mcp.NewClient(&mcp.Implementation{Name: "legacy-test", Version: "1"}, nil)
	_, err = legacyClient.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:             endpoint,
		HTTPClient:           &http.Client{Transport: bearerTransport{token: legacyToken}},
		DisableStandaloneSSE: true,
	}, nil)
	require.Error(t, err)

	owner := mcpSession(t, endpoint, tokens[0])
	other := mcpSession(t, endpoint, tokens[1])
	created, err := owner.CallTool(context.Background(), &mcp.CallToolParams{Name: "brands_create", Arguments: map[string]any{"name": "Koi", "domain": "koi.dev", "country": "IT"}})
	require.NoError(t, err)
	require.False(t, created.IsError)
	brandID := created.StructuredContent.(map[string]any)["id"].(string)
	found, err := owner.CallTool(context.Background(), &mcp.CallToolParams{Name: "brands_get", Arguments: map[string]any{"id": brandID}})
	require.NoError(t, err)
	require.False(t, found.IsError, "%s", found.Content[0].(*mcp.TextContent).Text)
	foreign, err := other.CallTool(context.Background(), &mcp.CallToolParams{Name: "brands_get", Arguments: map[string]any{"id": brandID}})
	require.NoError(t, err)
	require.True(t, foreign.IsError)
	revoked := doJSON(t, first, http.MethodDelete, server.URL+"/api/v1/credentials/"+ids[0].String(), "")
	require.Equal(t, 204, revoked.StatusCode)
	require.NoError(t, revoked.Body.Close())
	_, err = owner.ListTools(context.Background(), nil)
	require.Error(t, err)
}
