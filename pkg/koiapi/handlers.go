package koiapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/middlewares"
	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koidomain"
)

func Get[T, R any](parameter string, get func(context.Context, uuid.UUID) (*T, error), mapper func(*T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r, parameter)
		if !ok {
			return
		}

		entity, err := get(r.Context(), id)
		writeResult(w, r, http.StatusOK, entity, err, mapper)
	}
}

func List[T, R, F any](parse func(*http.Request) (F, error), list func(context.Context, F) (*koidomain.Page[T], error), mapper func(*T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := parse(r)
		if err != nil {
			apiresponse.WriteDomainError(w, r, err)
			return
		}

		page, err := list(r.Context(), filter)
		if err != nil {
			apiresponse.WriteDomainError(w, r, err)
			return
		}

		apiresponse.JSON(w, r, http.StatusOK, mapPage(page, mapper))
	}
}

func Create[Req, T, R any](create func(context.Context, Req) (*T, error), mapper func(*T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entity, err := create(r.Context(), middlewares.Body[Req](r))
		writeResult(w, r, http.StatusCreated, entity, err, mapper)
	}
}

func Update[Req, T, R any](parameter string, update func(context.Context, uuid.UUID, Req) (*T, error), mapper func(*T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r, parameter)
		if !ok {
			return
		}

		entity, err := update(r.Context(), id, middlewares.Body[Req](r))
		writeResult(w, r, http.StatusOK, entity, err, mapper)
	}
}

func Delete(parameter string, remove func(context.Context, uuid.UUID) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r, parameter)
		if !ok {
			return
		}

		if err := remove(r.Context(), id); err != nil {
			apiresponse.WriteDomainError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func writeResult[T, R any](w http.ResponseWriter, r *http.Request, status int, entity *T, err error, mapper func(*T) R) {
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}

	apiresponse.JSON(w, r, status, mapper(entity))
}
