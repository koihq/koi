package koiapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koidomain"
)

func newBrandTestServer(t *testing.T, q koidomain.BrandQuerier, c koidomain.BrandCommander) *httptest.Server {
	t.Helper()
	sess := scs.New()
	sess.Cookie.Name = "koi_session"
	r := chi.NewRouter()
	r.Use(sess.LoadAndSave)
	r.Get("/test-session/{id}", func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", chi.URLParam(r, "id"))
		w.WriteHeader(http.StatusNoContent)
	})
	r.Route("/api/v1/brands", NewBrandHandler(q, c, sess).Routes())
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts
}

func brandSession(t *testing.T, client *http.Client, serverURL string, userID uuid.UUID) {
	t.Helper()
	resp, err := client.Get(serverURL + "/test-session/" + userID.String())
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestBrandHandlerRequiresSession(t *testing.T) {
	ts := newBrandTestServer(t, NewMockBrandQuerier(t), NewMockBrandCommander(t))
	client := newCookieClient(t)
	id := uuid.New().String()
	for _, path := range []string{"/brands", "/brands/countries", "/brands/" + id} {
		resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1"+path, "")
		var body api.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.Equal(t, koidomain.CodeUnauthorized, body.Error.Code)
	}
	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		path := "/brands"
		if method == http.MethodPatch {
			path += "/" + id
		}
		resp := doJSON(t, client, method, ts.URL+"/api/v1"+path, `{}`)
		var body api.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.Equal(t, koidomain.CodeUnauthorized, body.Error.Code)
	}
}

func TestBrandHandlerRejectsMalformedSessionAndPagination(t *testing.T) {
	userID := uuid.New()
	q := NewMockBrandQuerier(t)
	q.EXPECT().List(mock.Anything, userID, 3, 100).Return(&koidomain.BrandPage{
		Items: []koidomain.Brand{}, CurrentPage: 3, HasPrev: true,
	}, nil)
	ts := newBrandTestServer(t, q, NewMockBrandCommander(t))
	client := newCookieClient(t)
	resp, err := client.Get(ts.URL + "/test-session/invalid")
	require.NoError(t, err)
	resp.Body.Close()
	resp = doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/brands", "")
	var denied api.Envelope
	readJSON(t, resp, &denied)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, koidomain.CodeUnauthorized, denied.Error.Code)

	brandSession(t, client, ts.URL, userID)
	for _, query := range []struct{ value, field string }{
		{"page=0", "page"}, {"page=-1", "page"}, {"page=abc", "page"},
		{"page=", "page"}, {"page=1&page=2", "page"},
		{"pageSize=0", "pageSize"}, {"pageSize=101", "pageSize"},
		{"pageSize=abc", "pageSize"}, {"pageSize=", "pageSize"},
	} {
		resp = doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/brands?"+query.value, "")
		var body api.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, query.value)
		require.Equal(t, koidomain.CodeValidation, body.Error.Code)
		require.Equal(t, []koidomain.FieldError{{Field: query.field, Code: koidomain.FieldInvalid}}, body.Error.Fields)
	}

	resp = doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/brands?page=3&pageSize=100", "")
	var page koidomain.BrandPage
	readJSON(t, resp, &page)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 3, page.CurrentPage)
}

func TestBrandHandlerOwnershipAndLimit(t *testing.T) {
	userA, userB, brandID := uuid.New(), uuid.New(), uuid.New()
	brand := &koidomain.Brand{
		BaseEntity: koidomain.BaseEntity{ID: brandID}, UserID: userA,
		Name: "Koi", Domain: "koi.dev", Country: "IT",
	}
	q := NewMockBrandQuerier(t)
	c := NewMockBrandCommander(t)
	q.EXPECT().List(mock.Anything, userA, 1, 10).Return(&koidomain.BrandPage{
		Items: []koidomain.Brand{}, CurrentPage: 1,
	}, nil).Once()
	q.EXPECT().List(mock.Anything, userA, 1, 10).Return(&koidomain.BrandPage{
		Items: []koidomain.Brand{*brand}, TotalItems: 1, TotalPages: 1, CurrentPage: 1,
	}, nil).Once()
	q.EXPECT().List(mock.Anything, userB, 1, 10).Return(&koidomain.BrandPage{
		Items: []koidomain.Brand{}, CurrentPage: 1,
	}, nil)
	q.EXPECT().Get(mock.Anything, userA, brandID).Return(brand, nil)
	q.EXPECT().Get(mock.Anything, userB, brandID).Return(nil, koidomain.ErrNotFound)
	c.EXPECT().Create(mock.Anything, userA, mock.MatchedBy(func(p koidomain.BrandParams) bool {
		return p.Name == "Koi" && p.Domain == "koi.dev" && p.Country == "IT"
	})).Return(brand, nil)
	c.EXPECT().Create(mock.Anything, userA, mock.Anything).Return(nil, koidomain.ErrBrandLimitReached)
	c.EXPECT().Update(mock.Anything, userB, brandID, mock.Anything).Return(nil, koidomain.ErrNotFound)
	c.EXPECT().Update(mock.Anything, userA, brandID, mock.MatchedBy(func(p koidomain.BrandPatch) bool {
		return p.Description != nil && *p.Description == "AI visibility"
	})).Return(brand, nil)

	ts := newBrandTestServer(t, q, c)
	clientA, clientB := newCookieClient(t), newCookieClient(t)
	brandSession(t, clientA, ts.URL, userA)
	brandSession(t, clientB, ts.URL, userB)
	base := ts.URL + "/api/v1/brands"

	resp := doJSON(t, clientA, http.MethodGet, base, "")
	var empty koidomain.BrandPage
	readJSON(t, resp, &empty)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, empty.Items)
	require.Empty(t, empty.Items)
	require.Equal(t, 1, empty.CurrentPage)

	resp = doJSON(t, clientA, http.MethodGet, base+"/countries", "")
	var countries []string
	readJSON(t, resp, &countries)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, countries, "IT")
	require.NotContains(t, countries, "ZZ")

	resp = doJSON(t, clientA, http.MethodPost, base,
		`{"user_id":"00000000-0000-0000-0000-000000000000","name":"Koi","domain":"koi.dev","country":"IT"}`)
	var created koidomain.Brand
	readJSON(t, resp, &created)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Equal(t, userA, created.UserID)

	resp = doJSON(t, clientA, http.MethodGet, base, "")
	var listA koidomain.BrandPage
	readJSON(t, resp, &listA)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, listA.Items, 1)
	require.Equal(t, userA, listA.Items[0].UserID)
	require.EqualValues(t, 1, listA.TotalItems)

	resp = doJSON(t, clientB, http.MethodGet, base, "")
	var listB koidomain.BrandPage
	readJSON(t, resp, &listB)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Empty(t, listB.Items)
	require.Zero(t, listB.TotalItems)

	resp = doJSON(t, clientA, http.MethodGet, base+"/"+brandID.String(), "")
	var own koidomain.Brand
	readJSON(t, resp, &own)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, brandID, own.ID)

	resp = doJSON(t, clientB, http.MethodGet, base+"/"+brandID.String(), "")
	var denied api.Envelope
	readJSON(t, resp, &denied)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, koidomain.CodeNotFound, denied.Error.Code)

	resp = doJSON(t, clientB, http.MethodPatch, base+"/"+brandID.String(), `{"name":"Stolen"}`)
	readJSON(t, resp, &denied)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, koidomain.CodeNotFound, denied.Error.Code)

	resp = doJSON(t, clientA, http.MethodPatch, base+"/"+brandID.String(),
		`{"description":"AI visibility","user_id":"00000000-0000-0000-0000-000000000000"}`)
	var updated koidomain.Brand
	readJSON(t, resp, &updated)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, userA, updated.UserID)

	resp = doJSON(t, clientA, http.MethodPost, base, `{"name":"Second","domain":"second.dev","country":"IT"}`)
	var limit api.Envelope
	readJSON(t, resp, &limit)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	require.Equal(t, koidomain.CodeBrandLimitReached, limit.Error.Code)
}
