# US-01-07 — Request Middleware — Request ID and Access Logging

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have

---

## 1. Story

As a **platform operator**, I need every inbound request to carry a unique identifier and produce a structured access log entry so that individual requests can be traced end-to-end across log entries, correlated with error reports, and used to diagnose issues in production.

---

## 2. Background

Request middleware in Chi is a standard `http.Handler` wrapper applied to the router before any route handler executes. Two middleware components are introduced here:

**Request ID middleware** — generates a unique identifier for each inbound request, stores it in the request context, adds it to the response as `X-Request-ID`, and accepts an existing `X-Request-ID` from the incoming request (for upstream propagation from load balancers or API gateways).

**Access logging middleware** — logs a structured entry for every completed request, recording the method, path, status code, response size, and duration. This entry uses the request ID from the context and the trace context from OTel.

Both middleware components must be placed before route handlers in the middleware chain. They must not interfere with each other's operation.

---

## 3. Acceptance Criteria

1. Every request receives a unique `requestId`. The value is:
   - Read from the incoming `X-Request-ID` header if present and a valid ULID or UUID format.
   - Generated as a new ULID if the header is absent or invalid.
2. The `requestId` is stored in the request context and accessible to all downstream handlers and middleware via a typed context key.
3. Every response includes an `X-Request-ID` header containing the request's `requestId`.
4. Every completed request produces one structured JSON log entry at `INFO` level containing:

   | Field | Value |
   |---|---|
   | `timestamp` | ISO 8601 UTC |
   | `level` | `INFO` |
   | `message` | `"request completed"` |
   | `requestId` | The request's unique ID |
   | `method` | HTTP method |
   | `path` | Request URL path (without query string) |
   | `status` | HTTP response status code |
   | `duration_ms` | Request duration in milliseconds |
   | `response_bytes` | Response body size in bytes |
   | `traceId` | OTel trace ID (from logger handler, US-01-05) |
   | `spanId` | OTel span ID (from logger handler, US-01-05) |

5. Health probe endpoints (`/health`, `/ready`, `/live`) do not produce access log entries. They are excluded from access logging to prevent probe noise in the log stream.
6. The `requestId` is available to the error response writer (US-01-06) for population of the `request_id` field in problem+json responses.
7. Unit tests cover: request ID generation when header absent, request ID propagation when header present, access log entry fields, health endpoint log exclusion.

---

## 4. Tasks

### Task 1 — Add ULID dependency
- Add `github.com/oklog/ulid/v2` to `api/go.mod`
- Run `go mod tidy` in `api/`

### Task 2 — Define the request ID context key
- Create `api/internal/middleware/requestid.go`
- Define an unexported `contextKey` type to avoid context key collisions
- Define `requestIDKey` as the typed context key for request IDs
- Implement `RequestIDFromContext(ctx context.Context) string` helper function
- Implement `ContextWithRequestID(ctx context.Context, id string) context.Context` helper function

### Task 3 — Implement request ID middleware
- In `api/internal/middleware/requestid.go`
- Implement `RequestID(next http.Handler) http.Handler`
- On each request:
  - Check `X-Request-ID` header — accept if present and matches ULID or UUID format
  - Generate a new ULID if header absent or invalid
  - Store in context via `ContextWithRequestID`
  - Set `X-Request-ID` response header before calling `next`

### Task 4 — Implement response writer wrapper
- Create `api/internal/middleware/responsewriter.go`
- Implement `statusRecorder` struct wrapping `http.ResponseWriter`
- Capture `status int` and `bytes int` on first `WriteHeader` and each `Write` call
- This wrapper is used by the access logger to capture the response status and size after the handler completes

### Task 5 — Implement access logging middleware
- Create `api/internal/middleware/accesslog.go`
- Implement `AccessLog(logger *slog.Logger) func(http.Handler) http.Handler`
- Accept a list of excluded paths as a configuration option — default excludes `/health`, `/ready`, `/live`
- For each non-excluded request:
  - Record start time before calling next
  - Wrap the response writer with `statusRecorder`
  - After `next` returns, log the structured access log entry per acceptance criteria
  - Extract `requestId` from context for the log entry

### Task 6 — Register middleware on the router
- Update `api/internal/router/router.go`
- Add `middleware.RequestID` as the first middleware in the chain
- Add `middleware.AccessLog(logger)` as the second middleware in the chain

### Task 7 — Update problem writer to include request ID
- Update `api/internal/problem/writer.go`
- In `Write`, extract `requestId` from the request context using `middleware.RequestIDFromContext`
- Populate `p.RequestID` before serialising the response

### Task 8 — Unit tests
- Create `api/internal/middleware/requestid_test.go`
- Test: request without `X-Request-ID` header receives a generated ULID
- Test: request with valid `X-Request-ID` header retains the provided value
- Test: request with invalid `X-Request-ID` header receives a generated ULID
- Test: `X-Request-ID` response header is set
- Test: request ID is retrievable from context in downstream handler
- Create `api/internal/middleware/accesslog_test.go`
- Test: completed request produces one log entry with all required fields
- Test: `/health` request produces no log entry
- Test: log entry contains correct method, path, status, and duration fields

---

## 5. Definition of Done

- All tasks completed
- Every non-health request produces a structured access log entry
- Every response carries `X-Request-ID`
- Problem responses include `request_id` from context
- Health endpoints produce no access log entries
- All unit tests pass with `-race` flag
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
