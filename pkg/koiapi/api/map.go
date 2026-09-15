package api

import (
	"errors"
	"net/http"

	"github.com/koihq/koi/pkg/koidomain"
)

func WriteDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, koidomain.ErrEmailTaken):
		Error(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, koidomain.ErrInvalidCredentials):
		Error(w, r, http.StatusUnauthorized, err.Error())
	case errors.Is(err, koidomain.ErrNotFound):
		Error(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, koidomain.ErrValidation):
		Error(w, r, http.StatusBadRequest, err.Error())
	default:
		Error(w, r, http.StatusInternalServerError, "internal error")
	}
}
