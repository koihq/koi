package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/domain"
)

type GormPasswordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(db *gorm.DB) *GormPasswordResetRepository {
	return &GormPasswordResetRepository{db: db}
}

func (r *GormPasswordResetRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	return mapDBError(r.db.WithContext(ctx).Create(token).Error)
}

func (r *GormPasswordResetRepository) DeleteForUser(ctx context.Context, userID uuid.UUID) error {
	return mapDBError(r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&domain.PasswordResetToken{}).Error)
}

// Consume deletes in one statement, so two requests with the same link cannot both succeed.
func (r *GormPasswordResetRepository) Consume(ctx context.Context, tokenHash string, now time.Time) (uuid.UUID, error) {
	var deleted []domain.PasswordResetToken

	result := r.db.WithContext(ctx).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "user_id"}}}).
		Where("token_hash = ? AND expires_at > ?", tokenHash, now).
		Delete(&deleted)
	if result.Error != nil {
		return uuid.Nil, mapDBError(result.Error)
	}

	if len(deleted) == 0 {
		return uuid.Nil, domain.ErrPasswordResetInvalid
	}

	return deleted[0].UserID, nil
}
