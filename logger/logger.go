package logger

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey struct{}

// New creates a structured JSON logger instance.
func New(serviceName, serviceCode, env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: env == "development",
	})

	return slog.New(handler).With(
		slog.String("service_name", serviceName),
		slog.String("service_code", serviceCode),
	)
}

// WithContext stores a logger in the context.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext retrieves the logger from context, or returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
