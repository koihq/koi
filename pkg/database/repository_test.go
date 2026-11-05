package database

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

func TestMapDBError(t *testing.T) {
	require.NoError(t, mapDBError(nil))
	require.ErrorIs(t, mapDBError(gorm.ErrRecordNotFound), domain.ErrNotFound)
	require.ErrorIs(t, mapDBError(fmt.Errorf("get: %w", gorm.ErrRecordNotFound)), domain.ErrNotFound)

	email := &pgconn.PgError{Code: pgUniqueViolation, ConstraintName: "idx_users_email"}
	require.ErrorIs(t, mapDBError(email), domain.ErrEmailTaken)
	require.ErrorIs(t, mapDBError(fmt.Errorf("create: %w", email)), domain.ErrEmailTaken)

	byColumn := &pgconn.PgError{Code: pgUniqueViolation, ColumnName: "email"}
	require.ErrorIs(t, mapDBError(byColumn), domain.ErrEmailTaken)
	competitor := &pgconn.PgError{Code: pgUniqueViolation, ConstraintName: "idx_competitors_brand_domain"}
	var validation *domain.Error
	require.ErrorAs(t, mapDBError(competitor), &validation)
	require.Equal(t, domain.CodeValidation, validation.Code)
	require.Equal(t, []domain.FieldError{
		{
			Field: "domain",
			Code:  domain.FieldTaken,
		},
	}, validation.Fields)

	other := &pgconn.PgError{Code: pgUniqueViolation, ConstraintName: "idx_memberships_user_org"}
	require.Equal(t, other, mapDBError(other))
}
