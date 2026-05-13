package logger

import (
	"context"
	"log/slog"
)

// fanoutHandler dispatches every record to all child handlers. Used so a single
// *slog.Logger can write to stdout AND emit OTel log records simultaneously.
type fanoutHandler struct {
	handlers []slog.Handler
}

func newFanoutHandler(handlers ...slog.Handler) slog.Handler {
	return &fanoutHandler{handlers: handlers}
}

func (h *fanoutHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, child := range h.handlers {
		if child.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (h *fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, child := range h.handlers {
		if !child.Enabled(ctx, r.Level) {
			continue
		}
		// slog.Record stores attrs in a shared slice; clone before passing to
		// each child so concurrent handlers can't stomp on one another.
		if err := child.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (h *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, child := range h.handlers {
		out[i] = child.WithAttrs(attrs)
	}
	return &fanoutHandler{handlers: out}
}

func (h *fanoutHandler) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, len(h.handlers))
	for i, child := range h.handlers {
		out[i] = child.WithGroup(name)
	}
	return &fanoutHandler{handlers: out}
}
