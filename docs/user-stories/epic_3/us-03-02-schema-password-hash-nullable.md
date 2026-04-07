# US-03-02 — Schema Migration — `users.password_hash` Nullable

**Epic:** 03
**Depends on:** US-03-01 closed

---

## Summary

Alter `users.password_hash` to allow `NULL`, reflecting the confirmed OAuth-only
authentication model. The column is retained — not dropped.

---

## Acceptance Criteria

- `password_hash` is altered from `NOT NULL` to nullable.
- Existing rows are unaffected — migration is safe on a non-empty database.
- No default value is set; `NULL` is the correct state for all OAuth-only users.
- Sequentially numbered SQL file after US-03-01.
- Corresponding `down` migration restores `NOT NULL`. The down migration must document
  that it is potentially destructive if any row has a `NULL` value at rollback time.
- `make migrate` and `make migrate-down` apply and reverse cleanly.

---

## Implementation Notes

- Audit the codebase for any assertions that `password_hash` is non-null. Update them in
  this story — do not leave code contradicting the schema.
- If the Epic 02 `UserRepository` or domain type enforces non-null at the Go level,
  relax it to a pointer or `sql.NullString` equivalent here.
