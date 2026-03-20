# US-01-09 — OpenTelemetry SDK Wiring

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have

---

## 1. Story

As a **platform operator**, I need the Postulate API to emit distributed traces and metrics via OpenTelemetry so that request flows, latencies, and error rates are observable in the platform's monitoring infrastructure.

---

## 2. Background

This story concerns Postulate's own operational observability as a SaaS platform — not the observability plugins that generate OTel configuration into user-created microservices. Those are entirely separate concerns addressed in Epic 12.

OpenTelemetry is the chosen observability standard. The SDK is wired here: a tracer provider and meter provider are initialised at startup, a trace context is attached to every inbound request, and the W3C `traceparent` header is read on inbound requests and propagated on outbound calls.

The OTel SDK export destination is configurable. When `observability.otlp_endpoint` is set in configuration, traces and metrics are exported to that OTLP gRPC endpoint. When absent, a no-op exporter is used — the SDK is still wired but nothing leaves the process. This allows the server to start cleanly in local development without a collector.

---

## 3. Acceptance Criteria

1. A `TracerProvider` is initialised at startup using the OTLP gRPC exporter if `observability.otlp_endpoint` is configured, or a no-op exporter if absent.
2. A `MeterProvider` is initialised at startup with the same export configuration.
3. The W3C `traceparent` propagator is registered as the global propagator — inbound `traceparent` headers are extracted and used as the parent span context.
4. Every inbound HTTP request has a span created for it. The span includes:
   - `http.method`
   - `http.route` (the Chi route pattern, not the raw path)
   - `http.status_code` (set after handler completes)
   - `http.url`
5. The trace ID and span ID from the active span are available in the request context for use by the logger (as defined in US-01-05 — this story provides the span context that the logger reads).
6. The following HTTP server metrics are emitted:
   - `http.server.request.duration` — histogram of request duration in milliseconds, labelled with `http.method`, `http.route`, and `http.status_code`
   - `http.server.active_requests` — gauge of currently active requests
7. The `TracerProvider` and `MeterProvider` are shut down as part of the graceful shutdown sequence (US-01-08) — after the HTTP server drains but before the process exits.
8. OTel SDK initialisation errors do not crash the server — they are logged at `WARN` level and the server continues with no-op providers.
9. Unit tests cover: span creation on requests, trace context propagation from inbound `traceparent` header, no-op provider used when endpoint absent.

---

## 4. Tasks

### Task 1 — Add OTel dependencies
- Add the following to `api/go.mod`:
  - `go.opentelemetry.io/otel`
  - `go.opentelemetry.io/otel/sdk`
  - `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`
  - `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc`
  - `go.opentelemetry.io/otel/propagators/w3c/tracecontext` (for W3C propagation)
  - `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
- Run `go mod tidy`

### Task 2 — Implement the OTel provider factory
- Create `api/internal/telemetry/telemetry.go`
- Implement `Setup(ctx context.Context, cfg config.ObservabilityConfig) (shutdown func(context.Context) error, err error)`
- Initialise trace exporter: OTLP gRPC if `cfg.OTLPEndpoint` is non-empty, otherwise no-op
- Initialise metric exporter: same conditional logic
- Configure `TracerProvider` with the service name from `cfg.ServiceID` as a resource attribute
- Configure `MeterProvider` with the same resource
- Register W3C `TraceContext` propagator as the global propagator
- Return a `shutdown` function that flushes and closes both providers

### Task 3 — Implement the OTel HTTP middleware
- Create `api/internal/middleware/tracing.go`
- Implement `Tracing(next http.Handler) http.Handler` using `otelhttp.NewHandler`
- Configure the handler to use the Chi route pattern as `http.route` (not the raw URL path)
- Ensure the span is closed after the handler returns with the correct status code

### Task 4 — Implement HTTP server metrics
- Create `api/internal/telemetry/metrics.go`
- Register `http.server.request.duration` histogram instrument
- Register `http.server.active_requests` up-down counter instrument
- Implement `RecordRequest(ctx context.Context, method, route string, status int, durationMs float64)` helper
- The metrics middleware records these values — wire into the access logging middleware (US-01-07) rather than adding a separate middleware

### Task 5 — Wire OTel into main
- Update `api/cmd/api/main.go`
- Call `telemetry.Setup` immediately after logger initialisation
- If setup returns an error, log at `WARN` and continue with no-op providers — do not exit
- Add the OTel `shutdown` function to the graceful shutdown sequence in US-01-08 — call it after `server.Shutdown` completes

### Task 6 — Register tracing middleware on the router
- Update `api/internal/router/router.go`
- Add `middleware.Tracing` as the first middleware in the chain — before request ID and access logging
- This ensures the trace context is established before any other middleware runs

### Task 7 — Unit tests
- Create `api/internal/telemetry/telemetry_test.go`
- Test: setup with empty OTLP endpoint returns no error and no-op providers
- Test: shutdown function executes without error
- Create `api/internal/middleware/tracing_test.go`
- Test: span is created for each request
- Test: inbound `traceparent` header is used as parent span context
- Test: span includes `http.method` and `http.status_code` attributes
- Test: trace ID is accessible from request context after middleware runs

---

## 5. Definition of Done

- All tasks completed
- Server emits spans for all inbound requests
- W3C `traceparent` propagation verified by test
- Server starts cleanly with no OTLP endpoint configured
- OTel providers shut down cleanly as part of graceful shutdown
- All unit tests pass with `-race` flag
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
