package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "github.com/popatkaran/postulate/api"

// Metrics holds the HTTP server metric instruments.
type Metrics struct {
	requestDuration metric.Float64Histogram
	activeRequests  metric.Int64UpDownCounter
}

// NewMetrics creates and registers the HTTP server metric instruments using mp.
// On instrument registration error it returns an error.
func NewMetrics(mp metric.MeterProvider) (*Metrics, error) {
	meter := mp.Meter(meterName)

	dur, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP server request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating request duration histogram: %w", err)
	}

	active, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP server requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating active requests counter: %w", err)
	}

	return &Metrics{requestDuration: dur, activeRequests: active}, nil
}

// RecordRequest records duration and decrements the active request counter.
func (m *Metrics) RecordRequest(ctx context.Context, method, route string, status int, durationMs float64) {
	attrs := metric.WithAttributes(
		attribute.String("http.method", method),
		attribute.String("http.route", route),
		attribute.Int("http.status_code", status),
	)
	m.requestDuration.Record(ctx, durationMs, attrs)
	m.activeRequests.Add(ctx, -1, attrs)
}

// IncActive increments the active request counter.
func (m *Metrics) IncActive(ctx context.Context, method, route string) {
	m.activeRequests.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.route", route),
		),
	)
}
