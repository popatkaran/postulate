package logger

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// otelHandler is a slog.Handler wrapper that injects OTel trace context fields
// and redacts sensitive attribute values before delegating to the underlying handler.
type otelHandler struct {
	inner slog.Handler
}

func (h *otelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
	traceID, spanID := extractSpanIDs(ctx)

	// Build a new record with a zero Time so the underlying handler does not emit
	// the default "time" key. We emit "timestamp" as an explicit attribute instead,
	// satisfying the Postulate logging standard.
	clean := slog.NewRecord(time.Time{}, r.Level, r.Message, r.PC)
	clean.AddAttrs(
		slog.Time("timestamp", r.Time),
		slog.String("traceId", traceID),
		slog.String("spanId", spanID),
	)

	// Copy original attributes, redacting sensitive keys.
	r.Attrs(func(a slog.Attr) bool {
		if isSensitive(a.Key) {
			a.Value = slog.StringValue("[redacted]")
		}
		clean.AddAttrs(a)
		return true
	})

	return h.inner.Handle(ctx, clean)
}

func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Redact sensitive attrs at the WithAttrs level too.
	safe := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		if isSensitive(a.Key) {
			a.Value = slog.StringValue("[redacted]")
		}
		safe[i] = a
	}
	return &otelHandler{inner: h.inner.WithAttrs(safe)}
}

func (h *otelHandler) WithGroup(name string) slog.Handler {
	return &otelHandler{inner: h.inner.WithGroup(name)}
}

// extractSpanIDs returns the hex traceId and spanId from the OTel span in ctx,
// or empty strings when no active span is present.
func extractSpanIDs(ctx context.Context) (traceID, spanID string) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return "", ""
	}
	return span.SpanContext().TraceID().String(),
		span.SpanContext().SpanID().String()
}
