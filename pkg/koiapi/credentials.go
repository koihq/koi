package koiapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

type credentialResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	ClientID  string     `json:"clientId,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

func credentialToResponse(c *koidomain.Credential) credentialResponse {
	return credentialResponse{ID: c.ID, Name: c.Name, ClientID: c.ClientID, CreatedAt: c.CreatedAt, ExpiresAt: c.ExpiresAt}
}

func credentialRoutes(service koidomain.CredentialService) http.Handler {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		page, err := pageParams(req)
		if err != nil {
			apiresponse.WriteDomainError(w, req, err)
			return
		}
		items, err := service.List(req.Context(), page)
		if err != nil {
			apiresponse.WriteDomainError(w, req, err)
			return
		}
		apiresponse.JSON(w, req, http.StatusOK, mapPage(items, credentialToResponse))
	})
	r.Delete("/{id}", func(w http.ResponseWriter, req *http.Request) {
		id, err := uuid.Parse(chi.URLParam(req, "id"))
		if err != nil {
			apiresponse.WriteDomainError(w, req, koidomain.ErrNotFound)
			return
		}
		err = service.Revoke(req.Context(), id)
		if err != nil {
			apiresponse.WriteDomainError(w, req, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return r
}

func bearerIdentity(store koidomain.Store, credentials koidomain.CredentialService, next http.Handler) http.Handler {
	verifier := func(ctx context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
		credential, err := credentials.Resolve(ctx, raw)
		if err != nil {
			return nil, err
		}
		if credential == nil || credential.Kind != "oauth_access" || credential.Audience != oauthOrigin()+"/mcp" {
			return nil, auth.ErrInvalidToken
		}
		scopes := strings.Fields(credential.Scope)
		info := &auth.TokenInfo{UserID: credential.UserID.String(), Scopes: scopes, Expiration: time.Time{}}
		if credential.ExpiresAt != nil {
			info.Expiration = *credential.ExpiresAt
		}
		return info, nil
	}
	withIdentity := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := auth.TokenInfoFromContext(r.Context())
		id, err := uuid.Parse(info.UserID)
		if err != nil {
			apiresponse.WriteDomainError(w, r, koidomain.ErrUnauthorized)
			return
		}
		identity, err := koidomain.ResolveIdentity(r.Context(), store, id)
		if err != nil {
			apiresponse.WriteDomainError(w, r, err)
			return
		}
		ctx := koiauth.WithIdentity(r.Context(), identity)
		ctx = context.WithValue(ctx, mcpScopeKey{}, strings.Join(info.Scopes, " "))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
	return auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{ResourceMetadataURL: oauthOrigin() + "/.well-known/oauth-protected-resource/mcp", AllowMissingExpiration: true})(withIdentity)
}
