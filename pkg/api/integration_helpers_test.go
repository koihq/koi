package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/database"
)

// integrationServer is the real API over a migrated Postgres schema. Its clients validate every
// documented response against docs/openapi.yaml.
type integrationServer struct {
	*httptest.Server
	DB    *gorm.DB
	Store *database.Store
}

func newIntegrationServer(t *testing.T) integrationServer {
	t.Helper()

	db, err := database.Open(testdb.URL(t))
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	require.NoError(t, database.Migrate(db))

	store := database.NewStore(db)
	server := httptest.NewServer(NewServer(store, scs.New(), ServerOptions{OpenSignup: true}))
	t.Cleanup(server.Close)

	return integrationServer{
		Server: server,
		DB:     db,
		Store:  store,
	}
}

// signup registers a new account and returns the client holding its session cookie.
func (s integrationServer) signup(t *testing.T) (*http.Client, AuthRes) {
	t.Helper()

	client := newCookieClient(t)
	body := `{"email":"` + uuid.NewString() + `@example.com","password":"password-1234"}`

	resp := doJSON(t, client, http.MethodPost, s.URL+"/api/v1/auth/signup", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var account AuthRes
	readJSON(t, resp, &account)

	return client, account
}

// readBody reads and closes a response body.
func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(body)
}
