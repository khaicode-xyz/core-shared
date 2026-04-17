package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/khaicode-xyz/core-shared/logger"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

// Logging returns a middleware that logs each HTTP request with structured fields.
func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			reqLogger := log.With(
				slog.String("request_id", GetRequestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			)
			ctx := logger.WithContext(r.Context(), reqLogger)

			next.ServeHTTP(sw, r.WithContext(ctx))

			reqLogger.Info("request completed",
				slog.Int("status", sw.status),
				slog.Int("size", sw.size),
				slog.Duration("duration", time.Since(start)),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}
