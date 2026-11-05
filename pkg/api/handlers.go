package api

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/api/middlewares"
	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

func Get[T, R any](get func(context.Context, uuid.UUID) (*T, error), mapper func(*T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}

		entity, err := get(r.Context(), id)
		writeResult(w, r, http.StatusOK, entity, err, mapper)
	}
}

// List parses the resource filter, then the shared page parameters.
func List[F, T, R any](parse func(*http.Request) (F, error), list func(context.Context, F, domain.PageParams) (*domain.Page[T], error), mapper func(*T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := parse(r)
		if err != nil {
			response.WriteDomainError(w, r, err)
			return
		}

		params, err := pageParams(r)
		if err != nil {
			response.WriteDomainError(w, r, err)
			return
		}

		page, err := list(r.Context(), filter, params)
		if err != nil {
			response.WriteDomainError(w, r, err)
			return
		}

		response.JSON(w, r, http.StatusOK, mapPage(page, mapper))
	}
}

func Create[Req, T, R any](create func(context.Context, Req) (*T, error), mapper func(*T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entity, err := create(r.Context(), middlewares.Body[Req](r))
		writeResult(w, r, http.StatusCreated, entity, err, mapper)
	}
}

func Update[Req, T, R any](update func(context.Context, uuid.UUID, Req) (*T, error), mapper func(*T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}

		entity, err := update(r.Context(), id, middlewares.Body[Req](r))
		writeResult(w, r, http.StatusOK, entity, err, mapper)
	}
}

func Delete(remove func(context.Context, uuid.UUID) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}

		writeNoContent(w, r, remove(r.Context(), id))
	}
}

// Action runs a command on one resource and returns the resource.
func Action[T, R any](act func(context.Context, uuid.UUID) (*T, error), mapper func(*T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}

		entity, err := act(r.Context(), id)
		writeResult(w, r, http.StatusOK, entity, err, mapper)
	}
}

// Command runs a collection command from the request body and returns no content.
func Command[Req any](run func(context.Context, Req) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeNoContent(w, r, run(r.Context(), middlewares.Body[Req](r)))
	}
}

// Read returns one value selected by parameters parsed from the request.
func Read[F, T, R any](parse func(*http.Request) (F, error), read func(context.Context, F) (*T, error), mapper func(*T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := parse(r)
		if err != nil {
			response.WriteDomainError(w, r, err)
			return
		}

		entity, err := read(r.Context(), filter)
		writeResult(w, r, http.StatusOK, entity, err, mapper)
	}
}

func writeResult[T, R any](w http.ResponseWriter, r *http.Request, status int, entity *T, err error, mapper func(*T) R) {
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	response.JSON(w, r, status, mapper(entity))
}

func writeNoContent(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := pathUUID(r, "id")
	if err != nil {
		response.WriteDomainError(w, r, err)
		return uuid.Nil, false
	}

	return id, true
}

// pathUUID reports a malformed identifier as not found, as for an unknown one.
func pathUUID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		return uuid.Nil, domain.ErrNotFound
	}

	return id, nil
}

func requireBrand(id uuid.UUID) error {
	if id == uuid.Nil {
		return domain.Validation(domain.FieldError{Field: "brandId", Code: domain.FieldInvalid})
	}

	return nil
}

func brandFilter(r *http.Request) (domain.BrandFilter, error) {
	value, err := optionalStringQuery(r, "brandId")
	if err != nil {
		return domain.BrandFilter{}, domain.Validation(domain.FieldError{Field: "brandId", Code: domain.FieldInvalid})
	}

	brandID, err := uuid.Parse(value)
	if err != nil {
		return domain.BrandFilter{}, domain.Validation(domain.FieldError{Field: "brandId", Code: domain.FieldInvalid})
	}

	if err := requireBrand(brandID); err != nil {
		return domain.BrandFilter{}, err
	}

	return domain.BrandFilter{BrandID: brandID}, nil
}

func pageParams(r *http.Request) (domain.PageParams, error) {
	page, err := positiveQueryInt(r, "page", 1, math.MaxInt)
	if err != nil {
		return domain.PageParams{}, err
	}

	pageSize, err := positiveQueryInt(r, "pageSize", 10, 100)
	if err != nil {
		return domain.PageParams{}, err
	}

	return domain.PageParams{Page: page, PageSize: pageSize}, nil
}

func optionalBoolQuery(r *http.Request, key string) (*bool, error) {
	values, present := r.URL.Query()[key]
	if !present {
		return nil, nil
	}

	if len(values) != 1 || values[0] != "true" && values[0] != "false" {
		return nil, domain.Validation(domain.FieldError{Field: key, Code: domain.FieldInvalid})
	}

	value := values[0] == "true"

	return &value, nil
}

func optionalStringQuery(r *http.Request, key string) (string, error) {
	values, present := r.URL.Query()[key]
	if !present {
		return "", nil
	}

	if len(values) != 1 {
		return "", domain.Validation(domain.FieldError{Field: key, Code: domain.FieldInvalid})
	}

	return strings.TrimSpace(values[0]), nil
}

func positiveQueryInt(r *http.Request, key string, defaultValue, maxValue int) (int, error) {
	values, present := r.URL.Query()[key]
	if !present {
		return defaultValue, nil
	}

	if len(values) != 1 {
		return 0, domain.Validation(domain.FieldError{Field: key, Code: domain.FieldInvalid})
	}

	value, err := strconv.Atoi(values[0])
	if err != nil || value < 1 || value > maxValue {
		return 0, domain.Validation(domain.FieldError{Field: key, Code: domain.FieldInvalid})
	}

	return value, nil
}

// identity is the mapper for reports whose domain JSON is already the public contract.
func identity[T any](value *T) *T {
	return value
}

func mapItems[T, R any](items []T, mapper func(*T) R) []R {
	result := make([]R, 0, len(items))
	for i := range items {
		result = append(result, mapper(&items[i]))
	}

	return result
}

func optionalResponse[T, R any](item *T, mapper func(*T) R) *R {
	if item == nil {
		return nil
	}

	result := mapper(item)

	return &result
}

func mapPage[T, R any](page *domain.Page[T], mapper func(*T) R) domain.Page[R] {
	return domain.Page[R]{
		Items:       mapItems(page.Items, mapper),
		TotalItems:  page.TotalItems,
		TotalPages:  page.TotalPages,
		CurrentPage: page.CurrentPage,
		HasNext:     page.HasNext,
		HasPrev:     page.HasPrev,
	}
}
