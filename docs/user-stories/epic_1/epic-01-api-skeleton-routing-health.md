# Epic 01 — API Skeleton, Routing, and Health

**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** 1 of 20
**Phase:** 1 — Foundation

---

## 1. Purpose

This Epic establishes the foundational execution infrastructure for the Postulate SaaS platform. It produces a running, deployable Go API server with correct project structure, HTTP routing, configuration management, observability wiring, and all baseline health and operational endpoints.

No functional capability — generation, plugins, authentication, projects — can be built until this Epic is complete. Every subsequent Epic depends on the runtime, structure, and conventions defined here.

---

## 2. Context and Background

Postulate is an API-first SaaS platform. The CLI is a thin client. All logic — generation, enforcement, plugin execution — runs server-side. This Epic delivers the server that hosts all of that logic.

The decisions governing this Epic have been resolved and are recorded below. They must not be revisited without an Architecture Review.

| Decision | Resolution | Rationale |
|---|---|---|
| HTTP framework | Chi | Stdlib-compatible `http.Handler` throughout; does not constrain plugin-contributed middleware |
| Repository structure | Monorepo with Go workspaces | API and CLI share types; atomic cross-cutting changes; official plugins co-located and CI-tested together |
| Configuration source | YAML file with environment variable override | Supports both local development and containerised deployment without code changes |
| OTel scope | Postulate's own SaaS observability | Traces, metrics, and logs for the Postulate API itself — not for generated microservices |

---

## 3. Scope

### 3.1 In Scope

- Go monorepo initialisation with `go.work` workspace structure
- Chi HTTP server with full middleware chain
- `/health`, `/ready`, `/live`, `/version` endpoints
- `/v1/` routing prefix and versioning structure
- YAML configuration loading with environment variable override and startup validation
- Structured JSON logging using `slog` (stdlib)
- RFC 7807 problem details error response format
- Request middleware — request ID injection, structured access logging
- Graceful shutdown on `SIGTERM` and `SIGINT` with configurable drain timeout
- OpenTelemetry SDK wiring for traces, metrics, and logs (Postulate's own observability)
- Multi-stage, non-root Dockerfile

### 3.2 Out of Scope

- Authentication and session management (Epic 02)
- Database connectivity (Epic 02)
- Any plugin system components (Epic 04)
- CLI implementation (Epic 11)
- Generated microservice observability (plugin concern, Epic 12)

---

## 4. User Stories

| Story ID | Title | Priority |
|---|---|---|
| US-01-01 | Monorepo and Go Workspace Initialisation | Must Have |
| US-01-02 | Chi HTTP Server with Lifecycle Management | Must Have |
| US-01-03 | Configuration Loading and Startup Validation | Must Have |
| US-01-04 | Health, Readiness, Liveness, and Version Endpoints | Must Have |
| US-01-05 | Structured JSON Logging Foundation | Must Have |
| US-01-06 | RFC 7807 Error Response Format | Must Have |
| US-01-07 | Request Middleware — Request ID and Access Logging | Must Have |
| US-01-08 | Graceful Shutdown | Must Have |
| US-01-09 | OpenTelemetry SDK Wiring | Must Have |
| US-01-10 | Multi-Stage Non-Root Dockerfile | Must Have |
| US-01-11 | Epic 01 Coverage Closure and Pre-Epic-02 Hardening | Must Have |

---

## 5. Acceptance Criteria

The Epic is complete when all of the following are true:

1. `go work build ./...` completes without error across all workspace modules.
2. The API server starts, passes all health checks, and serves `GET /health` with a `200` response containing valid JSON.
3. `GET /v1/version` returns the current build version, commit SHA, and Go runtime version.
4. The server shuts down cleanly within the configured drain timeout when it receives `SIGTERM` — no in-flight requests are dropped.
5. All log output is structured JSON with the required fields: `timestamp`, `level`, `traceId`, `spanId`, `serviceId`, `instanceId`, `message`.
6. All error responses conform to RFC 7807 — `application/problem+json` content type with `type`, `title`, `status`, `detail`, and `instance` fields.
7. Every request log entry contains a unique `requestId` that is also returned in the `X-Request-ID` response header.
8. OTel traces are emitted for all inbound requests and exportable to a configured OTLP endpoint.
9. The Docker image builds successfully, runs as a non-root user, and the container passes its health check.
10. Minimum 90% unit test coverage across all packages introduced in this Epic, verified by `go test -coverprofile` — gate enforced in CI.

---

## 6. Technical Constraints

- Go version: 1.26 or later
- All packages must be structured under the `internal/` convention for non-exported code
- No `init()` functions — all initialisation must be explicit and testable
- No global mutable state — all dependencies injected via constructor
- Configuration must fail fast at startup if required values are absent or invalid

---

## 7. Dependencies

| Dependency | Type | Notes |
|---|---|---|
| None | — | This is the foundational Epic; it has no upstream dependencies |

---

## 8. Definition of Done

- All User Stories in this Epic are closed
- All acceptance criteria above are met
- Code has been reviewed and approved
- CI pipeline passes — lint, test, coverage gate, build, image scan
- Coverage gate verified by CI — `go test -coverprofile` shows ≥90% per package and in aggregate
- The running API is reachable and serves health endpoints in the target environment
