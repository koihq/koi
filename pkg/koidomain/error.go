package koidomain

import "github.com/google/uuid"

const (
	CodeResourceConflict          = "resource_conflict"
	CodeValidation                = "validation"
	CodeEmailTaken                = "email_taken"
	CodeBrandLimitReached         = "brand_limit_reached"
	CodeInvalidCredentials        = "invalid_credentials"
	CodeNotFound                  = "not_found"
	CodeUnauthorized              = "unauthorized"
	CodeInvalidJSON               = "invalid_json"
	CodeInternal                  = "internal"
	CodeFeatureUnavailable        = "feature_unavailable"
	CodeGenerationConflict        = "generation_conflict"
	CodeIdempotencyConflict       = "idempotency_conflict"
	CodeInitialAlreadyCompleted   = "initial_already_completed"
	CodeRunNotReady               = "run_not_ready"
	CodePromptLimitReached        = "prompt_limit_reached"
	CodeBillingConflict           = "billing_conflict"
	CodeBillingCheckoutPending    = "billing_checkout_pending"
	CodeBillingSubscriptionExists = "billing_subscription_exists"
	CodeBillingPriceMismatch      = "billing_price_mismatch"
	CodeBillingRequired           = "billing_required"

	FieldInvalid   = "invalid"
	FieldMinLength = "min_length"
	FieldTaken     = "taken"
)

var (
	ErrResourceConflict = &Error{Code: CodeResourceConflict, Message: "resource has dependent records"}
	ErrNotFound         = &Error{Code: CodeNotFound, Message: "not found"}
	ErrEmailTaken       = &Error{
		Code:    CodeEmailTaken,
		Message: "email taken",
		Fields: []FieldError{
			{
				Field: "email",
				Code:  FieldTaken,
			},
		},
	}
	ErrBrandLimitReached       = &Error{Code: CodeBrandLimitReached, Message: "brand limit reached"}
	ErrInvalidCredentials      = &Error{Code: CodeInvalidCredentials, Message: "invalid credentials"}
	ErrValidation              = &Error{Code: CodeValidation, Message: "validation"}
	ErrUnauthorized            = &Error{Code: CodeUnauthorized, Message: "unauthorized"}
	ErrInvalidJSON             = &Error{Code: CodeInvalidJSON, Message: "invalid json"}
	ErrInternal                = &Error{Code: CodeInternal, Message: "internal error"}
	ErrFeatureUnavailable      = &Error{Code: CodeFeatureUnavailable, Message: "feature unavailable"}
	ErrGenerationConflict      = &Error{Code: CodeGenerationConflict, Message: "generation conflict"}
	ErrIdempotencyConflict     = &Error{Code: CodeIdempotencyConflict, Message: "idempotency conflict"}
	ErrInitialAlreadyCompleted = &Error{Code: CodeInitialAlreadyCompleted, Message: "initial already completed"}
	ErrPromptLimitReached      = &Error{Code: CodePromptLimitReached, Message: "prompt limit reached"}
	ErrBillingConflict         = &Error{Code: CodeBillingConflict, Message: "billing conflict"}
	ErrBillingRequired         = &Error{Code: CodeBillingRequired, Message: "trial or payment required"}
)

type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

type Error struct {
	Code    string
	Fields  []FieldError
	Message string
	RunID   uuid.UUID
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	if e.Message != "" {
		return e.Message
	}

	return e.Code
}

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}

	return e.Code == t.Code
}

func Validation(fields ...FieldError) error {
	return &Error{Code: CodeValidation, Message: "validation", Fields: fields}
}
