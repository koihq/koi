package koiapi

import (
	"net/http"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koidomain"
)

type HealthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

type HealthHandler struct {
	store koidomain.Store
}

func NewHealthHandler(store koidomain.Store) *HealthHandler {
	return &HealthHandler{store: store}
}

func (h *HealthHandler) Get(w http.ResponseWriter, r *http.Request) {
	db := "ok"
	if h.store == nil || h.store.Ping(r.Context()) != nil {
		db = "down"
	}
	api.JSON(w, r, http.StatusOK, HealthResponse{Status: "ok", DB: db})
}
