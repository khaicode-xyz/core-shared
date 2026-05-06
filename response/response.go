package response

import (
	"encoding/json"
	"net/http"

	"github.com/khaicode-xyz/core-shared/apperror"
)

// Response is the §4.1-compliant envelope.
//
// Legacy fields `Success` and `Error` are retained for back-compat; new call sites
// should set `Status` and `Errors[]` via the helpers below.
type Response struct {
	Status    bool        `json:"status"`
	Data      interface{} `json:"data,omitempty"`
	Errors    []ErrorItem `json:"errors,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
	RequestID string      `json:"request_id,omitempty"`

	// --- legacy fields ---
	Success bool       `json:"success,omitempty"`
	Error   *ErrorBody `json:"error,omitempty"`
}

// ErrorItem is one entry in §4.1's `errors[]` array.
type ErrorItem struct {
	Code    string                 `json:"code"`              // §4.3 code: SVC-FEAT-CLASS-NAME
	Class   string                 `json:"class,omitempty"`   // V | B | S
	Message string                 `json:"message"`
	Field   string                 `json:"field,omitempty"`   // optional — which input field caused this
	Meta    map[string]interface{} `json:"meta,omitempty"`    // optional — supplementary key/values (no PII per §6.4)
}

// ErrorBody is the legacy single-error wrapper retained for back-compat.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a success response. Sets both `status` (new) and `success` (legacy).
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Status:  true,
		Success: true,
		Data:    data,
	})
}

// OK writes a 200 success response.
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, data)
}

// Created writes a 201 success response.
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, data)
}

// Accepted writes a 202 response.
func Accepted(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusAccepted, data)
}

// NoContent writes a 204 response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error writes an error response. Builds a §4.1 envelope (`status=false`, `errors[]`)
// while also populating the legacy `error` field so existing clients keep working.
func Error(w http.ResponseWriter, err error) {
	appErr, ok := apperror.As(err)
	if !ok {
		appErr = &apperror.AppError{
			Code:       "INTERNAL_ERROR",
			Message:    "Internal Server Error",
			HTTPStatus: http.StatusInternalServerError,
			Class:      string(apperror.ClassS),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)
	json.NewEncoder(w).Encode(Response{
		Status:  false,
		Success: false,
		Errors: []ErrorItem{{
			Code:    appErr.Code,
			Class:   appErr.Class,
			Message: appErr.Message,
		}},
		Error: &ErrorBody{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}

// ErrorMulti writes a multi-error response — useful for surfacing all
// FluentValidation/validator failures at once per §4.4.
func ErrorMulti(w http.ResponseWriter, status int, items []ErrorItem) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := Response{
		Status:  false,
		Success: false,
		Errors:  items,
	}
	if len(items) > 0 {
		body.Error = &ErrorBody{Code: items[0].Code, Message: items[0].Message}
	}
	json.NewEncoder(w).Encode(body)
}
