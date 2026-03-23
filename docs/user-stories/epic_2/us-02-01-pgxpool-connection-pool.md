# US-02-01 — pgxpool Connection Pool and Lifecycle

**Epic:** Epic 02 — Database Schema and Migration Tooling
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have
**Depends on:** US-02-00 (PostgreSQL running locally)

---

## 1. Story

As a **platform engineer**, I need a `pgxpool` connection pool initialised at startup and closed during graceful shutdown so that the API server has a reliable, configurable, and lifecycle-managed database connection layer that all subsequent data access code can use.

---

## 2. Background

`pgxpool` is the native connection pool from the `pgx` v5 driver library. It manages a pool of reusable PostgreSQL connections, handles reconnection on failure, enforces connection limits, and integrates cleanly with Go's `context.Context` for timeout and cancellation propagation.

The pool is a shared dependency — constructed once in `main` and injected into every repository. It must not be accessed as a global variable. Its lifecycle is tied to the application lifecycle: opened after configuration is validated and before requests are accepted, closed after the HTTP server drains during graceful shutdown.

The pool configuration — maximum open connections, maximum idle connections, connection maximum lifetime — is sourced from the `database` section of the application configuration introduced in US-02-00.

---

## 3. Acceptance Criteria

1. The connection pool is initialised in `main` after configuration validation and before the startup database reachability check introduced in US-02-00.
2. Pool configuration is sourced from `config.DatabaseConfig`:
   - `max_open_conns` — maximum number of connections in the pool (default 25)
   - `max_idle_conns` — minimum number of idle connections maintained (default 5)
   - `conn_max_lifetime_seconds` — maximum time a connection may be reused (default 300)
3. The DSN (data source name) is constructed from individual config fields — host, port, name, user, password, ssl_mode — and never logged. The DSN is treated as a sensitive value equivalent to a password.
4. The pool performs an initial `Ping` on startup to verify the database is reachable. On failure the server exits with code 1 and logs an actionable error message (this replaces the temporary reachability check in US-02-00 — `startup.CheckDatabase` is refactored to use the pool).
5. The pool is closed as part of the graceful shutdown sequence, after the HTTP server has drained and after OTel providers have shut down. This ordering ensures in-flight requests that hold database connections complete before the pool is closed.
6. On pool close, the following is logged at `INFO` level: `"database connection pool closed"`.
7. The pool exposes a `Stats()` method result that is accessible for the health contributor (US-02-02) — specifically `AcquiredConns()`, `IdleConns()`, and `MaxConns()`.
8. The pool struct is wrapped in a thin `api/internal/database/pool.go` layer that exposes only the interface required by the application — `Acquire`, `Exec`, `Query`, `QueryRow`, `BeginTx`, `Ping`, `Stats`, `Close`. This prevents `pgxpool` internals from leaking into business logic packages.
9. Unit tests cover: DSN construction from config fields, pool configuration values applied correctly, pool close called during shutdown sequence.
10. Integration test covers: pool connects to `postulate_test` database, `Ping` returns nil, `Stats` returns non-zero `MaxConns`.

---

## 4. DSN Construction

The DSN is built from individual configuration fields using the `pgx` connection string format:

```
host=<host> port=<port> dbname=<name> user=<user> password=<password> sslmode=<ssl_mode>
```

The constructed DSN must never appear in log output. The `LogSafe` function already redacts `database.password` — the DSN itself is never stored in any struct field that passes through the logger.

---

## 5. Graceful Shutdown Ordering

The complete graceful shutdown sequence after this story, in order:

1. Receive `SIGTERM` / `SIGINT`
2. Set ready handler to not-ready (stops load balancer traffic)
3. HTTP server drain — wait for in-flight requests up to `shutdown_timeout_seconds`
4. OTel provider flush and close
5. **Database connection pool close** ← introduced in this story
6. Process exit with code 0

---

## 6. Tasks

### Task 1 — Add pgx dependency
- Add `github.com/jackc/pgx/v5` to `api/go.mod`
- Run `go mod tidy` in `api/`
- Verify the dependency resolves correctly

### Task 2 — Implement the database package
- Create `api/internal/database/pool.go`
- Define `Pool` interface exposing: `Acquire`, `Exec`, `Query`, `QueryRow`, `BeginTx`, `Ping`, `Stats`, `Close`
- Implement `New(ctx context.Context, cfg config.DatabaseConfig, logger *slog.Logger) (*pgxpool.Pool, error)`:
  - Build DSN from config fields
  - Build `pgxpool.Config` from pool size and lifetime settings
  - Call `pgxpool.NewWithConfig`
  - Call `Ping` — return error on failure with structured log entry containing `host`, `port`, `name` fields (no password)
  - Log `INFO` on successful connection: `"database connection pool established"` with `max_conns`, `host`, `name` fields
- Implement `BuildDSN(cfg config.DatabaseConfig) string` as a package-private function (exported only for tests)

### Task 3 — Refactor startup check to use pool
- Update `api/internal/startup/checks.go`
- Remove the temporary direct connection approach from US-02-00
- `CheckDatabase` now accepts a `*pgxpool.Pool` and calls `pool.Ping(ctx)`
- Update `api/cmd/api/main.go` to construct the pool first, then pass it to `CheckDatabase`

### Task 4 — Wire pool into graceful shutdown
- Update `api/cmd/api/main.go`
- Add pool close as step 5 in the shutdown sequence per Section 5
- Wrap pool close in a timeout context of 10 seconds — log `WARN` if it exceeds the timeout
- Log `INFO "database connection pool closed"` on successful close

### Task 5 — Unit tests
- Create `api/internal/database/pool_test.go`
- Test: `BuildDSN` constructs correct connection string from config fields
- Test: `BuildDSN` includes `sslmode` parameter correctly for all three ssl_mode values
- Test: `BuildDSN` output does not contain password when called with an empty password (boundary check)
- Test: pool configuration respects `max_open_conns`, `max_idle_conns`, and `conn_max_lifetime_seconds` from config
- All unit tests use `pgxmock` or construct the config struct directly — no real database connection

### Task 6 — Integration tests
- Create `api/internal/database/pool_integration_test.go` with `//go:build integration` tag
- Test: `New` returns a non-nil pool when connecting to `postulate_test` database
- Test: `Ping` returns nil on a healthy pool
- Test: `Stats().MaxConns()` equals the configured `max_open_conns` value
- Test: pool close completes without error

---

## 7. Definition of Done

- All tasks completed
- Pool initialises at startup and logs connection confirmation
- Pool closes cleanly during graceful shutdown in correct sequence order
- DSN never appears in any log output — verified by log output inspection in tests
- All unit tests pass with `-race` flag
- Integration tests pass against `postulate_test` database
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
