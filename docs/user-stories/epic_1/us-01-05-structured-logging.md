# US-01-05 — Structured JSON Logging Foundation

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have

---

## 1. Story

As a **platform operator**, I need all API server log output to be structured JSON with a consistent set of required fields so that logs are machine-parseable, aggregatable, and searchable in the centralised logging system without manual parsing.

---

## 2. Background

Structured logging is a Tier 1 standard in Postulate — every generated microservice carries it. Postulate's own API must also conform to this standard as the platform that enforces it.

Go 1.21 introduced `log/slog` as a structured logging package in the standard library. This is the chosen logger — it requires no external dependency, integrates cleanly with the OTel trace context, and produces the JSON output format required.

The logger is a shared dependency injected throughout the application. It must never be accessed as a global variable. All log entries must include the required fields defined in the scope document: `timestamp`, `level`, `traceId`, `spanId`, `serviceId`, `instanceId`, and `message`.

---

## 3. Acceptance Criteria

1. All log output is JSON-formatted when `server.environment` is `staging` or `production`. In `development`, human-readable text output is acceptable.
2. Every log entry contains the following fields as a minimum:

   | Field | Source | Format |
   |---|---|---|
   | `timestamp` | Log time | ISO 8601 UTC |
   | `level` | Log level | `DEBUG`, `INFO`, `WARN`, `ERROR` |
   | `traceId` | OTel trace context | Hex string or empty string |
   | `spanId` | OTel trace context | Hex string or empty string |
   | `serviceId` | Configuration — `observability.service_id` | String |
   | `instanceId` | Configuration — `observability.instance_id` or hostname | String |
   | `message` | Log call | String |

3. The log level is configurable via `observability.log_level` in the configuration. Invalid values default to `INFO` with a warning logged at startup.
4. The logger is constructed once in `main` and injected into all components as `*slog.Logger`. No `slog.Default()` or global logger access is permitted outside of `main`.
5. Sensitive data — passwords, tokens, PII — must never appear in log output. Any field name containing `password`, `token`, `secret`, or `key` passed to the logger must be automatically redacted to `[redacted]`.
6. Log entries at `ERROR` level must include a `error` field containing the error string.
7. Unit tests verify: JSON output format, required field presence, sensitive field redaction, and error field on error-level logs.

---

## 4. Tasks

### Task 1 — Implement the logger factory
- Create `api/internal/logger/logger.go`
- Implement `New(cfg config.ObservabilityConfig, environment string) *slog.Logger` function
- In `production` and `staging`: use `slog.NewJSONHandler` with `os.Stdout`
- In `development`: use `slog.NewTextHandler` with `os.Stdout`
- Set the minimum log level from `cfg.LogLevel` — default to `INFO` if invalid
- Return a `*slog.Logger` with `serviceId` and `instanceId` pre-set as default attributes

### Task 2 — Implement the custom JSON handler with required fields
- Create `api/internal/logger/handler.go`
- Implement a custom `slog.Handler` wrapper that adds the following to every log record before delegating to the underlying handler:
  - `traceId` and `spanId` extracted from the OTel trace context on the record's context (empty string if no span active)
  - Automatic redaction of sensitive field names
- The wrapper must implement the full `slog.Handler` interface: `Enabled`, `Handle`, `WithAttrs`, `WithGroup`

### Task 3 — Implement sensitive field redaction
- Create `api/internal/logger/redact.go`
- Define `sensitiveKeyPatterns` as a slice of lowercase substrings: `password`, `token`, `secret`, `key`, `credential`, `authorization`
- Implement `isSensitive(key string) bool` — returns true if the lowercase key contains any sensitive pattern
- In the custom handler's `Handle` method, iterate all attributes and replace sensitive values with `slog.StringValue("[redacted]")`

### Task 4 — Wire logger into main
- Update `api/cmd/api/main.go`
- Construct the logger immediately after configuration is loaded and validated
- Pass the logger to all components that require it via constructor injection
- Remove any `fmt.Println` or `log.Print` calls from main — all output goes through `slog`

### Task 5 — Unit tests
- Create `api/internal/logger/logger_test.go`
- Test: JSON handler produces valid JSON output
- Test: all required fields present in every log entry
- Test: `traceId` and `spanId` populated from OTel context when a span is active
- Test: `traceId` and `spanId` are empty strings when no span is active
- Create `api/internal/logger/redact_test.go`
- Test: field named `password` is redacted
- Test: field named `api_key` is redacted
- Test: field named `access_token` is redacted
- Test: field named `username` is not redacted
- Test: field named `message` is not redacted

---

## 5. Definition of Done

- All tasks completed
- Running server produces JSON log output in staging/production mode
- All required fields present in every log entry verified by tests
- Sensitive field redaction verified by tests
- No global logger access outside of `main`
- All unit tests pass with `-race` flag
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
