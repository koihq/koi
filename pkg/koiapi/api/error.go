package api

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/koihq/koi/pkg/koidomain"
)

type Envelope struct {
	Error Body `json:"error"`
}

type Body struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Fields    []koidomain.FieldError `json:"fields,omitempty"`
	StatusURL string                 `json:"status_url,omitempty"`
}

func JSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	render.Status(r, status)
	render.JSON(w, r, v)
}

func Error(w http.ResponseWriter, r *http.Request, status int, e *koidomain.Error) {
	JSON(w, r, status, envelope(e))
}

func envelope(e *koidomain.Error) Envelope {
	if e == nil {
		e = koidomain.ErrInternal
	}
	return Envelope{Error: Body{Code: e.Code, Message: e.Message, Fields: e.Fields}}
}
