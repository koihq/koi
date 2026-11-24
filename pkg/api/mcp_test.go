package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/domain"
)

func TestMCPStreamableHTTPListsProductTools(t *testing.T) {
	handler := newMCPHandler(newServices(NewMockStore(t), ServerOptions{}), nil)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: server.URL, DisableStandaloneSSE: true}, nil)
	require.NoError(t, err)
	defer session.Close()
	require.Contains(t, session.InitializeResult().Instructions, "brands_list")
	require.Contains(t, session.InitializeResult().Instructions, "idempotencyKey")
	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	names := make(map[string]bool)
	destructive := make(map[string]bool)
	for _, tool := range tools.Tools {
		names[tool.Name] = true

		require.NotEmpty(t, tool.Annotations.Title, tool.Name)
		require.NotNil(t, tool.Annotations.OpenWorldHint, tool.Name)
		require.NotNil(t, tool.Annotations.DestructiveHint, tool.Name)

		destructive[tool.Name] = *tool.Annotations.DestructiveHint
		require.False(t, tool.Annotations.ReadOnlyHint && destructive[tool.Name], tool.Name)
	}
	require.True(t, destructive["prompts_delete"])
	require.True(t, destructive["tracking_start"])
	require.False(t, destructive["prompts_create"])
	for _, name := range []string{"brands_list", "prompts_create", "competitors_list", "generations_create", "tracking_start", "runs_results", "analysis_summary"} {
		require.True(t, names[name], name)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "prompt_visibility" {
			schema, ok := tool.InputSchema.(map[string]any)
			require.True(t, ok)
			properties, ok := schema["properties"].(map[string]any)
			require.True(t, ok)
			require.Contains(t, properties, "days")
			require.Contains(t, properties, "engines")
		}
		if tool.Name == "analysis_summary" {
			require.True(t, tool.Annotations.ReadOnlyHint)
			require.Contains(t, tool.Description, "default 30")
		}
		if tool.Name == "generations_create" {
			require.False(t, tool.Annotations.ReadOnlyHint)
			require.Contains(t, tool.Description, "generations_get")
		}
	}
	require.False(t, names["billing_checkout"])
	require.False(t, names["prompt_history_list"], "deleted prompts are no longer listed")
	for _, tool := range tools.Tools {
		if tool.Name == "prompts_list" {
			properties := tool.InputSchema.(map[string]any)["properties"].(map[string]any)
			require.Contains(t, properties, "brandId")
			require.Contains(t, properties, "confirmed")
			require.Contains(t, properties, "inactive")
		}
		if tool.Name == "prompts_delete" {
			require.Contains(t, tool.Description, "no saved answers")
		}
	}
}

func TestMCPAgentsCannotChangeWhatTheUserPays(t *testing.T) {
	handler := newMCPHandler(newServices(NewMockStore(t), ServerOptions{}), &BillingHandler{})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: server.URL, DisableStandaloneSSE: true}, nil)
	require.NoError(t, err)
	defer session.Close()

	instructions := session.InitializeResult().Instructions
	require.NotContains(t, instructions, "billing_change_allowance")
	require.NotContains(t, instructions, "checkout")
	require.Contains(t, instructions, "Settings > Billing")

	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}

	require.True(t, names["billing_subscription"])

	for _, name := range []string{
		"billing_catalog",
		"billing_quote",
		"billing_checkout",
		"billing_confirm_checkout",
		"billing_change_allowance",
		"billing_portal",
	} {
		require.False(t, names[name], name)
	}

	plan, err := json.Marshal(mcpPlanToRes(&domain.BillingStatus{
		Subscription: &domain.BillingSubscription{
			Status:     "past_due",
			Quantity:   15,
			PaymentURL: "https://invoice.stripe.com/pay",
		},
		UsedPrompts: 4,
	}))
	require.NoError(t, err)
	require.JSONEq(t, `{"status":"past_due","firstLook":false,"promptLimit":0,"usedPrompts":4}`, string(plan))
}
