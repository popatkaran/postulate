# US-03-08 — Platform Admin Bootstrap and Role Enforcement

**Epic:** 03
**Depends on:** US-03-03 closed; US-03-04 and US-03-05 in progress

---

## Summary

Implement the platform admin bootstrap mechanism and enforce the role model during user
creation.

---

## Acceptance Criteria

**`POSTULATE_BOOTSTRAP_ADMIN_EMAIL`**

- If set, the user whose email matches is assigned `platform_admin` on first login,
  regardless of whether other users exist.
- If that user already exists as `platform_member`, their role is upgraded on login.
- Optional — startup must not fail if absent.

**Development fallback**

- If `POSTULATE_BOOTSTRAP_ADMIN_EMAIL` is not set and `users` contains zero rows at the
  time of the first successful login, that user receives `platform_admin`.
- All subsequent users receive `platform_member`.
- Fallback is disabled when `POSTULATE_ENV=production` and the variable is not set.
  A `WARN` log entry is emitted at startup in this configuration.

**Role in JWT**

- The `role` claim reflects the user's role at token issuance time. Role changes require
  a new login or token refresh to take effect.

**Service layer enforcement**

- Bootstrap logic applied inside `ResolveOrCreateUser`, not in handler code.
- Only `platform_admin` and `platform_member` are valid. Any attempt to persist an
  unlisted role returns a service-layer error, independent of the DB constraint.

---

## Implementation Notes

- Implement the zero-row check as a single transactional operation to prevent two
  concurrent first-logins both receiving `platform_admin`. Use `SELECT COUNT(*) FROM users`
  within the same transaction as the insert, with an appropriate isolation level or
  advisory lock. Document the chosen approach inline.
- Bootstrap check runs only during user creation — not on every subsequent login.
