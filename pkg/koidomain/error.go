package koidomain

const (
	CodeValidation         = "validation"
	CodeEmailTaken         = "email_taken"
	CodeInvalidCredentials = "invalid_credentials"
	CodeNotFound           = "not_found"
	CodeUnauthorized       = "unauthorized"
	CodeInvalidJSON        = "invalid_json"
	CodeInternal           = "internal"

	FieldInvalid   = "invalid"
	FieldMinLength = "min_length"
	FieldTaken     = "taken"
)

var (
	ErrNotFound           = &Error{Code: CodeNotFound, Message: "not found"}
	ErrEmailTaken         = &Error{Code: CodeEmailTaken, Message: "email taken", Fields: []FieldError{{Field: "email", Code: FieldTaken}}}
	ErrInvalidCredentials = &Error{Code: CodeInvalidCredentials, Message: "invalid credentials"}
	ErrValidation         = &Error{Code: CodeValidation, Message: "validation"}
	ErrUnauthorized       = &Error{Code: CodeUnauthorized, Message: "unauthorized"}
	ErrInvalidJSON        = &Error{Code: CodeInvalidJSON, Message: "invalid json"}
	ErrInternal           = &Error{Code: CodeInternal, Message: "internal error"}
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
