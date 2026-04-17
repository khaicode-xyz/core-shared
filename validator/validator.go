package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/khaicode-xyz/core-shared/apperror"
)

var validate = validator.New()

// Validate validates a struct using tags. Returns an *apperror.AppError on failure.
func Validate(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return apperror.ErrBadRequest("invalid request")
	}

	var messages []string
	for _, fe := range validationErrors {
		messages = append(messages, fmt.Sprintf("field '%s' failed on '%s' validation", fe.Field(), fe.Tag()))
	}

	return apperror.ErrBadRequest(strings.Join(messages, "; "))
}
