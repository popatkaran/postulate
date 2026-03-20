# US-01-04 — Health, Readiness, Liveness, and Version Endpoints

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have

---

## 1. Story

As a **platform operator**, I need the API server to expose standard operational endpoints so that the load balancer, Kubernetes probes, and monitoring systems can verify the server's availability and operational state at any time.

---

## 2. Background

These endpoints serve distinct purposes and must not be conflated:

- `/health` — aggregate health check. Returns the overall health of the instance including the status of all critical dependencies. Used by monitoring and alerting systems.
- `/ready` — readiness probe. Indicates whether the instance is ready to receive traffic. Returns `503` if the server is starting up or shutting down. Used by the Kubernetes readiness probe.
- `/live` — liveness probe. Indicates whether the process is alive and not deadlocked. Returns `200` as long as the process is running. Used by the Kubernetes liveness probe.
- `/version` — build metadata. Returns the version, commit SHA, build time, and Go runtime version. Used for deployment verification and debugging.

All four endpoints are unauthenticated. They must respond even when the authentication system is unavailable.

At this Epic stage, the only dependency the health check can report on is the server itself. Subsequent Epics will register additional health contributors (database, cache, plugin registry) into the health check aggregator introduced here.

---

## 3. Acceptance Criteria

1. `GET /health` returns `200` with `Content-Type: application/json` when all registered health contributors report healthy. Returns `503` when any contributor reports unhealthy. Response body conforms to the schema defined in Section 4.1.
2. `GET /ready` returns `200` when the server is ready to receive traffic. Returns `503` during startup initialisation and graceful shutdown drain. Response body is a minimal JSON object: `{"status": "ready"}` or `{"status": "not_ready"}`.
3. `GET /live` returns `200` unconditionally as long as the process is running. Response body: `{"status": "alive"}`.
4. `GET /v1/version` returns `200` with build metadata conforming to the schema defined in Section 4.2.
5. All four endpoints respond within 100 milliseconds under normal conditions.
6. The health check aggregator supports registering named contributors at startup. Each contributor implements a `Check(ctx context.Context) HealthStatus` interface.
7. At this stage, one contributor is registered: `server` — always returns healthy while the server is running.
8. Version metadata is injected at build time via Go linker flags (`-ldflags`) — it must not be hardcoded in source.
9. Unit tests cover all response shapes including the unhealthy aggregate case.

---

## 4. Response Schemas

### 4.1 Health Response

```json
{
  "status": "healthy",
  "timestamp": "2026-03-19T10:00:00Z",
  "checks": {
    "server": {
      "status": "healthy",
      "message": ""
    },
    "database": {
      "status": "unhealthy",
      "message": "connection refused"
    }
  }
}
```

`status` at the root level is `healthy` only when all contributors are healthy. Otherwise `unhealthy`. The HTTP status code mirrors this — `200` for healthy, `503` for unhealthy.

### 4.2 Version Response

```json
{
  "version": "1.0.0",
  "commit": "a3f4c1d",
  "build_time": "2026-03-19T09:00:00Z",
  "go_version": "go1.26.1",
  "environment": "production"
}
```

---

## 5. Tasks

### Task 1 — Define the health contributor interface
- Create `api/internal/health/health.go`
- Define `Status` type as a string enum: `StatusHealthy`, `StatusUnhealthy`
- Define `CheckResult` struct: `Status Status`, `Message string`
- Define `Contributor` interface: `Name() string`, `Check(ctx context.Context) CheckResult`
- Define `Aggregator` struct that holds a map of registered contributors
- Implement `Register(contributor Contributor)` method on `Aggregator`
- Implement `Check(ctx context.Context) AggregateResult` method that calls all contributors and returns an aggregate

### Task 2 — Implement the server health contributor
- Create `api/internal/health/server_contributor.go`
- Implement `ServerContributor` struct implementing the `Contributor` interface
- `Name()` returns `"server"`
- `Check()` returns `StatusHealthy` unconditionally — this contributor only fails if the process itself is unable to run

### Task 3 — Implement the health handler
- Create `api/internal/handler/health_handler.go`
- Implement `HealthHandler` struct accepting `*health.Aggregator` via constructor
- `ServeHTTP` calls the aggregator, serialises the result, and returns `200` or `503` per acceptance criteria
- Implement `ReadyHandler` as a separate struct — tracks a `ready` boolean toggled by the server lifecycle
- Implement `LiveHandler` as a separate struct — always returns `200`

### Task 4 — Implement the version handler
- Create `api/internal/handler/version_handler.go`
- Define `BuildInfo` struct: `Version string`, `Commit string`, `BuildTime string`, `GoVersion string`, `Environment string`
- Implement `VersionHandler` struct accepting `BuildInfo` and `environment string` via constructor
- `ServeHTTP` serialises and returns the build info as JSON with status `200`
- Add a `Makefile` target `build` that injects version, commit, and build time via `-ldflags`

### Task 5 — Register handlers on the router
- Update `api/internal/router/router.go`
- Register `HealthHandler` on `GET /health`
- Register `ReadyHandler` on `GET /ready`
- Register `LiveHandler` on `GET /live`
- Register `VersionHandler` on `GET /v1/version`

### Task 6 — Wire health aggregator into main
- Update `api/cmd/api/main.go`
- Instantiate `health.Aggregator`
- Register `ServerContributor`
- Pass aggregator to `HealthHandler` constructor

### Task 7 — Unit tests
- Create `api/internal/health/health_test.go`
- Test: aggregate returns healthy when all contributors healthy
- Test: aggregate returns unhealthy when any contributor unhealthy
- Test: aggregate result includes all contributor names and statuses
- Create `api/internal/handler/health_handler_test.go`
- Test: handler returns `200` when aggregator is healthy
- Test: handler returns `503` when aggregator is unhealthy
- Test: response body matches defined schema
- Create `api/internal/handler/version_handler_test.go`
- Test: handler returns `200` with correct build info fields

---

## 6. Definition of Done

- All tasks completed
- All four endpoints respond correctly as verified by manual request and automated tests
- Build metadata is injected via linker flags — not hardcoded
- Health aggregator accepts additional contributors (verified by registering a test contributor in tests)
- All unit tests pass with `-race` flag
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
