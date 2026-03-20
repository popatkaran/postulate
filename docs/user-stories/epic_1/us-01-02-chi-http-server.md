# US-01-02 — Chi HTTP Server with Lifecycle Management

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have

---

## 1. Story

As a **platform engineer**, I need a running Chi HTTP server with a correctly structured router, versioned route groups, and a server lifecycle that can be started and stopped cleanly so that all API endpoints have a consistent, versioned home and the server can be operated safely in production.

---

## 2. Background

Chi was selected as the HTTP router because it is fully stdlib-compatible — every handler and middleware uses the standard `http.Handler` interface. This compatibility is critical for the plugin system, where plugin authors may contribute middleware without needing to know which HTTP framework Postulate uses.

The server lifecycle must be managed explicitly. The `main` function wires dependencies and starts the server. The server runs until it receives a shutdown signal, at which point it drains in-flight requests before exiting. Graceful shutdown is covered fully in US-01-08 but the lifecycle hooks must be established here.

---

## 3. Acceptance Criteria

1. The API server starts on a configurable port (default `8080`) and logs its bound address on startup.
2. The router is structured with the following route groups established and ready to receive handlers:
   ```
   /health         — operational endpoints (unauthenticated)
   /v1/auth/       — authentication endpoints (unauthenticated)
   /v1/            — all other versioned API endpoints (authenticated)
   ```
3. A request to any unregistered route returns a JSON response conforming to RFC 7807 with status `404` — not the default Chi plain-text 404.
4. A request using an unsupported HTTP method on a registered route returns a JSON response conforming to RFC 7807 with status `405`.
5. The server exposes a `Shutdown(ctx context.Context) error` method that stops accepting new connections and waits for in-flight requests to complete.
6. The server struct is fully dependency-injected — it accepts a `chi.Router`, a `*slog.Logger`, and a configuration struct via constructor. It does not reference any global state.
7. `go vet ./...` and `golangci-lint run ./...` produce zero issues on all new code.
8. Unit tests cover: server startup, 404 response format, 405 response format, and the shutdown method.

---

## 4. Tasks

### Task 1 — Add Chi dependency
- Add `github.com/go-chi/chi/v5` to `api/go.mod`
- Run `go mod tidy` in `api/`
- Verify the dependency resolves correctly

### Task 2 — Define server configuration struct
- Create `api/internal/config/config.go`
- Define `ServerConfig` struct with fields: `Port int`, `ShutdownTimeoutSeconds int`, `Environment string`
- All fields must have `yaml` struct tags and corresponding environment variable override support (covered fully in US-01-03 — create the struct here, wire configuration loading later)

### Task 3 — Implement the router factory
- Create `api/internal/router/router.go`
- Implement `New(logger *slog.Logger) chi.Router` function
- Register custom 404 and 405 handlers that return RFC 7807-compliant JSON responses
- Define the `/health`, `/v1/auth/`, and `/v1/` route groups as empty route groups
- No handlers registered in this story — route group structure only

### Task 4 — Implement the HTTP server struct
- Create `api/internal/server/server.go`
- Define `Server` struct with fields: `httpServer *http.Server`, `logger *slog.Logger`
- Implement `New(cfg config.ServerConfig, router http.Handler, logger *slog.Logger) *Server` constructor
- Implement `Start() error` method that begins listening and logs the bound address
- Implement `Shutdown(ctx context.Context) error` method that delegates to `http.Server.Shutdown`

### Task 5 — Implement main entrypoint
- Create `api/cmd/api/main.go`
- Wire `ServerConfig`, router, logger, and server via explicit construction — no global variables
- Call `server.Start()` and block until shutdown signal received (signal handling wired in US-01-08 — for now block indefinitely)
- The `main` function must contain no business logic — only wiring and lifecycle orchestration

### Task 6 — Unit tests
- Create `api/internal/router/router_test.go`
- Test that unregistered route returns `404` with `application/problem+json` content type
- Test that unsupported method on registered route returns `405` with `application/problem+json` content type
- Create `api/internal/server/server_test.go`
- Test that `Shutdown` returns nil when called on a running server
- All tests must use `net/http/httptest` — no real network connections

---

## 5. Definition of Done

- All tasks completed
- Server starts and responds on the configured port
- 404 and 405 responses return RFC 7807 JSON
- All unit tests pass with `-race` flag
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
