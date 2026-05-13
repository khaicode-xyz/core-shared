// Package telemetry initializes OpenTelemetry traces, metrics and logs and
// exports them to a SignOz-compatible OTLP/gRPC endpoint.
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
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otellog "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
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

// Init wires OTLP exporters for traces, metrics and logs and installs them as
// the global providers. Returns a shutdown that flushes and closes exporters.
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

	traceOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithTimeout(10 * time.Second),
	}
	metricOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		otlpmetricgrpc.WithTimeout(10 * time.Second),
	}
	logOpts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.Endpoint),
		otlploggrpc.WithTimeout(10 * time.Second),
	}
	if cfg.Insecure {
		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
		logOpts = append(logOpts, otlploggrpc.WithInsecure())
	} else {
		creds := credentials.NewTLS(nil)
		traceOpts = append(traceOpts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(creds)))
		metricOpts = append(metricOpts, otlpmetricgrpc.WithDialOption(grpc.WithTransportCredentials(creds)))
		logOpts = append(logOpts, otlploggrpc.WithDialOption(grpc.WithTransportCredentials(creds)))
	}

	traceExp, err := otlptrace.New(ctx, otlptracegrpc.NewClient(traceOpts...))
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	metricExp, err := otlpmetricgrpc.New(ctx, metricOpts...)
	if err != nil {
		_ = traceExp.Shutdown(ctx)
		return nil, fmt.Errorf("create otlp metric exporter: %w", err)
	}

	logExp, err := otlploggrpc.New(ctx, logOpts...)
	if err != nil {
		_ = traceExp.Shutdown(ctx)
		_ = metricExp.Shutdown(ctx)
		return nil, fmt.Errorf("create otlp log exporter: %w", err)
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

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)
	otellog.SetLoggerProvider(lp)

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
			lp.Shutdown(shutdownCtx),
		)
	}, nil
}
