# US-03-07 — Authentication Middleware

**Epic:** 03
**Depends on:** US-03-06 closed

---

## Summary

Implement the JWT authentication middleware that protects all endpoints requiring
authentication, and the role enforcement middleware that gates by role.

---

## Acceptance Criteria

**`AuthRequired` middleware**

- Standard Chi middleware: `func AuthRequired(next http.Handler) http.Handler`.
- Extracts token from `Authorization: Bearer <token>` header.
- Validates JWT signature, `exp` claim, and `iat` claim (60-second clock skew tolerance).
- On success: injects `auth.Identity{ UserID uuid.UUID, Role string }` into request
  context using an unexported typed context key (not a `string` key).
- Helper `auth.IdentityFromContext(ctx) (auth.Identity, bool)` provided for handlers.
- Failure responses — all `401 Unauthorized`, RFC 7807. Message must not distinguish
  missing header, malformed token, or expired token.

**`RequireRole` middleware**

- `func RequireRole(role string) func(http.Handler) http.Handler`.
- Chains after `AuthRequired`. Returns `403 Forbidden` RFC 7807 if role does not match.
- Usage: `router.With(AuthRequired, RequireRole("platform_admin")).Get(...)`.

**Unauthenticated endpoints** — `AuthRequired` must not be applied to:

- `GET /health`, `GET /v1/version`
- `GET /v1/auth/oauth/google`, `GET /v1/auth/oauth/google/callback`
- `GET /v1/auth/oauth/github`, `GET /v1/auth/oauth/github/callback`
- `POST /v1/auth/token/refresh`

---

## Implementation Notes

- No database call in the middleware hot path — identity resolved entirely from JWT claims.
- Context key must be an unexported custom type, e.g. `type contextKey int`, defined
  within `internal/auth`.
- Known limitation: a valid JWT for a deactivated user still passes until natural expiry.
  Document as an inline comment in `AuthRequired` and in `docs/auth.md` (US-03-11).
