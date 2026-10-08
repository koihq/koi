package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koidomain"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
		fields []koidomain.FieldError
	}{
		{
			name: "validation",
			err: koidomain.Validation(koidomain.FieldError{
				Field: "email",
				Code:  koidomain.FieldInvalid,
			}),
			status: http.StatusBadRequest,
			code:   koidomain.CodeValidation,
			fields: []koidomain.FieldError{
				{
					Field: "email",
					Code:  koidomain.FieldInvalid,
				},
			},
		},
		{
			name:   "invalid_json",
			err:    koidomain.ErrInvalidJSON,
			status: http.StatusBadRequest,
			code:   koidomain.CodeInvalidJSON,
		},
		{
			name:   "email_taken",
			err:    koidomain.ErrEmailTaken,
			status: http.StatusConflict,
			code:   koidomain.CodeEmailTaken,
			fields: []koidomain.FieldError{
				{
					Field: "email",
					Code:  koidomain.FieldTaken,
				},
			},
		},
		{
			name:   "brand_limit_reached",
			err:    koidomain.ErrBrandLimitReached,
			status: http.StatusConflict,
			code:   koidomain.CodeBrandLimitReached,
		},
		{
			name:   "invalid_credentials",
			err:    koidomain.ErrInvalidCredentials,
			status: http.StatusUnauthorized,
			code:   koidomain.CodeInvalidCredentials,
		},
		{
			name:   "unauthorized",
			err:    koidomain.ErrUnauthorized,
			status: http.StatusUnauthorized,
			code:   koidomain.CodeUnauthorized,
		},
		{
			name:   "not_found",
			err:    koidomain.ErrNotFound,
			status: http.StatusNotFound,
			code:   koidomain.CodeNotFound,
		},
		{
			name:   "feature_unavailable",
			err:    koidomain.ErrFeatureUnavailable,
			status: http.StatusServiceUnavailable,
			code:   koidomain.CodeFeatureUnavailable,
		},
		{
			name:   "internal",
			err:    errors.New("boom"),
			status: http.StatusInternalServerError,
			code:   koidomain.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, env := MapError(tt.err)
			require.Equal(t, tt.status, status)
			require.Equal(t, tt.code, env.Error.Code)
			require.Equal(t, tt.fields, env.Error.Fields)
			if tt.code == koidomain.CodeInternal {
				require.Equal(t, "internal error", env.Error.Message)
			}
		})
	}
}

func TestWriteDomainError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", nil)
	rec := httptest.NewRecorder()
	WriteDomainError(rec, req, koidomain.Validation(koidomain.FieldError{Field: "password", Code: koidomain.FieldMinLength}))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Equal(t, koidomain.CodeValidation, env.Error.Code)
	require.Equal(t, []koidomain.FieldError{
		{
			Field: "password",
			Code:  koidomain.FieldMinLength,
		},
	}, env.Error.Fields)
}
