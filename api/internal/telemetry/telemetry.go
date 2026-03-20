// Package telemetry initialises the OpenTelemetry SDK for the Postulate API.
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/popatkaran/postulate/api/internal/config"
)

// traceExporterFactory and metricExporterFactory are package-level vars so tests
// can substitute failing implementations to exercise error paths.
var (
	traceExporterFactory  = defaultTraceExporter
	metricExporterFactory = defaultMetricExporter
)

// Setup initialises the global TracerProvider and MeterProvider.
// When cfg.OTLPEndpoint is non-empty, traces and metrics are exported via OTLP gRPC.
// When absent, no-op exporters are used — the SDK is wired but nothing leaves the process.
// Returns a shutdown function that must be called during graceful shutdown.
func Setup(ctx context.Context, cfg config.ObservabilityConfig) (shutdown func(context.Context) error, err error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceID)),
	)
	if err != nil {
		return noopShutdown, fmt.Errorf("creating OTel resource: %w", err)
	}

	tp, err := buildTracerProvider(ctx, res, cfg.OTLPEndpoint)
	if err != nil {
		return noopShutdown, err
	}

	mp, err := buildMeterProvider(ctx, res, cfg.OTLPEndpoint)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return noopShutdown, err
	}

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	// Register W3C TraceContext as the global text-map propagator.
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func(ctx context.Context) error {
		tErr := tp.Shutdown(ctx)
		mErr := mp.Shutdown(ctx)
		if tErr != nil {
			return tErr
		}
		return mErr
	}, nil
}

func buildTracerProvider(ctx context.Context, res *resource.Resource, endpoint string) (*sdktrace.TracerProvider, error) {
	exporter, err := traceExporterFactory(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	), nil
}

func buildMeterProvider(ctx context.Context, res *resource.Resource, endpoint string) (*sdkmetric.MeterProvider, error) {
	reader, err := metricExporterFactory(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	), nil
}

func defaultTraceExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	if endpoint == "" {
		return tracetest.NewNoopExporter(), nil
	}
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}
	return exp, nil
}

func defaultMetricExporter(ctx context.Context, endpoint string) (sdkmetric.Reader, error) {
	if endpoint == "" {
		return sdkmetric.NewManualReader(), nil
	}
	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP metric exporter: %w", err)
	}
	return sdkmetric.NewPeriodicReader(exp), nil
}

func noopShutdown(_ context.Context) error { return nil }
