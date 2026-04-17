package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Internal   error  `json:"-"`
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

func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func Wrap(code, message string, httpStatus int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Internal:   err,
	}
}

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
