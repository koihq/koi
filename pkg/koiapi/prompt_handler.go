package koiapi

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koiapi/middlewares"
	"github.com/koihq/koi/pkg/koidomain"
)

type PromptRequest struct {
	Text string `json:"text"`
}

type PromptPatchRequest struct {
	Text *string `json:"text"`
}

type PromptHandler struct {
	querier   koidomain.PromptQuerier
	commander koidomain.PromptCommander
	sess      *scs.SessionManager
}

func NewPromptHandler(q koidomain.PromptQuerier, c koidomain.PromptCommander, sess *scs.SessionManager) *PromptHandler {
	return &PromptHandler{querier: q, commander: c, sess: sess}
}

func (h *PromptHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(Authorizer(h.sess))
		r.Get("/", h.List)
		r.With(middlewares.DecodeBody[PromptRequest]()).Post("/", h.Create)
		r.Post("/confirm", h.ConfirmAll)
		r.Get("/{promptID}", h.Get)
		r.With(middlewares.DecodeBody[PromptPatchRequest]()).Patch("/{promptID}", h.Update)
		r.Delete("/{promptID}", h.Delete)
		r.Post("/{promptID}/confirm", h.Confirm)
	}
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		api.WriteDomainError(w, r, koidomain.ErrNotFound)
		return uuid.Nil, false
	}
	return id, true
}

func requestOwnerAndBrand(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	brandID, ok := pathID(w, r, "brandID")
	return userID, brandID, ok
}

func (h *PromptHandler) List(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	page, err := pageParams(r)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	confirmed, err := optionalBoolQuery(r, "confirmed")
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	query, err := optionalStringQuery(r, "q")
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	prompts, err := h.querier.List(r.Context(), brandID, page, query, confirmed)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, prompts)
}

func (h *PromptHandler) Get(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}
	prompt, err := h.querier.Get(r.Context(), brandID, promptID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, prompt)
}

func (h *PromptHandler) Create(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	prompt, err := h.commander.Create(r.Context(), brandID, koidomain.PromptParams{
		Text: middlewares.Body[PromptRequest](r).Text,
	})
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusCreated, prompt)
}

func (h *PromptHandler) Update(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}
	prompt, err := h.commander.Update(r.Context(), brandID, promptID, koidomain.PromptPatch{
		Text: middlewares.Body[PromptPatchRequest](r).Text,
	})
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, prompt)
}

func (h *PromptHandler) Delete(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}
	if err := h.commander.Delete(r.Context(), brandID, promptID); err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PromptHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}
	prompt, err := h.commander.Confirm(r.Context(), brandID, promptID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, prompt)
}

func (h *PromptHandler) ConfirmAll(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	if err := h.commander.ConfirmAll(r.Context(), brandID); err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
