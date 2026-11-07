package domain

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
	CodeIdempotencyConflict       = "idempotency_conflict"
	CodeInitialAlreadyCompleted   = "initial_already_completed"
	CodeRunNotReady               = "run_not_ready"
	CodePromptLimitReached        = "prompt_limit_reached"
	CodeBillingConflict           = "billing_conflict"
	CodeBillingCheckoutPending    = "billing_checkout_pending"
	CodeBillingSubscriptionExists = "billing_subscription_exists"
	CodeBillingPriceMismatch      = "billing_price_mismatch"
	CodeBillingRequired           = "billing_required"
	CodePasswordResetInvalid      = "password_reset_invalid"
	CodeRateLimited               = "rate_limited"
	CodeSignupClosed              = "signup_closed"
	CodeFirstLookUnavailable      = "first_look_unavailable"
	CodeCrossOriginRequest        = "cross_origin_request"

	FieldInvalid   = "invalid"
	FieldMinLength = "min_length"
	FieldMaxLength = "max_length"
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
	ErrIdempotencyConflict     = &Error{Code: CodeIdempotencyConflict, Message: "idempotency conflict"}
	ErrInitialAlreadyCompleted = &Error{Code: CodeInitialAlreadyCompleted, Message: "initial already completed"}
	ErrPromptLimitReached      = &Error{Code: CodePromptLimitReached, Message: "prompt limit reached"}
	ErrBillingConflict         = &Error{Code: CodeBillingConflict, Message: "billing conflict"}
	ErrBillingRequired         = &Error{Code: CodeBillingRequired, Message: "trial or payment required"}
	ErrPasswordResetInvalid    = &Error{Code: CodePasswordResetInvalid, Message: "password reset link invalid or expired"}
	ErrRateLimited             = &Error{Code: CodeRateLimited, Message: "too many requests"}
	ErrSignupClosed            = &Error{Code: CodeSignupClosed, Message: "sign-up is closed on this installation"}
	ErrFirstLookUnavailable    = &Error{Code: CodeFirstLookUnavailable, Message: "free check unavailable"}
	ErrCrossOriginRequest      = &Error{Code: CodeCrossOriginRequest, Message: "cross-origin request rejected"}
)

type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

type Error struct {
	Code    string
	Fields  []FieldError
	Message string
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
