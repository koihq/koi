package domain

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/auth"
)

type IssuedTokens struct {
	AccessToken  string
	RefreshToken string
	Scope        string
	ExpiresIn    int
}

type CredentialService interface {
	List(context.Context, PageParams) (*Page[Credential], error)
	Revoke(context.Context, uuid.UUID) error
	Resolve(context.Context, string) (*Credential, error)
	RegisterClient(context.Context, *OAuthClient) error
	Client(context.Context, string) (*OAuthClient, error)
	CreateCode(context.Context, *OAuthCode) error
	ConsumeCode(context.Context, string) (*OAuthCode, error)
	IssueTokens(context.Context, uuid.UUID, string, string, string, string, uuid.UUID) (IssuedTokens, error)
	RefreshTokens(context.Context, string, string, string) (IssuedTokens, error)
	RevokeToken(context.Context, string) error
}

type credentialService struct{ store Store }

func NewCredentialService(store Store) CredentialService { return &credentialService{store: store} }
func RandomCredential() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "koi_" + base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}
func credentialIdentity(ctx context.Context) (auth.Identity, error) {
	id, ok := auth.FromContext(ctx)
	if !ok || id.OrganizationID == uuid.Nil {
		return auth.Identity{}, ErrUnauthorized
	}
	return id, nil
}
func (s *credentialService) List(ctx context.Context, page PageParams) (*Page[Credential], error) {
	id, err := credentialIdentity(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.CredentialRepo().ListCredentials(ctx, id.UserID, page)
}
func (s *credentialService) Revoke(ctx context.Context, id uuid.UUID) error {
	identity, err := credentialIdentity(ctx)
	if err != nil {
		return err
	}
	deleted, err := s.store.CredentialRepo().RevokeCredential(ctx, identity.UserID, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}
func (s *credentialService) Resolve(ctx context.Context, raw string) (*Credential, error) {
	return s.store.CredentialRepo().CredentialByToken(ctx, raw)
}
func (s *credentialService) RegisterClient(ctx context.Context, c *OAuthClient) error {
	return s.store.CredentialRepo().CreateOAuthClient(ctx, c)
}
func (s *credentialService) Client(ctx context.Context, id string) (*OAuthClient, error) {
	return s.store.CredentialRepo().OAuthClient(ctx, id)
}
func (s *credentialService) CreateCode(ctx context.Context, c *OAuthCode) error {
	return s.store.CredentialRepo().CreateOAuthCode(ctx, c)
}
func (s *credentialService) ConsumeCode(ctx context.Context, raw string) (*OAuthCode, error) {
	return s.store.CredentialRepo().ConsumeOAuthCode(ctx, raw)
}
func issueTokens(ctx context.Context, repo CredentialRepository, userID uuid.UUID, clientID, name, scope, audience string, connectionID uuid.UUID) (IssuedTokens, error) {
	if connectionID == uuid.Nil {
		connectionID = uuid.New()
	}
	access, err := RandomCredential()
	if err != nil {
		return IssuedTokens{}, err
	}
	refresh, err := RandomCredential()
	if err != nil {
		return IssuedTokens{}, err
	}
	expires := time.Now().Add(time.Hour)
	refreshExpires := time.Now().Add(30 * 24 * time.Hour)
	a := &Credential{UserID: userID, TokenHash: HashCredential(access), Kind: "oauth_access", Name: name, ClientID: clientID, ConnectionID: connectionID, Audience: audience, Scope: scope, ExpiresAt: &expires}
	f := &Credential{UserID: userID, TokenHash: HashCredential(refresh), Kind: "oauth_refresh", Name: name, ClientID: clientID, ConnectionID: connectionID, Audience: audience, Scope: scope, ExpiresAt: &refreshExpires}
	if err := repo.CreateCredential(ctx, a); err != nil {
		return IssuedTokens{}, err
	}
	if err := repo.CreateCredential(ctx, f); err != nil {
		return IssuedTokens{}, err
	}
	return IssuedTokens{AccessToken: access, RefreshToken: refresh, Scope: scope, ExpiresIn: 3600}, nil
}
func (s *credentialService) IssueTokens(ctx context.Context, userID uuid.UUID, clientID, name, scope, audience string, connectionID uuid.UUID) (IssuedTokens, error) {
	var tokens IssuedTokens
	err := s.store.Atomic(ctx, func(tx Store) error {
		var e error
		tokens, e = issueTokens(ctx, tx.CredentialRepo(), userID, clientID, name, scope, audience, connectionID)
		return e
	})
	return tokens, err
}

// refreshReuseGrace spares a client that retries after losing the response to its first refresh.
const refreshReuseGrace = 10 * time.Second

func (s *credentialService) RefreshTokens(ctx context.Context, raw, clientID, audience string) (IssuedTokens, error) {
	var tokens IssuedTokens
	var replayed *Credential

	err := s.store.Atomic(ctx, func(tx Store) error {
		old, e := tx.CredentialRepo().ConsumeRefreshToken(ctx, raw)
		if e != nil {
			return e
		}

		if old == nil {
			replayed, e = tx.CredentialRepo().UsedRefreshToken(ctx, raw)
			if e != nil {
				return e
			}

			return ErrInvalidCredentials
		}

		if old.ClientID != clientID || old.Audience != audience {
			return ErrInvalidCredentials
		}

		tokens, e = issueTokens(ctx, tx.CredentialRepo(), old.UserID, clientID, old.Name, old.Scope, audience, old.ConnectionID)

		return e
	})

	// This runs after the failed transaction, whose rollback would otherwise undo the revocation.
	if replayed != nil && replayed.UsedAt != nil && time.Since(*replayed.UsedAt) > refreshReuseGrace {
		if e := s.store.CredentialRepo().RevokeConnection(ctx, replayed.UserID, replayed.ConnectionID); e != nil {
			return tokens, errors.Join(err, e)
		}
	}

	return tokens, err
}
func (s *credentialService) RevokeToken(ctx context.Context, raw string) error {
	return s.store.CredentialRepo().RevokeToken(ctx, raw)
}
