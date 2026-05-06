package logger

import (
	"github.com/khaicode-xyz/core-shared/apperror"
)

// errAsApp is a thin wrapper so logger.go doesn't pull apperror into its public API.
func errAsApp(err error) (*apperror.AppError, bool) {
	if err == nil {
		return nil, false
	}
	return apperror.As(err)
}
