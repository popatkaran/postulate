package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/popatkaran/postulate/api/internal/middleware"
)

// setupTracer installs an in-memory tracer provider and returns the exporter.
func setupTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return exp
}

func TestTracing_SpanCreatedForEachRequest(t *testing.T) {
	// Arrange
	exp := setupTracer(t)
	handler := middleware.Tracing(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/version", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	if len(exp.GetSpans()) == 0 {
		t.Error("expected at least one span to be created")
	}
}

func TestTracing_InboundTraceparentUsedAsParent(t *testing.T) {
	// Arrange
	exp := setupTracer(t)
	handler := middleware.Tracing(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	parentTraceID, _ := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
	parentSpanID, _ := trace.SpanIDFromHex("b7ad6b7169203331")
	traceparent := "00-" + parentTraceID.String() + "-" + parentSpanID.String() + "-01"

	req := httptest.NewRequest(http.MethodGet, "/v1/version", nil)
	req.Header.Set("Traceparent", traceparent)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert — the created span must share the parent trace ID.
	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected a span to be created")
	}
	if spans[0].SpanContext.TraceID() != parentTraceID {
		t.Errorf("expected trace ID %s, got %s", parentTraceID, spans[0].SpanContext.TraceID())
	}
}

func TestTracing_SpanIncludesHTTPMethodAttribute(t *testing.T) {
	// Arrange
	exp := setupTracer(t)
	handler := middleware.Tracing(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/v1/generate", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected a span")
	}
	found := false
	for _, attr := range spans[0].Attributes {
		if string(attr.Key) == "http.request.method" && attr.Value.AsString() == http.MethodPost {
			found = true
		}
	}
	if !found {
		t.Errorf("expected http.request.method=POST attribute in span, got: %v", spans[0].Attributes)
	}
}

func TestTracing_TraceIDAccessibleFromRequestContext(t *testing.T) {
	// Arrange
	setupTracer(t)
	var capturedCtx context.Context
	handler := middleware.Tracing(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/version", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	span := trace.SpanFromContext(capturedCtx)
	if !span.SpanContext().IsValid() {
		t.Error("expected a valid span context in request context")
	}
	if span.SpanContext().TraceID() == (trace.TraceID{}) {
		t.Error("expected non-zero trace ID in request context")
	}
}
