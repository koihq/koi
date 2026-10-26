package koiapi

import (
	"net/http"

	"github.com/google/uuid"

	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func authenticatedUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	identity, ok := koiauth.FromContext(r.Context())
	if !ok {
		apiresponse.WriteDomainError(w, r, koidomain.ErrUnauthorized)

		return uuid.Nil, false
	}

	return identity.UserID, true
}
