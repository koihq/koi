package koidatabase

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

func TestMapDBError(t *testing.T) {
	require.NoError(t, mapDBError(nil))
	require.ErrorIs(t, mapDBError(gorm.ErrRecordNotFound), koidomain.ErrNotFound)
	require.ErrorIs(t, mapDBError(fmt.Errorf("get: %w", gorm.ErrRecordNotFound)), koidomain.ErrNotFound)

	email := &pgconn.PgError{Code: pgUniqueViolation, ConstraintName: "idx_users_email"}
	require.ErrorIs(t, mapDBError(email), koidomain.ErrEmailTaken)
	require.ErrorIs(t, mapDBError(fmt.Errorf("create: %w", email)), koidomain.ErrEmailTaken)

	byColumn := &pgconn.PgError{Code: pgUniqueViolation, ColumnName: "email"}
	require.ErrorIs(t, mapDBError(byColumn), koidomain.ErrEmailTaken)
	competitor := &pgconn.PgError{Code: pgUniqueViolation, ConstraintName: "idx_competitors_brand_domain"}
	var validation *koidomain.Error
	require.ErrorAs(t, mapDBError(competitor), &validation)
	require.Equal(t, koidomain.CodeValidation, validation.Code)
	require.Equal(t, []koidomain.FieldError{
		{
			Field: "domain",
			Code:  koidomain.FieldTaken,
		},
	}, validation.Fields)

	other := &pgconn.PgError{Code: pgUniqueViolation, ConstraintName: "idx_memberships_user_org"}
	require.Equal(t, other, mapDBError(other))
}
