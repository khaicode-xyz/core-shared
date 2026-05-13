package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type ctxKey struct{}
type requestIDKey struct{}

// New creates a §5.4-compliant structured JSON logger.
//
// Always emits: service.name, service.code, level, msg, time.
// Per-record (when present in context): trace.id, span.id, request.id, feature.code, error.code, error.class.
func New(serviceName, serviceCode, env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}

	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: env == "development",
	})

	handler := newFanoutHandler(stdoutHandler, newOtelHandler(serviceName))

	return slog.New(handler).With(
		slog.String("service.name", serviceName),
		slog.String("service.code", serviceCode),
	)
}

// WithContext stores a logger in the context.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext retrieves the logger from context, enriched with §5.4 trace + request fields.
// If no logger was stored, the default logger is returned (still enriched).
func FromContext(ctx context.Context) *slog.Logger {
	l, ok := ctx.Value(ctxKey{}).(*slog.Logger)
	if !ok {
		l = slog.Default()
	}

	if span := trace.SpanContextFromContext(ctx); span.IsValid() {
		l = l.With(
			slog.String("trace.id", span.TraceID().String()),
			slog.String("span.id", span.SpanID().String()),
		)
	}
	if rid, ok := ctx.Value(requestIDKey{}).(string); ok && rid != "" {
		l = l.With(slog.String("request.id", rid))
	}
	return l
}

// WithRequestID stores a per-request id (e.g. from the X-Request-ID middleware) in the context
// so subsequent FromContext(ctx) calls auto-attach it as request.id (§5.4).
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// LogError emits an error log with §5.4 + §4.5 attributes (error.code, error.class, error.feature).
// Pass nil for the appErr arg if you only have a raw error.
func LogError(ctx context.Context, l *slog.Logger, msg string, err error, attrs ...any) {
	if l == nil {
		l = FromContext(ctx)
	}
	enriched := append([]any{}, attrs...)
	enriched = append(enriched, slog.String("error.message", err.Error()))
	if appErr, ok := errAsApp(err); ok {
		enriched = append(enriched,
			slog.String("error.code", appErr.Code),
			slog.String("error.class", appErr.Class),
			slog.String("error.feature", appErr.Feature),
		)
	}
	l.ErrorContext(ctx, msg, enriched...)
}
