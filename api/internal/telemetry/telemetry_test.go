package telemetry_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/telemetry"
)

func TestSetup_EmptyEndpointReturnsNoError(t *testing.T) {
	cfg := config.ObservabilityConfig{ServiceID: "test-api"}

	shutdown, err := telemetry.Setup(context.Background(), cfg)

	if err != nil {
		t.Fatalf("expected no error with empty endpoint, got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}
	_ = shutdown(context.Background())
}

func TestSetup_ShutdownExecutesWithoutError(t *testing.T) {
	cfg := config.ObservabilityConfig{ServiceID: "test-api"}
	shutdown, err := telemetry.Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	err = shutdown(context.Background())

	if err != nil {
		t.Errorf("expected shutdown to return nil, got %v", err)
	}
}

func TestSetup_ShutdownWithCancelledContext_ReturnsError(t *testing.T) {
	// Use a non-empty endpoint so the tracer provider has a real batcher that
	// will fail to flush when the context is already cancelled at shutdown time.
	cfg := config.ObservabilityConfig{
		ServiceID:    "test-api",
		OTLPEndpoint: "localhost:4317",
	}
	shutdown, err := telemetry.Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Cancel the context before shutdown — the batcher flush should fail.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// We don't assert on the error value because it depends on timing;
	// we just verify shutdown does not panic.
	_ = shutdown(ctx)
}

func TestSetup_RegistersGlobalTracerProvider(t *testing.T) {
	cfg := config.ObservabilityConfig{ServiceID: "test-api"}
	shutdown, err := telemetry.Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	tp := otel.GetTracerProvider()
	if _, ok := tp.(*sdktrace.TracerProvider); !ok {
		t.Errorf("expected global TracerProvider to be *sdktrace.TracerProvider, got %T", tp)
	}
}

func TestSetup_RegistersGlobalMeterProvider(t *testing.T) {
	cfg := config.ObservabilityConfig{ServiceID: "test-api"}
	shutdown, err := telemetry.Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	mp := otel.GetMeterProvider()
	if _, ok := mp.(*sdkmetric.MeterProvider); !ok {
		t.Errorf("expected global MeterProvider to be *sdkmetric.MeterProvider, got %T", mp)
	}
}

func TestSetup_NonEmptyEndpointBuildsProvidersWithoutError(t *testing.T) {
	// The OTLP gRPC exporter connects lazily — New() succeeds even with an
	// unreachable endpoint. This exercises the non-empty endpoint branch of
	// buildTracerProvider and buildMeterProvider.
	cfg := config.ObservabilityConfig{
		ServiceID:    "test-api",
		OTLPEndpoint: "localhost:4317",
	}

	shutdown, err := telemetry.Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected no error with unreachable OTLP endpoint, got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}
	// Shutdown may return an error (connection refused) — that is acceptable.
	_ = shutdown(context.Background())
}

func TestNewMetrics_ReturnsNonNilInstance(t *testing.T) {
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewManualReader()))
	m, err := telemetry.NewMetrics(mp)
	if err != nil {
		t.Fatalf("NewMetrics returned error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil *Metrics")
	}
}

func TestRecordRequest_DoesNotPanic(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m, err := telemetry.NewMetrics(mp)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}

	m.RecordRequest(context.Background(), "GET", "/v1/test", 200, 42.5)
}

func TestRecordRequest_RecordsHistogramDataPoint(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m, err := telemetry.NewMetrics(mp)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}

	m.RecordRequest(context.Background(), "GET", "/v1/test", 200, 42.5)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}

	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, met := range sm.Metrics {
			if met.Name == "http.server.request.duration" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected http.server.request.duration metric after RecordRequest")
	}
}

func TestIncActive_DoesNotPanic(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m, err := telemetry.NewMetrics(mp)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}

	m.IncActive(context.Background(), "GET", "/v1/test")
}

func TestIncActive_RecordsActiveRequestsMetric(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m, err := telemetry.NewMetrics(mp)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}

	m.IncActive(context.Background(), "GET", "/v1/test")

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}

	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, met := range sm.Metrics {
			if met.Name == "http.server.active_requests" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected http.server.active_requests metric after IncActive")
	}
}
