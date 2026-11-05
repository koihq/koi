package api

import (
	"net/http"

	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

type HealthRes struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

type HealthHandler struct {
	store domain.Store
}

func NewHealthHandler(store domain.Store) *HealthHandler {
	return &HealthHandler{store: store}
}

func (h *HealthHandler) Get(w http.ResponseWriter, r *http.Request) {
	db := "ok"
	if h.store == nil || h.store.Ping(r.Context()) != nil {
		db = "down"
	}

	response.JSON(w, r, http.StatusOK, HealthRes{Status: "ok", DB: db})
}
