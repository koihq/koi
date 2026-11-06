package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

type credentialResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	ClientID  string     `json:"clientId,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

func credentialToRes(c *domain.Credential) credentialResponse {
	return credentialResponse{ID: c.ID, Name: c.Name, ClientID: c.ClientID, CreatedAt: c.CreatedAt, ExpiresAt: c.ExpiresAt}
}

func credentialRoutes(service domain.CredentialService) http.Handler {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		page, err := pageParams(req)
		if err != nil {
			response.WriteDomainError(w, req, err)
			return
		}
		items, err := service.List(req.Context(), page)
		if err != nil {
			response.WriteDomainError(w, req, err)
			return
		}
		response.JSON(w, req, http.StatusOK, mapPage(items, credentialToRes))
	})
	r.Delete("/{id}", Delete(service.Revoke))
	return r
}

func bearerIdentity(store domain.Store, credentials domain.CredentialService, next http.Handler) http.Handler {
	verifier := func(ctx context.Context, raw string, _ *http.Request) (*mcpauth.TokenInfo, error) {
		credential, err := credentials.Resolve(ctx, raw)
		if err != nil {
			return nil, err
		}
		if credential == nil || credential.Kind != "oauth_access" || credential.Audience != oauthOrigin()+"/mcp" {
			return nil, mcpauth.ErrInvalidToken
		}
		scopes := strings.Fields(credential.Scope)
		info := &mcpauth.TokenInfo{UserID: credential.UserID.String(), Scopes: scopes, Expiration: time.Time{}}
		if credential.ExpiresAt != nil {
			info.Expiration = *credential.ExpiresAt
		}
		return info, nil
	}
	withIdentity := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := mcpauth.TokenInfoFromContext(r.Context())
		id, err := uuid.Parse(info.UserID)
		if err != nil {
			response.WriteDomainError(w, r, domain.ErrUnauthorized)
			return
		}
		identity, err := domain.ResolveIdentity(r.Context(), store, id)
		if err != nil {
			response.WriteDomainError(w, r, err)
			return
		}
		ctx := auth.WithIdentity(r.Context(), identity)
		ctx = context.WithValue(ctx, mcpScopeKey{}, strings.Join(info.Scopes, " "))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
	return mcpauth.RequireBearerToken(verifier, &mcpauth.RequireBearerTokenOptions{ResourceMetadataURL: oauthOrigin() + "/.well-known/oauth-protected-resource/mcp", AllowMissingExpiration: true})(withIdentity)
}
