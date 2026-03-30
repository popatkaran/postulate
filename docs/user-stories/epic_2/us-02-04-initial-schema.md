# US-02-04 — Initial Schema — Users, Sessions, Refresh Tokens

**Epic:** Epic 02 — Database Schema and Migration Tooling
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have
**Depends on:** US-02-03 (migration tooling in place)

---

## 1. Story

As a **platform engineer**, I need the initial database schema defining users, sessions, and refresh tokens so that Epic 03 (Authentication) has a correctly structured, indexed, and constrained persistence layer to build against.

---

## 2. Background

This story creates three migration file pairs. The schema covers only what Epic 03 requires — no speculative tables. Each table is designed with the following principles:

- UUIDs for all primary keys — no sequential integer IDs exposed externally
- `created_at` and `updated_at` timestamps on every table — UTC, not nullable
- Soft delete via `deleted_at` where appropriate — hard delete only for session and token cleanup
- All constraints enforced at the database level, not only in application code
- Indexes created for every foreign key and every column used in a WHERE clause

The schema is intentionally minimal. Columns will be added in later Epics via new migration files — not by modifying these migrations.

---

## 3. Schema Definitions

### 3.1 `users` Table

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    password_hash   TEXT NOT NULL,
    full_name       TEXT NOT NULL DEFAULT '',
    role            TEXT NOT NULL DEFAULT 'member',
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_role_check CHECK (role IN ('member', 'admin', 'platform_admin')),
    CONSTRAINT users_status_check CHECK (status IN ('active', 'suspended', 'pending_verification'))
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_status ON users (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NOT NULL;
```

### 3.2 `sessions` Table

```sql
CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL,
    ip_address      INET,
    user_agent      TEXT NOT NULL DEFAULT '',
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ,

    CONSTRAINT sessions_token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_token_hash ON sessions (token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at) WHERE revoked_at IS NULL;
```

### 3.3 `refresh_tokens` Table

```sql
CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT refresh_tokens_token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX idx_refresh_tokens_session_id ON refresh_tokens (session_id);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at) WHERE used_at IS NULL;
```

---

## 4. Down Migrations

Every `up` migration has a corresponding `down` migration that fully reverses it. Down migrations drop in reverse dependency order.

**`000001_create_users.down.sql`**
```sql
DROP TABLE IF EXISTS users CASCADE;
```

**`000002_create_sessions.down.sql`**
```sql
DROP TABLE IF EXISTS sessions CASCADE;
```

**`000003_create_refresh_tokens.down.sql`**
```sql
DROP TABLE IF EXISTS refresh_tokens CASCADE;
```

---

## 5. Acceptance Criteria

1. Three migration file pairs are created in `api/migrations/`:
   - `000001_create_users.up.sql` / `000001_create_users.down.sql`
   - `000002_create_sessions.up.sql` / `000002_create_sessions.down.sql`
   - `000003_create_refresh_tokens.up.sql` / `000003_create_refresh_tokens.down.sql`
2. All SQL in the up migrations matches the schema definitions in Section 3 exactly.
3. `make migrate-up` against a fresh `postulate_dev` database creates all three tables with correct columns, constraints, and indexes.
4. `make migrate-down` followed by `make migrate-up` applied three times restores the full schema without error.
5. `make migrate-down-all` drops all three tables cleanly.
6. `\d users`, `\d sessions`, and `\d refresh_tokens` in `psql` show all columns, constraints, and indexes as defined.
7. The following constraints are verified by attempting violating inserts:
   - `users.email` uniqueness constraint
   - `users.role` check constraint — insert with role `"superuser"` must fail
   - `users.status` check constraint — insert with status `"deleted"` must fail
   - `sessions.user_id` foreign key — insert with non-existent `user_id` must fail
   - `refresh_tokens.session_id` foreign key — insert with non-existent `session_id` must fail
   - `sessions` cascade delete — deleting a `user` must cascade-delete their sessions
   - `refresh_tokens` cascade delete — deleting a `session` must cascade-delete its refresh tokens
8. Migration files contain no application logic — only DDL statements.
9. No `SELECT`, `INSERT`, `UPDATE`, or `DELETE` statements appear in any migration file.

---

## 6. Design Notes

**Why UUIDs not serial integers?** Session tokens, user IDs, and service record IDs are exposed in API responses and URLs. Sequential integers leak record count information and are trivially enumerable. UUIDs are neither.

**Why `gen_random_uuid()` not `uuid_generate_v4()`?** `gen_random_uuid()` is built into PostgreSQL 13+. It requires no extension. `uuid_generate_v4()` requires the `uuid-ossp` extension.

**Why `TIMESTAMPTZ` not `TIMESTAMP`?** `TIMESTAMPTZ` stores UTC and handles timezone conversion correctly. `TIMESTAMP` has no timezone awareness and creates subtle bugs when the server timezone differs from UTC.

**Why `token_hash` not `token`?** Session tokens and refresh tokens are credentials. Storing a hash means a database breach does not directly expose valid tokens. The application stores the hash, verifies by hashing the incoming token and comparing.

**Why soft delete on `users` but hard delete on sessions and tokens?** Users may need to be recovered (accidental deletion, GDPR erasure requests that are reversed). Sessions and tokens are ephemeral — a revoked or expired session has no recovery value and should be purged by a cleanup job.

---

## 7. Tasks

### Task 1 — Create users migration
- Create `api/migrations/000001_create_users.up.sql` with the DDL from Section 3.1
- Create `api/migrations/000001_create_users.down.sql` with the DDL from Section 4
- Verify `make migrate-up` applies the migration cleanly against `postulate_dev`

### Task 2 — Create sessions migration
- Create `api/migrations/000002_create_sessions.up.sql` with the DDL from Section 3.2
- Create `api/migrations/000002_create_sessions.down.sql` with the DDL from Section 4
- Verify the migration applies after `000001`

### Task 3 — Create refresh tokens migration
- Create `api/migrations/000003_create_refresh_tokens.up.sql` with the DDL from Section 3.3
- Create `api/migrations/000003_create_refresh_tokens.down.sql` with the DDL from Section 4
- Verify all three migrations apply in sequence

### Task 4 — Verify constraints manually
- Connect to `postulate_dev` via `psql`
- Execute each constraint violation test from acceptance criteria point 7
- Document the results in the PR description

### Task 5 — Integration tests
- Create `api/migrations/migrations_integration_test.go` with `//go:build integration` tag
- Test: all three tables exist after `migrate-up`
- Test: all expected columns exist on each table with correct types — query `information_schema.columns`
- Test: all expected indexes exist — query `pg_indexes`
- Test: cascade delete from `users` removes associated `sessions` and `refresh_tokens`
- Test: `users.email` uniqueness constraint — second insert with same email returns `pgconn.PgError` with code `23505`
- Test: `users.role` check constraint — insert with invalid role returns `pgconn.PgError` with code `23514`

### Task 6 — Update migration README
- Update `api/migrations/README.md` to document each migration file, the table it creates, and a brief rationale for key design decisions (FK relationships, index strategy, soft vs hard delete choices)

---

## 8. Definition of Done

- All tasks completed
- All three migration files created and committed
- `make migrate-up` → `make migrate-down-all` → `make migrate-up` cycle completes without error
- All constraints verified by tests
- Integration tests pass against `postulate_test` database
- Migration README updated
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
