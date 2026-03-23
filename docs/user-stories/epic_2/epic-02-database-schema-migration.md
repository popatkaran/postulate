# Epic 02 — Database Schema and Migration Tooling

**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** 2 of 20
**Phase:** 1 — Foundation
**Depends on:** Epic 01 complete and closed

---

## 1. Purpose

This Epic establishes the persistence foundation for the Postulate SaaS platform. It delivers a production-grade PostgreSQL connection pool, a versioned migration system, the initial schema covering all entities required by Epics 01 through 03, a repository pattern that all subsequent data access code will follow, and a database health contributor wired into the operational health system introduced in Epic 01.

No feature Epic — authentication, generation, plugins, projects — can begin until this Epic is complete. Every entity those features read and write is defined here.

---

## 2. Context and Background

Postulate is API-first SaaS backed by a single PostgreSQL database. The database holds users, sessions, projects, generated service records, plugin registry entries, audit logs, and deviation workflows. This is a relational workload with transactional integrity requirements and native JSON storage needs for lockfile data.

The decisions governing this Epic have been resolved and are recorded below.

| Decision | Resolution | Rationale |
|---|---|---|
| Database engine | PostgreSQL 16 | Relational workload, JSONB for lockfiles, strong transactional guarantees, single engine across all environments |
| Connection pool | `pgxpool` (native pgx v5) | Postulate never switches databases — portability of `database/sql` is not needed; `pgxpool` gives better performance and full PostgreSQL type support |
| Migration tool | `golang-migrate` | SQL-first migrations, language-agnostic files, strong CLI, widely adopted in Go projects |
| Schema scope | Minimal — users, sessions, refresh tokens | Tables created when the feature that owns them is built; no speculative schema |
| Local development | Native PostgreSQL via Homebrew (US-02-00) | No Docker dependency for local development; same engine as production |

---

## 3. Scope

### 3.1 In Scope

- `pgxpool` connection pool initialisation and lifecycle management
- Connection pool configuration: max open connections, max idle connections, connection max lifetime
- Graceful shutdown integration — pool closed after HTTP server drains
- Database health contributor wired into the Epic 01 health aggregator
- `golang-migrate` integration — migration runner invoked at startup, `make migrate-*` CLI targets, CI enforcement
- Initial schema — `users`, `sessions`, `refresh_tokens` tables with indexes and constraints
- Repository pattern foundation — base interfaces, transaction support, context propagation
- Integration test harness using `postulate_test` database

### 3.2 Out of Scope

- Authentication business logic (Epic 03)
- Session management logic (Epic 03)
- Any schema beyond users, sessions, and refresh tokens
- ORM or query builder — raw SQL with `pgx` only
- Read replicas or connection routing
- Database backup and restore procedures (infrastructure concern)

---

## 4. User Stories

| Story ID | Title | Priority |
|---|---|---|
| US-02-00 | Local PostgreSQL Setup via Homebrew | Must Have |
| US-02-01 | pgxpool Connection Pool and Lifecycle | Must Have |
| US-02-02 | Database Health Contributor | Must Have |
| US-02-03 | golang-migrate Integration and Migration Tooling | Must Have |
| US-02-04 | Initial Schema — Users, Sessions, Refresh Tokens | Must Have |
| US-02-05 | Repository Pattern Foundation | Must Have |

---

## 5. Acceptance Criteria

The Epic is complete when all of the following are true:

1. The API server starts, connects to PostgreSQL, and runs all pending migrations automatically before accepting requests.
2. `GET /health` reflects database connectivity — returns `503` when PostgreSQL is unreachable.
3. `GET /ready` returns `200` only after the connection pool is established and all migrations have run.
4. The connection pool is closed cleanly during graceful shutdown — no connections are leaked.
5. Running `make migrate-up` applies all pending migrations. Running `make migrate-down` rolls back the most recent migration. Running `make migrate-status` shows applied and pending migrations.
6. The `users`, `sessions`, and `refresh_tokens` tables exist in `postulate_dev` after `make migrate-up`.
7. All repository interfaces have concrete implementations testable against `postulate_test`.
8. `database.password` never appears in any log output — verified by tests.
9. Minimum 90% unit and integration test coverage across all packages introduced in this Epic.
10. `make migrate-up` is idempotent — running it twice produces no error and no duplicate migrations.

---

## 6. Technical Constraints

- All database queries use `pgxpool` directly — no ORM, no query builder
- All queries must accept `context.Context` as their first argument
- No raw SQL outside of repository implementations and migration files
- Migration files are plain SQL — no Go migration functions
- All migration files follow the naming convention: `{version}_{description}.up.sql` and `{version}_{description}.down.sql`
- Migrations are numbered sequentially starting from `000001`
- No `SELECT *` — all queries must name their columns explicitly

---

## 7. Dependencies

| Dependency | Type | Notes |
|---|---|---|
| Epic 01 — closed | Hard | Configuration, logger, health aggregator, graceful shutdown all required |
| US-02-00 | Hard | PostgreSQL must be running locally before any other story can be developed or tested |

---

## 8. Definition of Done

- All User Stories in this Epic are closed
- All acceptance criteria above are met
- Code has been reviewed and approved
- CI pipeline passes — lint, unit tests, integration tests, coverage gate, migration idempotency check
- `make migrate-up` runs cleanly against a fresh database in CI
- Running API connects to PostgreSQL, runs migrations, and serves all health endpoints correctly
