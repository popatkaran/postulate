# US-03-11 — Epic 03 Coverage Gate and Hardening

**Epic:** 03
**Depends on:** US-03-01 through US-03-10 closed

---

## Summary

Close Epic 03 by verifying all coverage requirements are met and completing the hardening
checklist.

---

## Acceptance Criteria

**Coverage gate**

- `internal/auth` — 95% line / branch / function.
- `internal/middleware` — 95% line / branch / function.
- `internal/session` — 95% line / branch / function.
- All other packages in this Epic — 90% across all three dimensions.
- CI pipeline coverage gate passes on the Epic 03 integration branch.

**Hardening checklist**

- Startup validation tests confirm each of the following causes startup to fail with a
  clear error: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GITHUB_CLIENT_ID`,
  `GITHUB_CLIENT_SECRET`, `POSTULATE_JWT_SECRET` absent, `POSTULATE_JWT_SECRET` shorter
  than 32 bytes.
- Token issuance tests assert on `exp`, `iat`, `sub`, and `role` claim values explicitly.
- Refresh token rotation tested: a used refresh token is rejected on second use with
  `401 Unauthorized`.
- Development bootstrap fallback covered by an integration test using a
  transaction-isolated empty `users` table.
- Production bootstrap guard tested: `POSTULATE_ENV=production` with no
  `POSTULATE_BOOTSTRAP_ADMIN_EMAIL` does not fire the fallback.
- Concurrent first-login test: two simultaneous logins against an empty database result
  in exactly one `platform_admin`.
- CLI callback listener timeout tested using a shortened context timeout — not by
  waiting 120 seconds in CI.
- OAuth state parameter single-use tested: resubmitting a used state returns
  `400 Bad Request`.

**Documentation**

- `docs/auth.md` created covering: OAuth provider configuration, JWT lifecycle and
  claims, refresh token rotation, bootstrap admin configuration, development fallback
  behaviour, and the known JWT-validity-after-logout limitation.
- Known limitation documented as inline comment in both the logout handler and
  `RevokeAllSessions`.
