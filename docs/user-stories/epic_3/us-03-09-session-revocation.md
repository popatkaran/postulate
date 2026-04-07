# US-03-09 — Session Revocation (Logout)

**Epic:** 03
**Depends on:** US-03-06 closed, US-03-07 closed

---

## Summary

Implement the server-side logout endpoint, which revokes the user's active session and
signals the CLI to delete local credentials.

---

## Acceptance Criteria

- `DELETE /v1/auth/token` protected by `AuthRequired`.
- Calls `RevokeAllSessions(ctx, userID)` from US-03-06 — deletes all refresh tokens for
  the authenticated user.
- Returns `204 No Content` on success.
- Returns `204 No Content` if the user has no active refresh tokens — idempotent.
- Database error returns `500 Internal Server Error`, RFC 7807; raw error logged with
  trace ID, not returned in body.

---

## Implementation Notes

- Outstanding JWTs remain valid until natural 8-hour expiry after revocation. Document
  this explicitly as an inline comment in the handler — this is the accepted trade-off
  for stateless JWT design.
- The CLI (US-03-10) deletes `auth.json` immediately on receiving `204`.
