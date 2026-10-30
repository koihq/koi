package koidatabase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/koidomain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormCredentialRepository struct{ db *gorm.DB }

func NewGormCredentialRepository(db *gorm.DB) *GormCredentialRepository {
	return &GormCredentialRepository{db: db}
}

func (s *GormCredentialRepository) CreateCredential(ctx context.Context, credential *koidomain.Credential) error {
	return s.db.WithContext(ctx).Create(credential).Error
}

func (s *GormCredentialRepository) CredentialByToken(ctx context.Context, raw string) (*koidomain.Credential, error) {
	var credential koidomain.Credential
	err := s.db.WithContext(ctx).Where("token_hash = ?", koidomain.HashCredential(raw)).First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if credential.ExpiresAt != nil && !credential.ExpiresAt.After(time.Now()) {
		return nil, nil
	}
	return &credential, nil
}

func (s *GormCredentialRepository) ListCredentials(ctx context.Context, userID uuid.UUID, page koidomain.PageParams) (*koidomain.Page[koidomain.Credential], error) {
	query := s.db.WithContext(ctx).Model(&koidomain.Credential{}).Where("user_id = ? AND kind = ? AND used_at IS NULL AND expires_at > ?", userID, "oauth_refresh", time.Now())
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []koidomain.Credential
	if err := query.Order("created_at desc").Offset((page.Page - 1) * page.PageSize).Limit(page.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}
	if items == nil {
		items = []koidomain.Credential{}
	}
	totalPages := (total + int64(page.PageSize) - 1) / int64(page.PageSize)
	return &koidomain.Page[koidomain.Credential]{Items: items, TotalItems: total, TotalPages: totalPages, CurrentPage: page.Page, HasNext: int64(page.Page) < totalPages, HasPrev: page.Page > 1}, nil
}

func (s *GormCredentialRepository) RevokeCredential(ctx context.Context, userID, id uuid.UUID) (bool, error) {
	deleted := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var credential koidomain.Credential
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND id = ?", userID, id).First(&credential).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if credential.Kind != "oauth_refresh" {
			return nil
		}
		result := tx.Where("user_id = ? AND connection_id = ?", userID, credential.ConnectionID).Delete(&koidomain.Credential{})
		deleted = result.RowsAffected > 0
		return result.Error
	})
	return deleted, err
}

func (s *GormCredentialRepository) CreateOAuthClient(ctx context.Context, c *koidomain.OAuthClient) error {
	return s.db.WithContext(ctx).Create(c).Error
}
func (s *GormCredentialRepository) OAuthClient(ctx context.Context, id string) (*koidomain.OAuthClient, error) {
	var c koidomain.OAuthClient
	err := s.db.WithContext(ctx).First(&c, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}
func (s *GormCredentialRepository) CreateOAuthCode(ctx context.Context, c *koidomain.OAuthCode) error {
	return s.db.WithContext(ctx).Create(c).Error
}
func (s *GormCredentialRepository) ConsumeOAuthCode(ctx context.Context, raw string) (*koidomain.OAuthCode, error) {
	var code koidomain.OAuthCode
	result := s.db.WithContext(ctx).Clauses(clause.Returning{}).Where("hash = ?", koidomain.HashCredential(raw)).Delete(&code)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &code, nil
}
func (s *GormCredentialRepository) ConsumeRefreshToken(ctx context.Context, raw string) (*koidomain.Credential, error) {
	var credential koidomain.Credential
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&credential).Clauses(clause.Returning{}).Where("token_hash = ? AND kind = ? AND used_at IS NULL AND expires_at > ?", koidomain.HashCredential(raw), "oauth_refresh", now).Update("used_at", now)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &credential, nil
}
func (s *GormCredentialRepository) RevokeToken(ctx context.Context, raw string) error {
	return s.db.WithContext(ctx).Where("token_hash = ?", koidomain.HashCredential(raw)).Delete(&koidomain.Credential{}).Error
}

func (s *GormCredentialRepository) CleanupExpired(ctx context.Context, now time.Time) error {
	if err := s.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&koidomain.OAuthCode{}).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&koidomain.Credential{}).Error
}
