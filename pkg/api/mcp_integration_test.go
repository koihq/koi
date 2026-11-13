package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/domain"
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
	server := newIntegrationServer(t)
	store := server.Store
	service := domain.NewCredentialService(store)

	first, firstAccount := server.signup(t)
	second, secondAccount := server.signup(t)
	tokens := make([]string, 2)
	ids := make([]uuid.UUID, 2)
	firstUserID := uuid.MustParse(firstAccount.ID)

	accounts := []AuthRes{firstAccount, secondAccount}

	for i, client := range []*http.Client{first, second} {
		if i == 0 {
			response := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/credentials", `{"name":"Old path"}`)
			require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
			require.NoError(t, response.Body.Close())
		}

		issued, err := service.IssueTokens(t.Context(), uuid.MustParse(accounts[i].ID), "test-client", "Test agent", "koi:read koi:write", oauthOrigin()+"/mcp", uuid.Nil)
		require.NoError(t, err)
		tokens[i] = issued.AccessToken

		response := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/credentials", "")
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
	legacyToken, err := domain.RandomCredential()
	require.NoError(t, err)
	require.NoError(t, store.CredentialRepo().CreateCredential(t.Context(), &domain.Credential{
		UserID:    firstUserID,
		Name:      "Old personal token",
		TokenHash: domain.HashCredential(legacyToken),
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
	prompt, err := owner.CallTool(context.Background(), &mcp.CallToolParams{Name: "prompts_create", Arguments: map[string]any{"brandId": brandID, "text": "Best tools?"}})
	require.NoError(t, err)
	require.False(t, prompt.IsError)
	promptID := prompt.StructuredContent.(map[string]any)["id"].(string)
	stopped, err := owner.CallTool(context.Background(), &mcp.CallToolParams{Name: "prompts_update", Arguments: map[string]any{"id": promptID, "inactive": true}})
	require.NoError(t, err)
	require.False(t, stopped.IsError)
	for inactive, want := range map[bool]int{true: 1, false: 0} {
		listed, err := owner.CallTool(context.Background(), &mcp.CallToolParams{Name: "prompts_list", Arguments: map[string]any{"brandId": brandID, "inactive": inactive}})
		require.NoError(t, err)
		require.False(t, listed.IsError)
		require.Len(t, listed.StructuredContent.(map[string]any)["items"], want)
	}
	deleted, err := owner.CallTool(context.Background(), &mcp.CallToolParams{Name: "prompts_delete", Arguments: map[string]any{"id": promptID}})
	require.NoError(t, err)
	require.False(t, deleted.IsError, "a prompt with no saved answers can be deleted")
	empty, err := owner.CallTool(context.Background(), &mcp.CallToolParams{Name: "tracking_start", Arguments: map[string]any{"brandId": brandID}})
	require.NoError(t, err)
	require.True(t, empty.IsError, "tracking needs an active prompt")
	tracked, err := owner.CallTool(context.Background(), &mcp.CallToolParams{Name: "prompts_create", Arguments: map[string]any{"brandId": brandID, "text": "Best tracked tools?"}})
	require.NoError(t, err)
	require.False(t, tracked.IsError)
	foreignStart, err := other.CallTool(context.Background(), &mcp.CallToolParams{Name: "tracking_start", Arguments: map[string]any{"brandId": brandID}})
	require.NoError(t, err)
	require.True(t, foreignStart.IsError)
	for range 2 {
		started, err := owner.CallTool(context.Background(), &mcp.CallToolParams{Name: "tracking_start", Arguments: map[string]any{"brandId": brandID}})
		require.NoError(t, err)
		require.False(t, started.IsError, "%s", started.Content[0].(*mcp.TextContent).Text)
	}
	tracking, err := owner.CallTool(context.Background(), &mcp.CallToolParams{Name: "tracking_get", Arguments: map[string]any{"brandId": brandID}})
	require.NoError(t, err)
	require.False(t, tracking.IsError)
	require.Equal(t, true, tracking.StructuredContent.(map[string]any)["confirmed"])
	revoked := doJSON(t, first, http.MethodDelete, server.URL+"/api/v1/credentials/"+ids[0].String(), "")
	require.Equal(t, 204, revoked.StatusCode)
	require.NoError(t, revoked.Body.Close())
	_, err = owner.ListTools(context.Background(), nil)
	require.Error(t, err)
}
