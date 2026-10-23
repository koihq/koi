package response

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"github.com/koihq/koi/pkg/koidomain"
)

type Envelope struct {
	Error Body `json:"error"`
}

type Body struct {
	RequestID string                 `json:"requestId,omitempty"`
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Fields    []koidomain.FieldError `json:"fields,omitempty"`
	StatusURL string                 `json:"statusUrl,omitempty"`
}

func JSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	if env, ok := v.(Envelope); ok {
		env.Error.RequestID = middleware.GetReqID(r.Context())
		v = env
	}
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

	return Envelope{
		Error: Body{
			Code:    e.Code,
			Message: e.Message,
			Fields:  publicFields(e.Fields),
		},
	}
}

func publicFields(fields []koidomain.FieldError) []koidomain.FieldError {
	if fields == nil {
		return nil
	}
	result := make([]koidomain.FieldError, len(fields))
	for i, field := range fields {
		parts := strings.Split(field.Field, "_")
		for j := 1; j < len(parts); j++ {
			if parts[j] != "" {
				parts[j] = strings.ToUpper(parts[j][:1]) + parts[j][1:]
			}
		}
		result[i] = koidomain.FieldError{Field: strings.Join(parts, ""), Code: field.Code}
	}
	return result
}
