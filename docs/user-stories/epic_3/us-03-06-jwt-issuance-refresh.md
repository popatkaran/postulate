# US-03-06 — JWT Session Token Issuance and Refresh

**Epic:** 03
**Depends on:** US-03-01, US-03-02, US-03-03 closed
**Can be developed in parallel with:** US-03-04, US-03-05 (agree function signature first)

---

## Summary

Implement JWT session token issuance, refresh token lifecycle, and the token refresh
endpoint. This is the token infrastructure that all OAuth handlers depend on.

---

## Acceptance Criteria

**Session token issuance**

- Function signature:
  `IssueSessionToken(ctx, userID uuid.UUID, role string) (token, refreshToken string, expiresAt time.Time, err error)`
  implemented in `internal/auth`.
- JWT signed with HS256 using `POSTULATE_JWT_SECRET`. Required at startup; minimum 32
  bytes — shorter value or absence causes startup to fail with a clear error.
- JWT payload: `sub` (user ID), `role`, `iat`, `exp` (8h from issuance). No PII beyond
  user ID and role.

**Refresh token issuance and storage**

- Cryptographically random 256-bit value, hex-encoded.
- Stored in `refresh_tokens` as `token_hash` (SHA-256 of raw token — raw token never
  stored), with `user_id`, `expires_at` (30 days), `created_at`.
- Raw token returned to caller once; not recoverable after issuance.
- If Epic 02 stored the raw token rather than a hash, add a schema migration in this
  story (numbered after US-03-03) to replace the column with `token_hash`.

**Token refresh endpoint**

- `POST /v1/auth/token/refresh` — unauthenticated.
- Accepts `{ "refresh_token": "<raw>" }`.
- Hashes submitted value, queries `refresh_tokens`, verifies hash match, not expired,
  not previously consumed.
- On success: issue new JWT and refresh token, delete old refresh token record (rotation).
- Response: `{ "token", "refresh_token", "expires_at", "role" }`.
- On failure: `401 Unauthorized`, RFC 7807. Message must not distinguish "not found" from
  "expired".

**Session revocation helper**

- `RevokeAllSessions(ctx, userID uuid.UUID) error` in `internal/auth`.
- Deletes all `refresh_tokens` rows for the user.
- Document inline: outstanding JWTs remain valid until natural expiry — this is the
  accepted trade-off for a stateless JWT design.

---

## Implementation Notes

- Do not use `sessions` table rows to gate JWT validity on the hot path. JWT validity is
  determined solely by signature and `exp` claim.
- Token issuance tests must assert on `exp`, `iat`, `sub`, and `role` claim values
  explicitly — not just on token presence.
