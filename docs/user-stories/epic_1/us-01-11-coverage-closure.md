# US-01-11 — Epic 01 Coverage Closure and Pre-Epic-02 Hardening

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have — Epic 01 blocker
**Created:** 2026-03-20
**Trigger:** Epic 01 Validation Report — coverage gate failure and three pre-Epic-02 risk items

---

## 1. Story

As a **platform engineer**, I need the Epic 01 coverage gate to pass and three identified pre-Epic-02 risks to be resolved so that Epic 01 can be formally closed and Epic 02 begins on a stable, tested, and correctly specified foundation.

---

## 2. Background

Epic 01 functional implementation is complete. All ten original stories passed functional review. The Epic cannot close because:

1. **Coverage gate fails** — 76.4% total against a 90% requirement. Three packages are below the gate: `internal/telemetry` (43.1%), `internal/logger` (58.0%), `internal/middleware` (78.7%). The root cause in each case is that production code paths are only exercised from `main` and were never covered by unit tests.

2. **`timestamp` vs `time` field name** — The Epic 01 AC and the Postulate logging standard specify `timestamp` as the log field name. The Go stdlib `slog.NewJSONHandler` emits `time`. This divergence is currently harmless but will cause silent mismatches in log aggregation queries, alerting rules, and any Epic 02 work that references the field name. It must be corrected at the handler level before Epic 02 builds on top of it.

3. **`config.LogSafe` untested** — The function exists but has 0% coverage. It is currently harmless because no sensitive configuration fields exist. Epic 02 introduces database credentials into the configuration struct. If `LogSafe` has no regression coverage when those fields land, there is no protection against accidental credential logging.

4. **Metrics wiring nominal only** — Every `AccessLog` test passes `nil` for the metrics argument. The `RecordRequest`, `IncActive`, and `NewMetrics` code paths have never been exercised in tests. Epic 02 adds further instrumentation; the metrics path must be tested end-to-end before that work begins.

---

## 3. Acceptance Criteria

### 3.1 Coverage Gate

1. `go test -coverprofile=coverage.out ./internal/...` from `api/` produces coverage at or above 90% for every package and in aggregate.
2. The following specific packages must meet the gate individually:

   | Package | Minimum Coverage |
   |---|---|
   | `internal/logger` | ≥ 90% |
   | `internal/middleware` | ≥ 90% |
   | `internal/telemetry` | ≥ 90% |

3. No existing passing tests are modified to reduce their coverage or weaken their assertions in order to meet the gate. Coverage must be earned by new tests, not by removing existing ones.

### 3.2 `timestamp` Field Name

4. All JSON log output emits `timestamp` as the field name for the log time, not `time`.
5. The fix is implemented in the custom `slog.Handler` wrapper (`internal/logger/handler.go`) by replacing the `time` key with `timestamp` during the `Handle` method — not by modifying the upstream `slog` library or changing the log call sites.
6. The existing logger tests that previously asserted for `time` are updated to assert for `timestamp`.
7. No other field names in the log output are changed.

### 3.3 `config.LogSafe` Coverage

8. `config.LogSafe` has unit tests covering:
   - A configuration with no sensitive fields returns all values unredacted.
   - A configuration with a field designated sensitive returns `[redacted]` for that field's value.
   - The returned map contains every top-level configuration key — no keys are silently dropped.
9. A `// SECURITY:` comment is added to `LogSafe` in source documenting which fields are currently designated sensitive and the process for adding new ones.

### 3.4 Metrics Wiring

10. All `AccessLog` middleware tests that previously passed `nil` for the metrics argument are updated to pass a real `*telemetry.Metrics` instance.
11. Tests assert that `RecordRequest` is called with the correct method, route, status, and duration values after a request completes.
12. Tests assert that `IncActive` increments on request start and decrements on request completion.
13. `NewMetrics`, `RecordRequest`, and `IncActive` each have at least one direct unit test independent of the middleware tests.

### 3.5 General

14. All new and modified tests pass with the `-race` flag.
15. `make lint` passes with zero issues after all changes.
16. The Epic 01 document is updated to record US-01-11 in the story index table and the definition of done reflects the 90% coverage gate as verified.

---

## 4. Tasks

### Task 1 — Fix `timestamp` field name in JSON logger

**File:** `api/internal/logger/handler.go`

- In the `otelHandler.Handle` method, after injecting `traceId` and `spanId` attributes, iterate the record's attributes and replace the key `time` with `timestamp` for the time field
- Alternatively, use `slog.Record.Clone()` and reconstruct with the renamed key before delegating to the underlying handler — this is the cleaner approach
- The most reliable mechanism: override the key at the `Handle` level by calling `r.Attrs` and rebuilding with `slog.Time("timestamp", r.Time)` as the first attribute, then delegating the modified record to the underlying handler
- Update all tests in `logger_test.go` that assert for `"time"` to assert for `"timestamp"`
- Verify no other test in the repository references `"time"` as a log field key

---

### Task 2 — `internal/logger` coverage to ≥90%

**Files to add/modify:** `api/internal/logger/logger_test.go`, `api/internal/logger/handler_test.go`

**`logger.New` — production constructor**
- Test: calling `logger.New` with `environment = "production"` returns a logger that emits JSON output to the provided writer
- Test: calling `logger.New` with `environment = "development"` returns a logger that emits text output
- Test: calling `logger.New` with `environment = "staging"` returns a logger that emits JSON output
- Use `bytes.Buffer` as the writer in tests — inject a custom writer rather than using `os.Stdout`
  - Note: if `logger.New` currently hardcodes `os.Stdout`, add an optional `io.Writer` parameter or a `NewWithWriter` test variant that accepts a writer — this is the correct pattern for testable loggers

**`parseLevel` — all branches**
- Test: `"debug"` returns `slog.LevelDebug`
- Test: `"warn"` returns `slog.LevelWarn`
- Test: `"error"` returns `slog.LevelError`
- Test: `"info"` returns `slog.LevelInfo`
- Test: empty string returns `slog.LevelInfo` (default)
- Test: unrecognised value `"verbose"` returns `slog.LevelInfo` (default)

**`otelHandler.WithGroup`**
- Test: `WithGroup` returns a new handler with the group name applied
- Test: log records emitted after `WithGroup` are nested under the group name in JSON output

---

### Task 3 — `internal/middleware` coverage to ≥90%

**Files to add/modify:** `api/internal/middleware/accesslog_test.go`, `api/internal/middleware/requestid_test.go`

**`statusRecorder.Write` — byte counting**
- Test: writing a 50-byte response body sets `recorder.bytes` to 50
- Test: writing in two separate calls (25 bytes + 25 bytes) sets `recorder.bytes` to 50
- Test: `responseBytes` field in the access log entry matches the actual response body size

**`AccessLog` with custom excluded paths**
- Test: a path registered in the `excludedPaths` option produces no log entry
- Test: a path not in `excludedPaths` produces a log entry
- Test: default exclusions (`/health`, `/health/ready`, `/health/live`) are excluded even when `excludedPaths` is nil

**`RequestIDFromContext` — empty branch**
- Test: calling `RequestIDFromContext` on a context with no request ID set returns an empty string (not a panic)

---

### Task 4 — `internal/telemetry` coverage to ≥90%

**Files to add/modify:** `api/internal/telemetry/telemetry_test.go`, `api/internal/telemetry/metrics_test.go`

**Provider construction — both branches**
- Test: `Setup` with a non-empty `OTLPEndpoint` attempts OTLP gRPC exporter construction — use a test gRPC server or mock the exporter interface to avoid a real network call
- If mocking the exporter is complex, use an in-process `otlptracegrpc` server from `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/otlptracetest` if available, otherwise stub via interface
- Test: `Setup` with empty `OTLPEndpoint` uses no-op providers and returns no error — this path is already tested, verify it still passes

**`noopShutdown`**
- Test: `noopShutdown(ctx)` returns nil — trivial but required for coverage

**`buildTracerProvider` and `buildMeterProvider`**
- Test: both functions return non-nil providers when called with valid arguments
- Test: providers are registered as the global OTel providers after `Setup` completes

**`NewMetrics` — instrument registration**
- Test: `NewMetrics` returns a non-nil `*Metrics` instance
- Test: `NewMetrics` returns an error if the meter provider is nil
- Use `sdkmetric.NewManualReader()` and `sdkmetric.NewMeterProvider` in tests — no real exporter needed

**`RecordRequest` — histogram and counter recording**
- Test: calling `RecordRequest` with a real `*Metrics` instance does not panic
- Test: after calling `RecordRequest`, collect metrics from the `ManualReader` and assert that `http.server.request.duration` has one data point with the expected labels (`method`, `route`, `status`)

**`IncActive` — counter increment and decrement**
- Test: `IncActive(ctx, 1)` increments the active request counter
- Test: `IncActive(ctx, -1)` decrements it
- Test: net result of one increment followed by one decrement is zero — collect from `ManualReader` and assert

---

### Task 5 — Fix `AccessLog` tests to use real Metrics

**File:** `api/internal/middleware/accesslog_test.go`

- Replace all `nil` metrics arguments in existing `AccessLog` tests with a real `*telemetry.Metrics` instance constructed using `sdkmetric.NewManualReader()` and `sdkmetric.NewMeterProvider`
- After each test request, collect metrics from the `ManualReader` and assert:
  - `http.server.request.duration` has one data point
  - The data point carries the correct `http.method` and `http.status_code` labels
- This closes the gap where `RecordRequest` was always receiving `nil` in test execution

---

### Task 6 — `config.LogSafe` tests and security comment

**Files:** `api/internal/config/redact_test.go`, `api/internal/config/redact.go`

- Add a `// SECURITY:` block comment above `LogSafe` in `redact.go` with the following content:
  ```go
  // SECURITY: LogSafe returns a representation of the configuration safe for
  // logging. Any field designated sensitive must be added to the sensitiveFields
  // slice below and must appear in the unit tests in redact_test.go.
  // Current sensitive fields: none (Epic 01).
  // Epic 02 will add: database.password, database.dsn.
  ```
- Add tests in `redact_test.go`:
  - Test: config with no sensitive fields — all keys present in returned map, all values match original
  - Test: when a field is manually designated sensitive in the function, its value in the map is `"[redacted]"` and its key is still present
  - Test: the map contains keys for `server` and `observability` top-level sections — no key is silently absent
  - Test: `LogSafe` does not mutate the original `Config` struct

---

### Task 7 — Update Epic 01 document

**File:** `epic-01-api-skeleton-routing-health.md`

- Add `US-01-11` to the story index table with title "Epic 01 Coverage Closure and Pre-Epic-02 Hardening" and priority "Must Have"
- Update acceptance criterion #10 to read: "Minimum 90% unit test coverage across all packages introduced in this Epic, verified by `go test -coverprofile` — gate enforced in CI"
- Update the definition of done to add: "Coverage gate verified by CI — `go test -coverprofile` shows ≥90% per package and in aggregate"

---

### Task 8 — Verify coverage gate in CI

- Run `go test -race -coverprofile=coverage.out ./internal/...` from `api/`
- Run `go tool cover -func=coverage.out` and confirm every package meets ≥90%
- The CI pipeline step added in US-01-10 must enforce the gate — add `-covermode=atomic` and a threshold check using `go tool cover` output or a coverage enforcement tool
- The pipeline must fail if any package falls below 90%

---

## 5. Definition of Done

- All tasks completed
- `go test -race -coverprofile=coverage.out ./internal/...` shows ≥90% for every package individually and in aggregate
- All JSON log output emits `timestamp` — verified by logger tests
- `config.LogSafe` has passing tests covering all paths including the sensitive field redaction path
- All `AccessLog` tests pass a real `*telemetry.Metrics` instance and assert on recorded metric values
- No existing tests weakened or removed
- `make lint` passes with zero issues
- Epic 01 document updated to include US-01-11
- CI coverage gate enforced — pipeline fails below 90%
- PR reviewed and approved by the same reviewer who produced the validation report
