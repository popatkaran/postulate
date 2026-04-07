# US-03-04 — Google OAuth Flow

**Epic:** 03
**Depends on:** US-03-01, US-03-02, US-03-03 closed; US-03-06 in progress

---

## Summary

Implement the server-side Google OAuth 2.0 flow via Goth. An engineer is redirected to
Google, returns via callback, and has a Postulate user record looked up or created.

---

## Acceptance Criteria

**Configuration**

- `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` required at startup — absence causes
  startup to fail with a clear error.
- Callback URL derived from `POSTULATE_BASE_URL`, defaulting to `http://localhost:8080`
  in development.

**Endpoints**

- `GET /v1/auth/oauth/google` — redirects to Google with scope `openid email profile`,
  cryptographically random state parameter, and redirect URI.
- `GET /v1/auth/oauth/google/callback` — validates state, exchanges code, retrieves
  profile via Goth.

**State parameter**

- Cryptographically random value stored server-side (Redis or short-lived store) with a
  5-minute TTL.
- Validated on callback; mismatch or missing state returns `400 Bad Request` RFC 7807.
- Single-use — deleted after first validation.

**User lookup and creation**

- Look up `oauth_accounts` by `provider = 'google'` and `provider_uid = <sub claim>`.
- If found: load `users` record; update `access_token`, `refresh_token`, `token_expiry`
  in `oauth_accounts`.
- If not found: create `users` record (`role = platform_member`, subject to US-03-08
  bootstrap logic) and linked `oauth_accounts` record. Populate `email` and
  `display_name` from Google profile.
- If `users` record exists with same email but no Google `oauth_accounts` link: insert
  the link, preserve existing role.
- All lookup/creation logic lives in `ResolveOrCreateUser(ctx, ProviderUser)` in
  `internal/auth` — not in the handler.

**Session issuance**

- Call shared `IssueSessionToken` from US-03-06.
- Response: `{ "token", "refresh_token", "expires_at", "role" }`.
- For CLI clients: redirect to `http://127.0.0.1:<port>/callback` with token data as
  query parameters.

**Error handling**

- State mismatch — `400 Bad Request`.
- Google returns error (user denied) — `401 Unauthorized`.
- No email in profile — `422 Unprocessable Entity`.
- Database error — `500 Internal Server Error`; raw error logged, not returned.

---

## Implementation Notes

- Goth provider setup centralised in `internal/auth/providers.go`.
- Map `goth.User` to internal `auth.ProviderUser` immediately — do not pass `goth.User`
  beyond `internal/auth`.
- Token issuance must call the shared function from US-03-06, not a local implementation.
