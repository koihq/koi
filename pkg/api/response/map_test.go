package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/domain"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
		fields []domain.FieldError
	}{
		{
			name: "validation",
			err: domain.Validation(domain.FieldError{
				Field: "email",
				Code:  domain.FieldInvalid,
			}),
			status: http.StatusBadRequest,
			code:   domain.CodeValidation,
			fields: []domain.FieldError{
				{
					Field: "email",
					Code:  domain.FieldInvalid,
				},
			},
		},
		{
			name:   "invalid_json",
			err:    domain.ErrInvalidJSON,
			status: http.StatusBadRequest,
			code:   domain.CodeInvalidJSON,
		},
		{
			name:   "email_taken",
			err:    domain.ErrEmailTaken,
			status: http.StatusConflict,
			code:   domain.CodeEmailTaken,
			fields: []domain.FieldError{
				{
					Field: "email",
					Code:  domain.FieldTaken,
				},
			},
		},
		{
			name:   "password_reset_invalid",
			err:    domain.ErrPasswordResetInvalid,
			status: http.StatusBadRequest,
			code:   domain.CodePasswordResetInvalid,
		},
		{
			name:   "brand_limit_reached",
			err:    domain.ErrBrandLimitReached,
			status: http.StatusConflict,
			code:   domain.CodeBrandLimitReached,
		},
		{
			name:   "invalid_credentials",
			err:    domain.ErrInvalidCredentials,
			status: http.StatusUnauthorized,
			code:   domain.CodeInvalidCredentials,
		},
		{
			name:   "unauthorized",
			err:    domain.ErrUnauthorized,
			status: http.StatusUnauthorized,
			code:   domain.CodeUnauthorized,
		},
		{
			name:   "not_found",
			err:    domain.ErrNotFound,
			status: http.StatusNotFound,
			code:   domain.CodeNotFound,
		},
		{
			name:   "feature_unavailable",
			err:    domain.ErrFeatureUnavailable,
			status: http.StatusServiceUnavailable,
			code:   domain.CodeFeatureUnavailable,
		},
		{
			name:   "internal",
			err:    errors.New("boom"),
			status: http.StatusInternalServerError,
			code:   domain.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, env := MapError(tt.err)
			require.Equal(t, tt.status, status)
			require.Equal(t, tt.code, env.Error.Code)
			require.Equal(t, tt.fields, env.Error.Fields)
			if tt.code == domain.CodeInternal {
				require.Equal(t, "internal error", env.Error.Message)
			}
		})
	}
}

func TestWriteDomainError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", nil)
	rec := httptest.NewRecorder()
	WriteDomainError(rec, req, domain.Validation(domain.FieldError{Field: "password", Code: domain.FieldMinLength}))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Equal(t, domain.CodeValidation, env.Error.Code)
	require.Equal(t, []domain.FieldError{
		{
			Field: "password",
			Code:  domain.FieldMinLength,
		},
	}, env.Error.Fields)
}
