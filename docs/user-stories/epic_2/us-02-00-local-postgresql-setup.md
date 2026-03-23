# US-02-00 — Local PostgreSQL Setup via Homebrew

**Epic:** Epic 02 — Database Schema and Migration Tooling
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have — Epic 02 prerequisite
**Sequencing:** Must be complete before any other Epic 02 story begins

---

## 1. Story

As a **platform engineer setting up a local development environment**, I need clear, scripted instructions to install and run PostgreSQL locally via Homebrew so that I can develop and test the Postulate API against the same database engine used in production without requiring Docker or any container runtime.

---

## 2. Background

Postulate uses PostgreSQL exclusively. Using any other database engine locally — including SQLite — is not supported. The SQL dialect, transaction behaviour, constraint enforcement, and native types used throughout the codebase are PostgreSQL-specific and will not function correctly against any other engine.

This story covers local macOS development via Homebrew. Linux native installation instructions are included as a secondary section. Windows is not a supported local development platform for this project — engineers on Windows should use WSL2 with the Linux path.

This story also introduces a startup guard into the API server: if the database is unreachable at startup, the server logs a clear, actionable error and exits rather than starting in a degraded state that produces confusing errors at request time.

The database credentials configured here are for local development only. They are never used in staging or production environments, where credentials are injected via secrets management.

---

## 3. Acceptance Criteria

### Local Installation

1. Following the setup instructions in `docs/local-setup.md` from a clean macOS machine with Homebrew installed produces a running PostgreSQL 16 instance.
2. The following are created as part of setup and documented in the setup script:
   - PostgreSQL role: `postulate_dev` with password `postulate_dev`
   - Database: `postulate_dev` owned by `postulate_dev`
   - Database: `postulate_test` owned by `postulate_dev` (used for test runs)
3. `psql -U postulate_dev -d postulate_dev -c "SELECT version();"` executes successfully after setup.

### Makefile Targets

4. The root `Makefile` contains the following operational targets:
   - `make db-start` — starts the PostgreSQL service via `brew services start`
   - `make db-stop` — stops the PostgreSQL service via `brew services stop`
   - `make db-status` — prints the current service status
   - `make db-setup` — runs the setup script to create roles, databases, and local config file
   - `make db-reset` — drops and recreates `postulate_dev` and `postulate_test` databases (development only — must prompt for confirmation before executing)
5. Each target prints a human-readable status message on completion.
6. `make db-start` is documented in `CONTRIBUTING.md` as a required step before running the API locally.

### Local Configuration

7. `make db-setup` creates `api/config.local.yaml` if it does not already exist, populated with the local development database connection values. This file is listed in `.gitignore` and must never be committed.
8. `api/config.example.yaml` is updated to include the database configuration section with all fields documented:

   ```yaml
   database:
     host: localhost           # required
     port: 5432                # required — integer
     name: postulate_dev       # required
     user: postulate_dev       # required
     password: ""              # required — sensitive — never log
     ssl_mode: disable         # required — disable | require | verify-full
     max_open_conns: 25        # optional — default 25
     max_idle_conns: 5         # optional — default 5
     conn_max_lifetime_seconds: 300   # optional — default 300
   ```

9. The `config.DatabaseConfig` struct is added to `api/internal/config/config.go` with all fields above and correct `yaml` struct tags.
10. `config.Validate` is extended to validate the new database fields:
    - `host` — non-empty string
    - `port` — integer 1–65535
    - `name` — non-empty string
    - `user` — non-empty string
    - `password` — non-empty string
    - `ssl_mode` — one of `disable`, `require`, `verify-full`
11. `config.LogSafe` designates `database.password` as sensitive — it must appear as `[redacted]` in all log output. The `// SECURITY:` comment introduced in US-01-11 is updated to reflect this.

### API Startup Guard

12. The API server performs a database reachability check during startup, after configuration is loaded and validated but before the server begins accepting requests.
13. If the database is unreachable at startup, the server logs the following at `ERROR` level and exits with code 1:
    ```
    database unreachable at startup — ensure PostgreSQL is running
    host=localhost port=5432 name=postulate_dev
    hint: run 'make db-start' to start the local PostgreSQL service
    ```
14. The `hint` field in the log entry is only emitted when `server.environment` is `development`. It is omitted in `staging` and `production`.
15. The readiness probe (`/ready`) returns `503` until the startup database check passes. It transitions to `200` only after all startup checks succeed.
16. The database health contributor introduced in US-02-02 plugs into this same reachability mechanism — the startup guard and the health check share the same underlying probe function.

### Documentation

17. `docs/local-setup.md` is created covering:
    - Prerequisites — Homebrew, Go 1.22+
    - PostgreSQL installation via `brew install postgresql@16`
    - Running `make db-setup` to create roles and databases
    - Running `make db-start` to start the service
    - Copying `api/config.example.yaml` to `api/config.local.yaml` and setting `POSTULATE_CONFIG_FILE`
    - Running `make run` to start the API
    - Verifying the API is running with `curl http://localhost:8080/health`
    - Troubleshooting section — PostgreSQL not running, role does not exist, wrong password

---

## 4. Setup Script

The following shell script is created at `scripts/db-setup.sh` and invoked by `make db-setup`.

```bash
#!/usr/bin/env bash
# scripts/db-setup.sh
# Creates local PostgreSQL roles and databases for Postulate development.
# Safe to run multiple times — uses IF NOT EXISTS throughout.

set -euo pipefail

DB_USER="postulate_dev"
DB_PASS="postulate_dev"
DB_NAME_DEV="postulate_dev"
DB_NAME_TEST="postulate_test"

echo "→ Creating role: ${DB_USER}"
psql postgres -tc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" \
  | grep -q 1 \
  || psql postgres -c "CREATE ROLE ${DB_USER} WITH LOGIN PASSWORD '${DB_PASS}';"

echo "→ Creating database: ${DB_NAME_DEV}"
psql postgres -tc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME_DEV}'" \
  | grep -q 1 \
  || psql postgres -c "CREATE DATABASE ${DB_NAME_DEV} OWNER ${DB_USER};"

echo "→ Creating database: ${DB_NAME_TEST}"
psql postgres -tc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME_TEST}'" \
  | grep -q 1 \
  || psql postgres -c "CREATE DATABASE ${DB_NAME_TEST} OWNER ${DB_USER};"

echo "→ Writing api/config.local.yaml"
if [ ! -f api/config.local.yaml ]; then
  cp api/config.example.yaml api/config.local.yaml
  echo "   Created api/config.local.yaml — update values as needed"
else
  echo "   api/config.local.yaml already exists — skipping"
fi

echo ""
echo "✓ Local database setup complete."
echo "  Start PostgreSQL with: make db-start"
echo "  Then start the API with: make run"
```

---

## 5. Tasks

### Task 1 — Setup script and Makefile targets
- Create `scripts/db-setup.sh` per Section 4
- Make script executable: `chmod +x scripts/db-setup.sh`
- Add `db-setup`, `db-start`, `db-stop`, `db-status`, and `db-reset` targets to root `Makefile`
- `db-reset` must use `read -p` to prompt for confirmation before dropping databases
- Add `scripts/` directory to `.gitignore` exclusions list — except the setup script itself which must be committed

### Task 2 — Extend configuration struct and validation
- Add `DatabaseConfig` struct to `api/internal/config/config.go` per acceptance criteria
- Add `Database DatabaseConfig` field to the root `Config` struct
- Extend `config.Validate` with database field validation rules
- Extend `config.LogSafe` to redact `database.password`
- Update the `// SECURITY:` comment in `redact.go` to document `database.password` as a sensitive field

### Task 3 — Implement the startup database reachability check
- Create `api/internal/startup/checks.go`
- Implement `CheckDatabase(ctx context.Context, cfg config.DatabaseConfig, logger *slog.Logger) error`
- The function opens a temporary connection using `pgxpool`, calls `Ping`, and closes the connection
- On failure, log the structured error with `host`, `port`, `name` fields and conditionally the `hint` field
- Wire into `api/cmd/api/main.go` — call after configuration validation, before server start
- On error, exit with code 1
- Update `ReadyHandler` — set `ready = false` until `CheckDatabase` returns nil

### Task 4 — Update configuration example and gitignore
- Update `api/config.example.yaml` with the `database` section per acceptance criteria
- Add `api/config.local.yaml` to `.gitignore`
- Add `api/config.local.yaml` to `.dockerignore`

### Task 5 — Create local setup documentation
- Create `docs/local-setup.md` covering all items in acceptance criteria point 17
- Update `CONTRIBUTING.md` to reference `docs/local-setup.md` for environment setup
- Update the root `README.md` to link to `docs/local-setup.md` under a "Getting Started" section

### Task 6 — Linux installation instructions
- Add a "Linux (Ubuntu/Debian)" section to `docs/local-setup.md`:
  - `sudo apt install postgresql-16`
  - `sudo systemctl start postgresql`
  - Role and database creation using `sudo -u postgres psql`
- Note that `make db-start` and `make db-stop` use `brew services` and are macOS-only — Linux engineers should use `systemctl` directly

### Task 7 — Unit tests
- Extend `api/internal/config/validation_test.go`:
  - Test: valid database config passes validation
  - Test: missing `host` fails validation with named error
  - Test: invalid `ssl_mode` value fails validation
  - Test: missing `password` fails validation
- Extend `api/internal/config/redact_test.go`:
  - Test: `database.password` appears as `[redacted]` in `LogSafe` output
  - Test: `database.user` is not redacted
- Create `api/internal/startup/checks_test.go`:
  - Test: `CheckDatabase` returns nil when database is reachable — use a real local test database (`postulate_test`) in integration test tagged with `//go:build integration`
  - Test: `CheckDatabase` returns a descriptive error when the host is unreachable — use an invalid host to trigger a connection failure without a real database

---

## 6. Linux Quick Reference

For engineers not on macOS, the equivalent steps without Homebrew:

```bash
# Ubuntu / Debian
sudo apt update && sudo apt install -y postgresql-16
sudo systemctl enable postgresql
sudo systemctl start postgresql

# Create role and databases
sudo -u postgres psql -c "CREATE ROLE postulate_dev WITH LOGIN PASSWORD 'postulate_dev';"
sudo -u postgres psql -c "CREATE DATABASE postulate_dev OWNER postulate_dev;"
sudo -u postgres psql -c "CREATE DATABASE postulate_test OWNER postulate_dev;"
```

---

## 7. Definition of Done

- All tasks completed
- `make db-setup` runs without error on a clean machine and creates the expected roles and databases
- `make db-start` and `make db-stop` start and stop the PostgreSQL service
- API exits with code 1 and a clear error message when PostgreSQL is not running at startup
- API starts successfully and `/health` returns `200` when PostgreSQL is running
- `database.password` never appears in log output — verified by `LogSafe` tests
- `docs/local-setup.md` reviewed by a second engineer following the instructions from scratch
- All unit tests pass with `-race` flag
- Integration test tagged and excluded from default `make test` run — runs only with `make test-integration`
- `make lint` passes with zero issues
- PR reviewed and approved
