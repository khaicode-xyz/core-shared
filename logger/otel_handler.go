package logger

import (
	"context"
	"log/slog"
	"time"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

// otelHandler bridges slog records to the OpenTelemetry global LoggerProvider.
//
// The provider is resolved on every Handle call (not cached at construction)
// so logger.New can run before telemetry.Init without losing log signals once
// telemetry comes online. If no provider is configured the global default is a
// no-op and this handler is effectively free.
type otelHandler struct {
	scope string
	attrs []slog.Attr
	group string
}

func newOtelHandler(scope string) slog.Handler {
	return &otelHandler{scope: scope}
}

func (h *otelHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
	otelLogger := global.GetLoggerProvider().Logger(h.scope)

	var rec otellog.Record
	rec.SetTimestamp(r.Time)
	rec.SetBody(otellog.StringValue(r.Message))
	rec.SetSeverity(slogToOtelSeverity(r.Level))
	rec.SetSeverityText(r.Level.String())

	for _, a := range h.attrs {
		rec.AddAttributes(slogAttrToKV(h.group, a))
	}
	r.Attrs(func(a slog.Attr) bool {
		rec.AddAttributes(slogAttrToKV(h.group, a))
		return true
	})

	otelLogger.Emit(ctx, rec)
	return nil
}

func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *h
	cp.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &cp
}

func (h *otelHandler) WithGroup(name string) slog.Handler {
	cp := *h
	if h.group != "" {
		cp.group = h.group + "." + name
	} else {
		cp.group = name
	}
	return &cp
}

func slogToOtelSeverity(l slog.Level) otellog.Severity {
	switch {
	case l >= slog.LevelError:
		return otellog.SeverityError
	case l >= slog.LevelWarn:
		return otellog.SeverityWarn
	case l >= slog.LevelInfo:
		return otellog.SeverityInfo
	default:
		return otellog.SeverityDebug
	}
}

func slogAttrToKV(group string, a slog.Attr) otellog.KeyValue {
	key := a.Key
	if group != "" {
		key = group + "." + key
	}
	return otellog.KeyValue{Key: key, Value: slogValueToOtelValue(a.Value)}
}

func slogValueToOtelValue(v slog.Value) otellog.Value {
	v = v.Resolve()
	switch v.Kind() {
	case slog.KindBool:
		return otellog.BoolValue(v.Bool())
	case slog.KindFloat64:
		return otellog.Float64Value(v.Float64())
	case slog.KindInt64:
		return otellog.Int64Value(v.Int64())
	case slog.KindUint64:
		return otellog.Int64Value(int64(v.Uint64()))
	case slog.KindString:
		return otellog.StringValue(v.String())
	case slog.KindTime:
		return otellog.StringValue(v.Time().Format(time.RFC3339Nano))
	case slog.KindDuration:
		return otellog.StringValue(v.Duration().String())
	case slog.KindGroup:
		kvs := make([]otellog.KeyValue, 0, len(v.Group()))
		for _, a := range v.Group() {
			kvs = append(kvs, otellog.KeyValue{Key: a.Key, Value: slogValueToOtelValue(a.Value)})
		}
		return otellog.MapValue(kvs...)
	default:
		return otellog.StringValue(v.String())
	}
}
