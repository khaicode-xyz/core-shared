package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// Class is the error class per Khaicode_LogStandard_v1.1 §4.5:
//   - V: validation (HTTP 400, no alert)
//   - B: business    (HTTP 400/403/404/409, no alert)
//   - S: system      (HTTP 5xx, page on-call)
type Class string

const (
	ClassV Class = "V"
	ClassB Class = "B"
	ClassS Class = "S"
)

// AppError carries the §4.3-coded error envelope plus §4.5 class.
//
// Code format: {SVC}-{FEAT}-{CLASS}-{NAME} (max 60 chars).
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Internal   error  `json:"-"`

	// §4.3 fields — populated by NewCoded / V / B / S helpers.
	// When you build an AppError via the legacy New/Wrap helpers, these stay empty
	// and only Code / Message / HTTPStatus are set (back-compat).
	Class   string `json:"class,omitempty"`   // "V" | "B" | "S"
	Service string `json:"service,omitempty"` // 3-char service code, e.g. "BAN"
	Feature string `json:"feature,omitempty"` // 2-10 char feature code, e.g. "DOWNLOAD"
	Name    string `json:"name,omitempty"`    // SCREAMING_SNAKE error name, e.g. "GRPC_TIMEOUT"
}

func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Internal
}

// NewCoded builds a §4.3-compliant AppError. Pass class explicitly when the
// HTTP status alone doesn't disambiguate (e.g. a 400 that's V vs B).
func NewCoded(svc, feat string, class Class, name, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       fmt.Sprintf("%s-%s-%s-%s", svc, feat, class, name),
		Message:    message,
		HTTPStatus: httpStatus,
		Class:      string(class),
		Service:    svc,
		Feature:    feat,
		Name:       name,
	}
}

// V is the standard validation-class factory (HTTP 400, §4.5: no alert).
func V(svc, feat, name, message string) *AppError {
	return NewCoded(svc, feat, ClassV, name, message, http.StatusBadRequest)
}

// B is the standard business-class factory. Caller picks the HTTP status
// (typically 400 / 403 / 404 / 409). §4.5: no alert.
func B(svc, feat, name, message string, httpStatus int) *AppError {
	return NewCoded(svc, feat, ClassB, name, message, httpStatus)
}

// S is the standard system-class factory (HTTP 500). §4.5: pages on-call.
// `err` is preserved as the inner cause for log/trace correlation.
func S(svc, feat, name, message string, err error) *AppError {
	a := NewCoded(svc, feat, ClassS, name, message, http.StatusInternalServerError)
	a.Internal = err
	return a
}

// New is the legacy constructor — kept for back-compat. Prefer NewCoded / V / B / S.
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap is the legacy wrapping constructor — kept for back-compat. Prefer S(...).
func Wrap(code, message string, httpStatus int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Internal:   err,
	}
}

// --- Legacy generic factories (back-compat). Codes here do NOT follow §4.3
// because they don't carry SVC/FEAT context. New call sites should use V / B / S
// with a concrete service + feature code. ---

func ErrBadRequest(message string) *AppError {
	return New("BAD_REQUEST", message, http.StatusBadRequest)
}

func ErrNotFound(message string) *AppError {
	return New("NOT_FOUND", message, http.StatusNotFound)
}

func ErrUnauthorized(message string) *AppError {
	return New("UNAUTHORIZED", message, http.StatusUnauthorized)
}

func ErrForbidden(message string) *AppError {
	return New("FORBIDDEN", message, http.StatusForbidden)
}

func ErrInternal(message string, err error) *AppError {
	return Wrap("INTERNAL_ERROR", message, http.StatusInternalServerError, err)
}

func ErrConflict(message string) *AppError {
	return New("CONFLICT", message, http.StatusConflict)
}

func ErrTimeout(message string) *AppError {
	return New("TIMEOUT", message, http.StatusGatewayTimeout)
}

func ErrTooLarge(message string) *AppError {
	return New("PAYLOAD_TOO_LARGE", message, http.StatusRequestEntityTooLarge)
}

// As extracts an *AppError from the error chain.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
