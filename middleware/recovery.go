package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/khaicode-xyz/core-shared/response"
)

// Recovery returns a middleware that recovers from panics, logs them, and returns a 500 JSON response.
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := string(debug.Stack())
					log.Error("panic recovered",
						slog.Any("error", err),
						slog.String("stack", stack),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.String("request_id", GetRequestID(r.Context())),
					)
					response.Error(w, nil)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
