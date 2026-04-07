# US-03-01 — Schema Migration — `oauth_accounts` Table

**Epic:** 03
**Depends on:** Epic 02 schema present

---

## Summary

Introduce the `oauth_accounts` table, which links third-party OAuth provider identities
to internal Postulate user records.

---

## Acceptance Criteria

**Table definition**

| Column | Type | Constraints |
|---|---|---|
| `id` | `uuid` | Primary key, default `gen_random_uuid()` |
| `user_id` | `uuid` | Not null, FK → `users.id` `ON DELETE CASCADE` |
| `provider` | `text` | Not null |
| `provider_uid` | `text` | Not null |
| `email` | `text` | Not null |
| `access_token` | `text` | Nullable |
| `refresh_token` | `text` | Nullable |
| `token_expiry` | `timestamptz` | Nullable |
| `created_at` | `timestamptz` | Not null, default `now()` |
| `updated_at` | `timestamptz` | Not null, default `now()` |

- Unique constraint on `(provider, provider_uid)`.
- Index on `user_id`.

**Migration tooling**

- Sequentially numbered SQL file under `internal/db/migrations/`, continuing from the
  highest Epic 02 sequence number.
- Corresponding `down` migration cleanly drops the table, constraints, and index.
- `make migrate` and `make migrate-down` apply and reverse cleanly against PostgreSQL 16.

---

## Implementation Notes

- Apply the same `updated_at` trigger pattern used on `users` in Epic 02, if one exists.
  If not, document the manual update requirement in the migration README.
- `access_token` and `refresh_token` store the OAuth provider's tokens, not Postulate
  JWTs. Nullable because not all providers return both values.
- Do not modify `users`, `sessions`, or `refresh_tokens` in this story.
