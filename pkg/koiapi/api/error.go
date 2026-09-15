package api

import (
	"net/http"

	"github.com/go-chi/render"
)

func JSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	render.Status(r, status)
	render.JSON(w, r, v)
}

func Error(w http.ResponseWriter, r *http.Request, status int, msg string) {
	JSON(w, r, status, map[string]string{"error": msg})
}
