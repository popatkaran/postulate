# Epic 01 — Validation Report

**Validated:** 2026-03-20
**Validator:** Staff Engineer review — code, tests, and configuration only (no checklists)
**Result:** Functional implementation complete. Coverage gate fails.

---

## Story-by-Story Verdict

### US-01-01 — Monorepo and Go Workspace Initialisation ✅ PASS

`go.work` declares `go 1.26` and uses all four modules: `./api`, `./cli`, `./sdk`, `./plugins/platform-standards`. Each has its own `go.mod`. `go build ./api/... ./cli/... ./sdk/... ./plugins/platform-standards/...` exits 0. Stub `doc.go` files exist in `cli`, `sdk`, and `plugins/platform-standards` to satisfy the workspace. The `api` module is on `go 1.26` with Chi, OTel, and YAML dependencies correctly declared.

---

### US-01-02 — Chi HTTP Server with Lifecycle Management ✅ PASS

`server.New` constructs an `http.Server` with `ReadHeaderTimeout: 10s` and no global state. `Start()` calls `ListenAndServe`, `Shutdown()` delegates to `http.Server.Shutdown`. Chi is used in `router.New`. Middleware chain order is correct: `Tracing → RequestID → AccessLog`. Three server tests verify: shutdown returns nil on a running server, in-flight requests complete before shutdown, new requests after shutdown are refused.

---

### US-01-03 — Configuration Loading and Startup Validation ✅ PASS

`config.Load()` reads from `POSTULATE_CONFIG_FILE` or defaults to `./config.yaml` (missing default is silently skipped; missing explicit path is an error). `applyEnvOverrides` maps all `POSTULATE_*` env vars. `config.Validate` collects all errors into `ValidationErrors` and returns them together. Validated fields: port range 1–65535, environment one of `development|staging|production`, `shutdown_timeout_seconds > 0`, `service_id` non-empty. Nine tests cover all paths including multi-error collection.

**Gap:** `config.LogSafe` has 0% coverage — called in `main` but has no unit test.

---

### US-01-04 — Health, Readiness, Liveness, and Version Endpoints ✅ PASS

All four endpoints implemented and routed:
- `GET /health` → `HealthHandler` — 200/503 based on `Aggregator.Check()`, JSON body with `status`, `timestamp`, `checks`
- `GET /health/ready` → `ReadyHandler` — atomic bool, 200/503
- `GET /health/live` → `LiveHandler` — unconditional 200
- `GET /v1/version` → `VersionHandler` — returns `version`, `commit`, `build_time`, `go_version`, `environment`

`DefaultBuildInfo()` reads linker-injected vars; `NewVersionHandler` always populates `GoVersion` from `runtime.Version()`. Build flags in `Makefile` and `Dockerfile` inject `VERSION`, `COMMIT`, `BUILD_TIME`. Seven handler tests cover all status paths and response body schema.

**Gap:** `DefaultBuildInfo()` has 0% coverage — only called from `main`.

---

### US-01-05 — Structured JSON Logging Foundation ✅ PASS

`logger.New` uses `slog.NewJSONHandler` for `production`/`staging`, `slog.NewTextHandler` for `development`. `serviceId` and `instanceId` are pre-set as default attributes. The `otelHandler` wrapper injects `traceId` and `spanId` from the OTel span context on every record. `parseLevel` defaults to `INFO` for unrecognised values.

**Naming discrepancy:** Epic AC specifies `timestamp` but `slog.NewJSONHandler` emits `time`. The test checks for `time` and passes. This is correct Go stdlib behaviour but diverges from the AC wording.

**Coverage gap:** `logger.New` (production constructor) has 0% coverage. Tests use `NewWithHandler` (test-friendly variant). `parseLevel` also has 0% coverage. The production code path for environment-based handler selection is untested.

---

### US-01-06 — RFC 7807 Error Response Format ✅ PASS

`problem.Problem` has all five required fields: `type`, `title`, `status`, `detail`, `instance`. `problem.Write` sets `Content-Type: application/problem+json`, populates `instance` from `r.URL.Path` when empty, and populates `request_id` from context. `ValidationProblem` extends with `errors []FieldError`. Six tests cover: content type, status code, instance population, errors array, no stack trace leak, all required fields present. Router registers RFC 7807 handlers for 404 and 405.

---

### US-01-07 — Request Middleware — Request ID and Access Logging ✅ PASS

`RequestID` middleware: generates a ULID via `oklog/ulid/v2` when no valid header is present, accepts existing ULID (26-char Crockford base32) or UUID (8-4-4-4-12 hex), stores in context via unexported `contextKey`, echoes in `X-Request-ID` response header. Five tests cover all paths.

`AccessLog` middleware: logs one entry per request with `requestId`, `method`, `path`, `status`, `duration_ms`, `response_bytes`. Health paths (`/health`, `/health/ready`, `/health/live`) are excluded from logging. `statusRecorder` wraps `ResponseWriter` to capture status and byte count. Three tests verify required fields, health path exclusion, and correct method/path/status values.

**Gap:** `statusRecorder.Write` has 0% coverage — the byte-counting path is never exercised in tests.

---

### US-01-08 — Graceful Shutdown ✅ PASS

`main.go` uses `signal.NotifyContext` for `SIGTERM`/`SIGINT`. On signal: `readyHandler.SetNotReady()` is called first (stops load balancer traffic), then `server.Shutdown(ctx)` with the configured timeout, then `otelShutdown`. `ReadyHandler.SetNotReady` uses an `atomic.Bool`. Three shutdown tests verify: clean shutdown, in-flight request completes (200ms handler, shutdown after 50ms), new requests refused after shutdown.

**Gap:** `SetNotReady()` has 0% coverage in unit tests — only exercised in `main`.

---

### US-01-09 — OpenTelemetry SDK Wiring ✅ PASS

`telemetry.Setup` initialises `TracerProvider` and `MeterProvider`. When `OTLPEndpoint` is empty, uses `tracetest.NewNoopExporter()` and `sdkmetric.NewManualReader()`. When set, uses OTLP gRPC exporters. W3C `TraceContext` propagator is registered globally. `Tracing` middleware uses `otelhttp.NewHandler` with Chi route pattern as span name. Metrics instruments: `http.server.request.duration` histogram and `http.server.active_requests` up-down counter.

**Coverage gap — significant:** `telemetry/metrics.go` has 0% coverage. `NewMetrics`, `RecordRequest`, and `IncActive` are never called in tests. The `AccessLog` middleware calls these but tests pass `nil` for the metrics argument. `buildTracerProvider` and `buildMeterProvider` are at 50% (only the no-endpoint branch tested). `noopShutdown` is 0%.

---

### US-01-10 — Multi-Stage Non-Root Dockerfile ✅ PASS

Two-stage build: `golang:1.26-alpine` builder, `gcr.io/distroless/static-debian12` final. `USER nonroot:nonroot` is set. `HEALTHCHECK` uses `-healthcheck` flag which calls `GET /health` and exits 0/1. `EXPOSE 8080`. Build args `VERSION`, `COMMIT`, `BUILD_TIME` passed through to `-ldflags`. CI pipeline runs `trivy` image scan with `--severity CRITICAL --exit-code 1`.

---

## Coverage Gate — FAIL ❌

Epic AC #10 and ADR-0003 require **90% unit test coverage across all packages**. Actual results:

| Package | Coverage | Status |
|---|---|---|
| `internal/config` | 95.7% | ✅ |
| `internal/handler` | 92.6% | ✅ |
| `internal/health` | 100.0% | ✅ |
| `internal/logger` | 58.0% | ❌ |
| `internal/middleware` | 78.7% | ❌ |
| `internal/problem` | 93.3% | ✅ |
| `internal/router` | 100.0% | ✅ |
| `internal/server` | 100.0% | ✅ |
| `internal/telemetry` | 43.1% | ❌ |
| **Total** | **76.4%** | ❌ |

### What needs tests before Epic 01 can close

**`internal/logger` (58%):**
- `logger.New` — production constructor, environment-based handler selection
- `parseLevel` — all branches (debug, warn, error, invalid/default)
- `otelHandler.WithGroup` — 0% coverage

**`internal/middleware` (78.7%):**
- `statusRecorder.Write` — byte counting path
- `AccessLog` with custom `excludedPaths` argument
- `RequestIDFromContext` empty-string branch (no value in context)

**`internal/telemetry` (43.1%):**
- `NewMetrics` — instrument registration
- `RecordRequest` — histogram and counter recording
- `IncActive` — counter increment
- `noopShutdown` — trivial but uncounted

---

## Known Issues for Epic 02

1. **`timestamp` vs `time` field name** — AC says `timestamp`, slog emits `time`. Decide whether to align the AC or wrap the handler to rename the field before Epic 02 logging work builds on this.
2. **`config.LogSafe` untested** — if Epic 02 adds sensitive fields (DB passwords, session secrets), this function becomes security-critical and must have coverage before those fields are added.
3. **Metrics wiring is nominal only** — `AccessLog` accepts `nil` for metrics in all tests. Epic 02 will likely add more instrumentation; the metrics path should be tested end-to-end before that work starts.

---

## Three things worth flagging before you start Epic 02:

1. Coverage gate is the blocker — 76.4% total, needs to hit 90%. The three packages to fix are internal/telemetry (43%), internal/logger (58%), and internal/middleware (79%
). The telemetry metrics package is the biggest gap — NewMetrics, RecordRequest, and IncActive have zero tests because every test passes nil for the metrics argument.

2. timestamp vs time — the epic AC says timestamp but slog emits time. It's correct Go stdlib behaviour, but if Epic 02 docs or downstream tooling reference timestamp, you'
ll hit a mismatch. Worth a one-line decision now.

3. config.LogSafe needs a test before Epic 02 adds DB credentials — right now it's harmless, but once database config lands it becomes a security-relevant function and
should be covered before that happens.

---

## Post-Validation Fix Log

### 2026-03-23 — CI Pipeline Unblocked

The CI pipeline was failing before any test or coverage results could be produced, due to a toolchain mismatch independent of the coverage gaps identified in this report.

**Root cause:** `golangci-lint-action@v6` bundled golangci-lint `v1.64.8`, which does not support the `version: "2"` / `formatters` keys in `.golangci.yml`. The `config verify` step aborted with `additional properties not allowed`.

**Fix applied:**
- `golangci-lint-action` upgraded to `v9` with golangci-lint pinned to `v2.11.4` (latest, 2026-03-22).
- Go version in CI corrected to `1.26.1`.
- `api/go.mod` bumped from `go 1.24` → `go 1.26`.

The coverage failures documented in this report remain the active work items under US-01-11.
