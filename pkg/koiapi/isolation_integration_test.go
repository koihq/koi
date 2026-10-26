package koiapi

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestFlatAPIIsolationAndExplicitBrandScope(t *testing.T) {
	db, err := koidatabase.Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, koidatabase.Migrate(db))
	server := httptest.NewServer(NewServer(koidatabase.NewStore(db), scs.New(), nil, false))
	t.Cleanup(server.Close)
	first, second := newCookieClient(t), newCookieClient(t)
	for _, client := range []*http.Client{first, second} {
		resp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/signup", `{"email":"`+uuid.NewString()+`@example.com","password":"password123"}`)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NoError(t, resp.Body.Close())
	}
	var brandA, brandB BrandRes
	for _, target := range []*BrandRes{&brandA, &brandB} {
		resp := doJSON(t, first, http.MethodPost, server.URL+"/api/v1/brands", `{"name":"Koi","domain":"koi.dev","country":"IT"}`)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		readJSON(t, resp, target)
	}
	var prompt PromptRes
	resp := doJSON(t, first, http.MethodPost, server.URL+"/api/v1/prompts", `{"brandId":"`+brandA.ID.String()+`","text":"First brand"}`)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	readJSON(t, resp, &prompt)
	var competitor CompetitorRes
	resp = doJSON(t, first, http.MethodPost, server.URL+"/api/v1/competitors", `{"brandId":"`+brandA.ID.String()+`","name":"Acme","domain":"acme.dev"}`)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	readJSON(t, resp, &competitor)
	resp = doJSON(t, first, http.MethodGet, server.URL+"/api/v1/visibility/summary?brandId="+brandA.ID.String(), "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var summary koidomain.VisibilitySummary
	readJSON(t, resp, &summary)
	require.True(t, summary.HasConfirmedPrompts)
	require.Len(t, summary.Competitors, 1)
	require.Equal(t, competitor.ID, summary.Competitors[0].ID)
	require.Equal(t, "Acme", summary.Competitors[0].Name)
	require.Equal(t, "acme.dev", summary.Competitors[0].Domain)
	require.Nil(t, summary.Metric.Rate)

	// The preference can change without changing the destination of an explicit request.
	resp = doJSON(t, first, http.MethodPost, server.URL+"/api/v1/brands/"+brandB.ID.String()+"/activate", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
	resp = doJSON(t, first, http.MethodPost, server.URL+"/api/v1/prompts", `{"brandId":"`+brandA.ID.String()+`","text":"Still first brand"}`)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	readJSON(t, resp, &prompt)
	require.Equal(t, brandA.ID, prompt.BrandID)
	resp = doJSON(t, first, http.MethodGet, server.URL+"/api/v1/prompts", "")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.NoError(t, resp.Body.Close())

	for _, request := range []struct{ method, path, body string }{
		{http.MethodGet, "/prompts?brandId=" + brandA.ID.String(), ""},
		{http.MethodGet, "/prompts/" + prompt.ID.String(), ""},
		{http.MethodPatch, "/prompts/" + prompt.ID.String(), `{"text":"Forbidden"}`},
		{http.MethodDelete, "/prompts/" + prompt.ID.String(), ""},
		{http.MethodPost, "/prompts/" + prompt.ID.String() + "/confirm", ""},
		{http.MethodGet, "/competitors/" + competitor.ID.String(), ""},
		{http.MethodDelete, "/competitors/" + competitor.ID.String(), ""},
		{http.MethodPost, "/competitors", `{"brandId":"` + brandA.ID.String() + `","name":"Other","domain":"other.dev"}`},
		{http.MethodGet, "/runs?brandId=" + brandA.ID.String(), ""},
		{http.MethodGet, "/generations?brandId=" + brandA.ID.String(), ""},
		{http.MethodGet, "/prompt-history?brandId=" + brandA.ID.String(), ""},
		{http.MethodGet, "/visibility/summary?brandId=" + brandA.ID.String(), ""},
		{http.MethodGet, "/tracking?brandId=" + brandA.ID.String(), ""},
	} {
		resp := doJSON(t, second, request.method, server.URL+"/api/v1"+request.path, request.body)
		require.Equal(t, http.StatusNotFound, resp.StatusCode, request.method+request.path)
		var body struct {
			Error struct {
				Code      string `json:"code"`
				RequestID string `json:"requestId"`
			} `json:"error"`
		}
		readJSON(t, resp, &body)
		require.Equal(t, "not_found", body.Error.Code)
		require.NotEmpty(t, body.Error.RequestID)
		require.Equal(t, body.Error.RequestID, resp.Header.Get("X-Request-ID"))
	}

	var membershipReads atomic.Int32
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("count_request_membership", func(tx *gorm.DB) {
		if tx.Statement.Table == "memberships" {
			membershipReads.Add(1)
		}
	}))
	resp = doJSON(t, first, http.MethodGet, server.URL+"/api/v1/auth/me", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var account MeRes
	readJSON(t, resp, &account)
	require.NotNil(t, account.ActiveBrand)
	require.Equal(t, brandB.ID, account.ActiveBrand.ID)
	require.EqualValues(t, 1, membershipReads.Load())
	require.NoError(t, db.Callback().Query().Remove("count_request_membership"))

	// No old nested route is retained as an alias.
	resp = doJSON(t, first, http.MethodGet, server.URL+"/api/v1/brands/"+brandA.ID.String()+"/prompts", "")
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}
