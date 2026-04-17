// Package telemetry initializes OpenTelemetry traces + metrics and exports
// them to a SignOz-compatible OTLP/gRPC endpoint.
//
// Usage:
//
//	shutdown, err := telemetry.Init(ctx, telemetry.Config{...})
//	defer shutdown(ctx)
//
// If Endpoint is empty, Init returns a no-op shutdown and installs no-op
// providers — the app runs unchanged.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	Endpoint    string
	Insecure    bool
	SampleRate  float64
	ServiceName string
	ServiceCode string
	Env         string
}

type ShutdownFunc func(context.Context) error

// Init wires OTLP exporters for traces and metrics and installs them as the
// global providers. Returns a shutdown that flushes and closes exporters.
func Init(ctx context.Context, cfg Config, logger *slog.Logger) (ShutdownFunc, error) {
	if cfg.Endpoint == "" {
		logger.Info("signoz disabled (SIGNOZ_ENDPOINT empty)")
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceNamespace(cfg.ServiceCode),
			semconv.DeploymentEnvironment(cfg.Env),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	dialOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithTimeout(10 * time.Second),
	}
	metricOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		otlpmetricgrpc.WithTimeout(10 * time.Second),
	}
	if cfg.Insecure {
		dialOpts = append(dialOpts, otlptracegrpc.WithInsecure())
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
	} else {
		creds := credentials.NewTLS(nil)
		dialOpts = append(dialOpts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(creds)))
		metricOpts = append(metricOpts, otlpmetricgrpc.WithDialOption(grpc.WithTransportCredentials(creds)))
	}
	traceExp, err := otlptrace.New(ctx, otlptracegrpc.NewClient(dialOpts...))
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	metricExp, err := otlpmetricgrpc.New(ctx, metricOpts...)
	if err != nil {
		_ = traceExp.Shutdown(ctx)
		return nil, fmt.Errorf("create otlp metric exporter: %w", err)
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp, sdkmetric.WithInterval(30*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	logger.Info("signoz telemetry initialized",
		slog.String("endpoint", cfg.Endpoint),
		slog.Bool("insecure", cfg.Insecure),
		slog.Float64("sample_rate", cfg.SampleRate),
	)

	return func(shutdownCtx context.Context) error {
		return errors.Join(
			tp.Shutdown(shutdownCtx),
			mp.Shutdown(shutdownCtx),
		)
	}, nil
}
