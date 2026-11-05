package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

func authenticatedUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		response.WriteDomainError(w, r, domain.ErrUnauthorized)

		return uuid.Nil, false
	}

	return identity.UserID, true
}
