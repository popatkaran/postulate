# US-02-02 — Database Health Contributor

**Epic:** Epic 02 — Database Schema and Migration Tooling
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have
**Depends on:** US-02-01 (pool available), US-01-04 (health aggregator available)

---

## 1. Story

As a **platform operator**, I need the `/health` endpoint to reflect the real-time connectivity status of the PostgreSQL database so that monitoring systems, load balancers, and on-call engineers can detect database connectivity failures immediately from the standard health endpoint.

---

## 2. Background

The health aggregator introduced in US-01-04 accepts named `Contributor` implementations. The `ServerContributor` registered in Epic 01 always returns healthy. This story introduces `DatabaseContributor` — the second contributor — which performs a lightweight database ping on each health check call and reports the result.

A health check that simply returns healthy without checking the database provides false confidence. If PostgreSQL becomes unreachable, the API will return `500` errors on every data-accessing endpoint. The health endpoint must reflect this state so that the load balancer can route traffic away from the affected instance and alerting fires promptly.

The health check ping must be lightweight — it must not acquire a full connection from the pool for the duration of the request. `pgxpool.Ping` uses an idle connection from the pool and returns it immediately after the ping completes.

A timeout is applied to every health check ping. If the database does not respond within the timeout, the contributor returns unhealthy — a slow database is operationally equivalent to an unreachable one for health check purposes.

---

## 3. Acceptance Criteria

1. `GET /health` returns `503` with `"status": "unhealthy"` and a `"database"` entry in the `checks` map when the database is unreachable or the ping times out.
2. `GET /health` returns `200` with `"status": "healthy"` and a `"database"` entry showing `"status": "healthy"` when the database is reachable.
3. The database health check ping uses a 2-second timeout context — if the ping does not complete within 2 seconds, the contributor returns unhealthy with message `"ping timeout"`.
4. The `CheckResult.Message` field for a database connectivity failure contains the error string — e.g. `"dial tcp: connection refused"`. This aids on-call diagnosis without requiring log access.
5. The `DatabaseContributor` is registered with the health aggregator in `main` immediately after the pool is constructed.
6. The contributor is named `"database"` — this is the key that appears in the `checks` map of the health response.
7. The database health check does not appear in access logs — `/health` is already excluded from access logging per US-01-07. No additional exclusion is needed.
8. Pool statistics are included in the health check response as an extension field when the database is healthy:
   ```json
   "database": {
     "status": "healthy",
     "message": "",
     "stats": {
       "acquired_conns": 2,
       "idle_conns": 3,
       "max_conns": 25
     }
   }
   ```
9. Pool statistics are omitted from the response when the database is unhealthy — only `status` and `message` are returned.
10. Unit tests cover: healthy ping returns healthy status with stats, failed ping returns unhealthy status with error message, ping timeout returns unhealthy with `"ping timeout"` message.

---

## 4. Health Response Shape Update

The `CheckResult` struct introduced in US-01-04 must be extended to support optional extension data. The current shape:

```go
type CheckResult struct {
    Status  Status
    Message string
}
```

Must be extended to:

```go
type CheckResult struct {
    Status     Status
    Message    string
    Extensions map[string]any  // optional — omitted from JSON if nil
}
```

This extension mechanism allows any contributor to attach structured metadata to its health result without modifying the core health interfaces. The `DatabaseContributor` uses it for pool statistics. Future contributors (cache, plugin registry) may use it for their own metadata.

The `HealthHandler` must be updated to include `Extensions` in the JSON response when non-nil.

---

## 5. Tasks

### Task 1 — Extend CheckResult with Extensions field
- Update `api/internal/health/health.go`
- Add `Extensions map[string]any` to `CheckResult` struct with `json:"extensions,omitempty"` tag
- Update `HealthHandler` in `api/internal/handler/health_handler.go` to serialise `Extensions` when non-nil
- Update existing `ServerContributor` tests to verify the `Extensions` field is absent (nil) in its result — no behaviour change, just confirm the omitempty tag works

### Task 2 — Implement DatabaseContributor
- Create `api/internal/health/database_contributor.go`
- Define `DatabaseContributor` struct accepting a `*pgxpool.Pool` via constructor
- Implement `Name() string` — returns `"database"`
- Implement `Check(ctx context.Context) CheckResult`:
  - Create a child context with a 2-second timeout
  - Call `pool.Ping(pingCtx)`
  - On success: return `StatusHealthy` with `Extensions` containing pool stats from `pool.Stat()`
  - On timeout: return `StatusUnhealthy` with message `"ping timeout"`
  - On other error: return `StatusUnhealthy` with message set to `err.Error()`

### Task 3 — Define pool stats extension shape
- Create `api/internal/health/pool_stats.go`
- Define `PoolStats` struct: `AcquiredConns int32`, `IdleConns int32`, `MaxConns int32`
- Implement `poolStatsFromPgx(stat *pgxpool.Stat) PoolStats` helper
- Use this struct as the value under the `"stats"` key in `Extensions`

### Task 4 — Register DatabaseContributor in main
- Update `api/cmd/api/main.go`
- After pool construction, instantiate `health.NewDatabaseContributor(pool)`
- Register it with the health aggregator: `healthAggregator.Register(dbContributor)`

### Task 5 — Unit tests
- Create `api/internal/health/database_contributor_test.go`
- Use `pgxmock` to mock the pool interface for unit tests — no real database required
- Test: `Check` returns `StatusHealthy` when ping succeeds — verify `Extensions` contains pool stats
- Test: `Check` returns `StatusUnhealthy` with error message when ping fails
- Test: `Check` returns `StatusUnhealthy` with `"ping timeout"` message when ping exceeds 2-second timeout — use a mock that blocks then use a fast-expiring context
- Test: `Name()` returns `"database"`
- Update `api/internal/handler/health_handler_test.go`:
  - Test: response body includes `extensions` field when contributor returns non-nil Extensions
  - Test: response body omits `extensions` field when contributor returns nil Extensions

### Task 6 — Integration test
- Create `api/internal/health/database_contributor_integration_test.go` with `//go:build integration` tag
- Test: `Check` against `postulate_test` database returns `StatusHealthy`
- Test: health response from a running test server includes `"database": {"status": "healthy"}` in the checks map

---

## 6. Definition of Done

- All tasks completed
- `GET /health` returns `503` when PostgreSQL is stopped — verified manually with `make db-stop`
- `GET /health` returns `200` with database stats when PostgreSQL is running
- `CheckResult` extension mechanism works for nil and non-nil extensions — verified by tests
- All unit tests pass with `-race` flag
- Integration tests pass against `postulate_test` database
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
