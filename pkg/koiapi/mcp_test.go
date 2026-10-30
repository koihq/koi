package koiapi

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestMCPStreamableHTTPListsProductTools(t *testing.T) {
	handler := newMCPHandler(NewMockStore(t), nil, false, nil)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: server.URL, DisableStandaloneSSE: true}, nil)
	require.NoError(t, err)
	defer session.Close()
	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	names := make(map[string]bool)
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	for _, name := range []string{"brands_list", "prompts_create", "competitors_list", "generations_create", "runs_results", "analysis_summary"} {
		require.True(t, names[name], name)
	}
	require.False(t, names["billing_checkout"])
}
