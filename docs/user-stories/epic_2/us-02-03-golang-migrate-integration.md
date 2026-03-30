# US-02-03 — golang-migrate Integration and Migration Tooling

**Epic:** Epic 02 — Database Schema and Migration Tooling
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have
**Depends on:** US-02-01 (pool available)

---

## 1. Story

As a **platform engineer**, I need `golang-migrate` integrated into the API startup sequence and available as Makefile targets so that database schema changes are versioned, applied automatically on deployment, and reversible without manual SQL execution.

---

## 2. Background

`golang-migrate` applies versioned SQL migration files in sequence. Each migration has an `up` file (applies the change) and a `down` file (reverses it). The tool tracks which migrations have been applied in a `schema_migrations` table it manages in the target database.

Migrations run automatically at API startup — before the server begins accepting requests. This ensures the schema is always in sync with the deployed code. If a migration fails at startup, the server exits with code 1 and logs the failing migration file name and error.

Migration files live in `api/migrations/` and are embedded into the Go binary using `go:embed`. This means the production binary is self-contained — no separate migration file deployment is required.

The Makefile provides developer-facing targets for migration operations during local development. These targets operate against `postulate_dev` by default and accept a `DB` environment variable override to target other databases.

---

## 3. Acceptance Criteria

1. `golang-migrate` is added as a dependency and migration files are embedded into the binary via `go:embed`.
2. Migrations run automatically at API startup, after the connection pool is established and the database ping succeeds, before the HTTP server begins accepting requests.
3. If a migration fails at startup, the server logs the migration version number and filename, the error, and exits with code 1.
4. If no pending migrations exist at startup, the server logs `INFO "database schema up to date"` and continues normally.
5. The following Makefile targets are operational:

   | Target | Behaviour |
   |---|---|
   | `make migrate-up` | Applies all pending migrations to `postulate_dev` |
   | `make migrate-down` | Rolls back the most recent applied migration |
   | `make migrate-down-all` | Rolls back all applied migrations (prompts for confirmation) |
   | `make migrate-status` | Lists all migrations with applied/pending status and applied timestamp |
   | `make migrate-create name=<name>` | Creates a new numbered migration file pair in `api/migrations/` |
   | `make migrate-version` | Prints the current schema version |

6. `make migrate-up` is idempotent — running it twice against the same database produces no error and no duplicate applications.
7. Migration files follow the naming convention `{version}_{description}.up.sql` and `{version}_{description}.down.sql` where `version` is a zero-padded 6-digit integer: `000001`, `000002`, etc.
8. Every migration file pair must have both an `up` and a `down` file. A migration with only an `up` file fails the CI lint step.
9. The `schema_migrations` table is created automatically by `golang-migrate` on first run — no manual setup required.
10. CI pipeline runs `make migrate-up` against a fresh `postulate_test` database and verifies it exits with code 0.
11. Migration files are plain SQL — no Go migration functions.

---

## 4. Migration File Structure

```
api/
  migrations/
    000001_create_users.up.sql
    000001_create_users.down.sql
    000002_create_sessions.up.sql
    000002_create_sessions.down.sql
    000003_create_refresh_tokens.up.sql
    000003_create_refresh_tokens.down.sql
  internal/
    migrate/
      migrate.go         # migration runner
      embed.go           # go:embed declaration
```

---

## 5. Startup Sequence Update

After this story, the full startup sequence in `main` is:

1. Load and validate configuration
2. Initialise logger
3. Initialise OTel providers
4. Construct database connection pool (`pgxpool.New`)
5. Ping database (startup guard)
6. **Run pending migrations** ← introduced in this story
7. Register health contributors
8. Construct router and middleware chain
9. Start HTTP server
10. Block until shutdown signal

---

## 6. Tasks

### Task 1 — Add golang-migrate dependency
- Add `github.com/golang-migrate/migrate/v4` to `api/go.mod`
- Add the `pgx/v5` driver adapter: `github.com/golang-migrate/migrate/v4/database/pgx/v5`
- Add the `iofs` source driver: `github.com/golang-migrate/migrate/v4/source/iofs`
- Run `go mod tidy`

### Task 2 — Implement the migration embed and runner
- Create `api/internal/migrate/embed.go`:
  ```go
  package migrate

  import "embed"

  //go:embed ../../migrations/*.sql
  var MigrationFiles embed.FS
  ```
- Create `api/internal/migrate/migrate.go`:
  - Implement `Run(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error`
  - Construct an `iofs` source from `MigrationFiles`
  - Construct a `pgx/v5` database driver from the pool
  - Instantiate `migrate.NewWithInstance`
  - Call `m.Up()` — handle `migrate.ErrNoChange` as a non-error (log `INFO "database schema up to date"`)
  - On error: log `ERROR` with migration version and filename, return the error
  - On success: log `INFO "database migrations applied"` with the count of applied migrations

### Task 3 — Wire migration runner into main
- Update `api/cmd/api/main.go`
- Call `migrate.Run(ctx, pool, logger)` at step 6 per Section 5
- On error, exit with code 1

### Task 4 — Create migrations directory
- Create `api/migrations/.gitkeep` to ensure the directory is committed before migration files are added
- Create a `README.md` in `api/migrations/` documenting the naming convention and the process for creating a new migration using `make migrate-create`

### Task 5 — Makefile migration targets
- Add all six targets from acceptance criteria to the root `Makefile`
- Targets use `golang-migrate` CLI — add `make install-tools` target that installs the CLI via `go install`
- `migrate-create` target validates that `name` argument is provided — prints usage and exits 1 if absent
- `migrate-down-all` uses `read -p` to prompt for confirmation
- Each target accepts `DB_URL` environment variable override — defaults to the value constructed from `config.local.yaml`
- Document all targets in `CONTRIBUTING.md` under a "Database Migrations" section

### Task 6 — CI migration check
- Add a `migrate-ci` step to the CI pipeline configuration:
  - Start a PostgreSQL service container in CI
  - Run `make migrate-up DB_URL=<ci-db-url>`
  - Verify exit code is 0
  - Run `make migrate-up` a second time — verify it is idempotent (exit code 0, `ErrNoChange`)
  - Run `make migrate-down-all` — verify exit code is 0
- Add a migration file lint step: fail CI if any `.up.sql` file exists without a corresponding `.down.sql` file

### Task 7 — Unit tests
- Create `api/internal/migrate/migrate_test.go`
- Test: `Run` with `ErrNoChange` logs `"database schema up to date"` and returns nil
- Test: `Run` with a successful migration logs `"database migrations applied"` and returns nil
- Test: `Run` with a migration error logs the error and returns the error
- Use interface mocking for the migrate instance — no real database for unit tests

### Task 8 — Integration test
- Create `api/internal/migrate/migrate_integration_test.go` with `//go:build integration` tag
- Test: `Run` against a fresh `postulate_test` database applies all migrations and returns nil
- Test: calling `Run` a second time returns nil (idempotent)
- After test, run all down migrations to leave the test database clean for other integration tests

---

## 7. Definition of Done

- All tasks completed
- API starts, applies pending migrations, and logs the result before accepting requests
- `make migrate-up` applies all migrations against `postulate_dev`
- `make migrate-down` rolls back the most recent migration
- `make migrate-status` shows correct applied/pending state
- `make migrate-up` is idempotent — verified by running twice
- CI pipeline runs migration check successfully
- All unit tests pass with `-race` flag
- Integration tests pass against `postulate_test` database
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
