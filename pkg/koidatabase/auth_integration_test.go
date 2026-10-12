package koidatabase

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestConcurrentSignupKeepsOneAccount(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))
	commander := koidomain.NewAuthCommander(NewStore(db))
	params := koidomain.SignupParams{Email: uuid.NewString() + "@example.com", Password: "password123"}
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := commander.Signup(t.Context(), params)
			results <- err
		}()
	}
	first, second := <-results, <-results
	if first != nil {
		first, second = second, first
	}
	require.NoError(t, first)
	require.ErrorIs(t, second, koidomain.ErrEmailTaken)
	for _, model := range []any{&koidomain.User{}, &koidomain.Organization{}, &koidomain.Membership{}} {
		var count int64
		require.NoError(t, db.Model(model).Count(&count).Error)
		require.EqualValues(t, 1, count)
	}
}
