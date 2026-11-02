package koiapi

import (
	"net/http/httptest"
	"testing"

	"github.com/koihq/koi/pkg/koidomain"
	"github.com/stretchr/testify/require"
)

func TestVisibilityScopeMatchesHTTPAndMCP(t *testing.T) {
	httpScope := httptest.NewRequest("GET", "/api/v1/visibility/summary?days=7&engines=gemini,gemini,chatgpt", nil)
	httpDays, httpEngines, err := visibilityScope(httpScope)
	require.NoError(t, err)
	mcpDays, mcpEngines, err := visibilityScopeValues(7, []koidomain.EngineID{
		koidomain.EngineGemini,
		koidomain.EngineGemini,
		koidomain.EngineChatGPT,
	})
	require.NoError(t, err)
	require.Equal(t, httpDays, mcpDays)
	require.Equal(t, httpEngines, mcpEngines)

	_, _, err = visibilityScopeValues(30, []koidomain.EngineID{"unknown"})
	require.ErrorIs(t, err, koidomain.ErrValidation)
	_, _, err = visibilityScopeValues(8, nil)
	require.ErrorIs(t, err, koidomain.ErrValidation)
	_, _, err = visibilityScope(httptest.NewRequest("GET", "/?days=0", nil))
	require.ErrorIs(t, err, koidomain.ErrValidation)
}

func TestMCPPageParamsMatchHTTPDefaultsAndFieldErrors(t *testing.T) {
	page, err := mcpPageParams(0, 0)
	require.NoError(t, err)
	require.Equal(t, 1, page.Page)
	require.Equal(t, 10, page.PageSize)

	_, err = mcpPageParams(1, 101)
	var domainErr *koidomain.Error
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, "pageSize", domainErr.Fields[0].Field)
}
