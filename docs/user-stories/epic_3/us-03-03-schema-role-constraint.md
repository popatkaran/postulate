# US-03-03 — Schema Migration — `users.role` Constraint Update

**Epic:** 03
**Depends on:** US-03-02 closed

---

## Summary

Update the `role` column on `users` to enforce only `platform_admin` and
`platform_member`, and set `platform_member` as the column default.

---

## Acceptance Criteria

- Any existing check constraint or enum on `role` referencing prior values is dropped.
- New check constraint: `role IN ('platform_admin', 'platform_member')`.
- Column default set to `'platform_member'`.
- Existing rows with values outside the new constraint are either updated to
  `platform_member` or the migration fails with a clear error. Silent data corruption
  is not acceptable. The chosen approach is documented as a comment in the migration file.
- Sequentially numbered SQL file after US-03-02.
- Corresponding `down` migration removes the new constraint and default and restores the
  prior state accurately.
- `make migrate` and `make migrate-down` apply and reverse cleanly.

---

## Implementation Notes

- Audit the codebase for any role string literals and update them to reference only
  `platform_admin` and `platform_member` in the same PR.
- If a domain type for user role exists in `internal/domain`, update it to reflect the
  two valid values. Schema change and code change land together.
