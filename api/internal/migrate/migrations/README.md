# Migrations

SQL migration files managed by [golang-migrate](https://github.com/golang-migrate/migrate).
Files are embedded into the API binary at build time — no separate deployment step required.

## Naming Convention

```
{version}_{description}.up.sql
{version}_{description}.down.sql
```

- `version` is a zero-padded 6-digit integer: `000001`, `000002`, etc.
- Every migration **must** have both an `.up.sql` and a `.down.sql` file.
- A migration with only an `.up.sql` file will fail the CI lint step.
- Migration files contain DDL only — no `SELECT`, `INSERT`, `UPDATE`, or `DELETE`.

## Migration Files

| Version | Description | Tables |
|---------|-------------|--------|
| `000001` | `create_users` | `users` — primary identity table with soft delete |
| `000002` | `create_sessions` | `sessions` — active login sessions, FK → `users` |
| `000003` | `create_refresh_tokens` | `refresh_tokens` — token rotation, FK → `sessions` + `users` |

### 000001 — users

Core identity table. Key design decisions:

- UUID primary key (`gen_random_uuid()`) — no sequential IDs exposed externally.
- `email` has a unique constraint and an index — the primary lookup key for login.
- `role` and `status` are constrained by `CHECK` — invalid values are rejected at the DB level.
- Soft delete via `deleted_at` — users can be recovered; hard delete is not used.
- Partial indexes on `status` and `deleted_at` keep index size small.

### 000002 — sessions

Tracks active login sessions. Key design decisions:

- FK to `users(id) ON DELETE CASCADE` — deleting a user removes all their sessions.
- `token_hash` stores a hash of the session token, never the token itself.
- `revoked_at` allows explicit revocation without immediate deletion.
- Partial index on `expires_at WHERE revoked_at IS NULL` — only active sessions are indexed.

### 000003 — refresh_tokens

Supports token rotation for long-lived sessions. Key design decisions:

- FK to `sessions(id) ON DELETE CASCADE` — revoking a session removes all its refresh tokens.
- FK to `users(id) ON DELETE CASCADE` — deleting a user removes all their tokens.
- `used_at` marks a token as consumed; a used token cannot be reused (enforced in application).
- Partial index on `expires_at WHERE used_at IS NULL` — only unused tokens are indexed.

## Creating a New Migration

```bash
make migrate-create name=<description>
# Example:
make migrate-create name=add_org_id_to_users
```

This creates a numbered pair:
```
000004_add_org_id_to_users.up.sql
000004_add_org_id_to_users.down.sql
```

## Applying Migrations

```bash
make migrate-up          # apply all pending migrations to postulate_dev
make migrate-down        # roll back the most recent migration
make migrate-down-all    # roll back all migrations (prompts for confirmation)
make migrate-status      # show current schema version
make migrate-version     # print current schema version
```

All targets accept a `DB_URL` override:
```bash
make migrate-up DB_URL=postgres://user:pass@host:5432/dbname?sslmode=disable
```
