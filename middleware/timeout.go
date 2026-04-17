package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/khaicode-xyz/core-shared/apperror"
	"github.com/khaicode-xyz/core-shared/response"
)

// Timeout returns a middleware that cancels the request context after the given duration.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					response.Error(w, apperror.ErrTimeout("request timeout"))
				}
			}
		})
	}
}
