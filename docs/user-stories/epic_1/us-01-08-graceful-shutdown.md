# US-01-08 — Graceful Shutdown

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have

---

## 1. Story

As a **platform operator**, I need the API server to handle termination signals by stopping the acceptance of new connections and completing in-flight requests before exiting so that rolling deployments, pod restarts, and scheduled maintenance do not cause request failures for active clients.

---

## 2. Background

In a Kubernetes deployment, the platform receives a `SIGTERM` signal when a pod is scheduled for termination. Without graceful shutdown, any in-flight request at the moment of termination returns an abrupt connection reset to the client. With graceful shutdown, the server stops accepting new connections, waits for in-flight requests to complete up to a configurable timeout, then exits cleanly.

The readiness probe (`/ready`) introduced in US-01-04 must transition to `not_ready` immediately on receiving a termination signal — before the drain period begins. This prevents the load balancer from routing new traffic to an instance that is shutting down.

The shutdown timeout is configurable via `server.shutdown_timeout_seconds`. If in-flight requests do not complete within the timeout, the server exits anyway — waiting indefinitely is not acceptable in an automated deployment pipeline.

---

## 3. Acceptance Criteria

1. The server handles both `SIGTERM` and `SIGINT` signals.
2. On receiving a termination signal:
   - The readiness handler immediately returns `503` for subsequent `/ready` requests.
   - The server stops accepting new connections.
   - In-flight requests are allowed to complete.
   - The server waits up to `server.shutdown_timeout_seconds` for in-flight requests to complete.
   - If the timeout is exceeded, the server exits with log level `WARN` and exit code `0`.
   - If all in-flight requests complete within the timeout, the server exits with log level `INFO` and exit code `0`.
3. The server logs the following events during shutdown:
   - `INFO` — shutdown signal received, naming the signal
   - `INFO` — drain period started, with the configured timeout value
   - `INFO` or `WARN` — shutdown complete (with or without timeout exceeded)
4. The shutdown process completes within `shutdown_timeout_seconds + 2` seconds in all cases — the 2-second buffer accounts for signal handling overhead.
5. A test demonstrates that a request in-flight during shutdown completes successfully.

---

## 4. Tasks

### Task 1 — Implement signal handling in main
- Update `api/cmd/api/main.go`
- Create a `context.Context` with cancel attached to `os.Signal` notification for `syscall.SIGTERM` and `os.Interrupt` (`SIGINT`)
- Use `signal.NotifyContext` (available in Go 1.16+) for clean context-based signal handling

### Task 2 — Wire shutdown sequence
- Update `api/cmd/api/main.go`
- When the signal context is cancelled, execute the following sequence in order:
  1. Call `readyHandler.SetNotReady()` to immediately return `503` from `/ready`
  2. Log `INFO` — shutdown signal received
  3. Create a shutdown context with `shutdown_timeout_seconds` deadline
  4. Call `server.Shutdown(shutdownCtx)`
  5. Log `INFO` or `WARN` depending on whether shutdown completed within the timeout
  6. Exit with code `0`

### Task 3 — Implement SetNotReady on the ready handler
- Update `api/internal/handler/health_handler.go`
- Add `SetNotReady()` method to `ReadyHandler` that sets an internal `ready` boolean to `false`
- The `ready` boolean must be managed with an `atomic.Bool` to avoid a data race between the shutdown goroutine and the handler goroutine

### Task 4 — Integration test for graceful shutdown
- Create `api/internal/server/shutdown_test.go`
- Start a test server on a random port using `net/http/httptest`
- Issue a slow request (using a handler that sleeps 200ms before responding)
- Send a shutdown signal to the server while the request is in-flight
- Assert that the slow request completes with a `200` response
- Assert that a new request issued after shutdown begins returns a connection error

### Task 5 — Update Makefile
- Add a `run` target to the Makefile that starts the API with a local `config.yaml`
- Document in `CONTRIBUTING.md` how to test graceful shutdown locally using `kill -SIGTERM`

---

## 5. Definition of Done

- All tasks completed
- Server handles `SIGTERM` and `SIGINT` cleanly
- Readiness probe returns `503` immediately after signal received
- In-flight requests complete before server exits
- Shutdown timeout is respected — server exits even if requests do not complete in time
- Integration test passes with `-race` flag
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
