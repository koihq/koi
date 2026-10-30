package koidomain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

type Credential struct {
	BaseEntity
	UserID       uuid.UUID  `json:"-" gorm:"type:uuid;not null;index"`
	Name         string     `json:"name" gorm:"not null"`
	TokenHash    string     `json:"-" gorm:"uniqueIndex;not null"`
	Kind         string     `json:"kind" gorm:"not null"`
	ClientID     string     `json:"-"`
	ConnectionID uuid.UUID  `json:"-" gorm:"type:uuid;index"`
	Audience     string     `json:"-"`
	Scope        string     `json:"-"`
	ExpiresAt    *time.Time `json:"expiresAt"`
	UsedAt       *time.Time `json:"-"`
}

func (Credential) TableName() string { return "credentials" }

func HashCredential(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

type OAuthClient struct {
	ID           string    `json:"clientId" gorm:"primaryKey"`
	RedirectURIs []string  `json:"redirectUris" gorm:"serializer:json;not null"`
	Name         string    `json:"name" gorm:"not null"`
	CreatedAt    time.Time `json:"createdAt" gorm:"not null;autoCreateTime"`
}

func (OAuthClient) TableName() string { return "oauth_clients" }

type OAuthCode struct {
	Hash        string    `gorm:"primaryKey"`
	UserID      uuid.UUID `gorm:"type:uuid;not null"`
	ClientID    string    `gorm:"not null"`
	ClientName  string    `gorm:"not null"`
	RedirectURI string    `gorm:"not null"`
	Challenge   string    `gorm:"not null"`
	Scope       string    `gorm:"not null"`
	ExpiresAt   time.Time `gorm:"not null"`
}

func (OAuthCode) TableName() string { return "oauth_codes" }

type CredentialRepository interface {
	CreateCredential(context.Context, *Credential) error
	CredentialByToken(context.Context, string) (*Credential, error)
	ListCredentials(context.Context, uuid.UUID, PageParams) (*Page[Credential], error)
	RevokeCredential(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	CreateOAuthClient(context.Context, *OAuthClient) error
	OAuthClient(context.Context, string) (*OAuthClient, error)
	CreateOAuthCode(context.Context, *OAuthCode) error
	ConsumeOAuthCode(context.Context, string) (*OAuthCode, error)
	ConsumeRefreshToken(context.Context, string) (*Credential, error)
	RevokeToken(context.Context, string) error
	CleanupExpired(context.Context, time.Time) error
}
