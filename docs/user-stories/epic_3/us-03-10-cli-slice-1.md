# US-03-10 — CLI Slice 1 — `postulate login` and `postulate logout`

**Epic:** 03
**Depends on:** US-03-04, US-03-05, US-03-06, US-03-07, US-03-09 closed

---

## Summary

Implement the first two CLI commands. All remaining CLI commands are deferred to Epic 11.

---

## Acceptance Criteria

**`postulate login` — OAuth flow**

- Starts a temporary HTTP listener on `127.0.0.1`, random port in range `18000–18099`.
- Constructs the OAuth initiation URL with `redirect_uri` pointing to the local listener.
- Opens URL in the default system browser (`xdg-open` / `open` / `start`). If browser
  cannot be opened, prints URL to stdout with instruction to open manually.
- Prints a waiting message while listener is active.
- The Postulate API callback redirects to `http://127.0.0.1:<port>/callback` with token
  data as query parameters. CLI extracts values from query parameters.
- Writes to `~/.postulate/auth.json` (permissions `600`):

```json
{
  "token": "<jwt>",
  "refresh_token": "<raw>",
  "expires_at": "<iso8601>",
  "role": "<role>",
  "api_url": "<api url>"
}
```

- Prints: `Logged in as <email> (<role>).` on success.
- Listener shuts down after callback or after 120-second timeout. Timeout exits with
  non-zero code and clear error message. Uses `context.WithTimeout` — does not sleep.
- Creates `~/.postulate/` with permissions `700` if it does not exist.

**`postulate login` — provider selection**

- Without flags: prompts user to select Google or GitHub.
- `--provider google|github` bypasses the prompt.

**Silent token refresh (pattern for all future commands)**

- Before any API call, check `expires_at` in `auth.json`. If within 30 minutes of expiry,
  call `POST /v1/auth/token/refresh`, receive new values, overwrite `auth.json`.
- Refresh failure: print `Session expired. Run postulate login to continue.` to stderr,
  exit non-zero. No stack trace or raw error output.

**`postulate logout`**

- Calls `DELETE /v1/auth/token` with Bearer token from `auth.json`.
- On `204`: deletes `auth.json`, prints `Logged out.`
- If `auth.json` missing or token already expired: deletes file if present, prints
  `Logged out.` — does not error on already-clean state.
- On API error: reports to stderr, does not delete `auth.json`.
- `--force` flag deletes `auth.json` regardless of API response.

**`--api-url` global flag**

- Overrides the API base URL stored in `auth.json`.
- Default: `https://api.postulate.internal` or `POSTULATE_API_URL` env var if set.
- Stored in `auth.json` on successful login.

**Binary scope**

- This binary contains only `postulate login`, `postulate logout`, `--api-url`, `--help`,
  and `version`. Epic 11 adds all remaining commands.

---

## Implementation Notes

- All `~/.postulate/` and `auth.json` operations centralised in a single package (e.g.,
  `internal/credentials`). No file paths or permission constants in handler code.
- Local callback server communicates with the main goroutine via a buffered channel, not
  shared memory.
- The query-parameter redirect mechanism (agreed in this story) must be documented in the
  PR description and in `docs/auth.md`.
- Cross-platform browser launch tested via a mock exec function in unit tests — no
  dependency on system browser in CI.
- Windows browser launch is best-effort in Phase 1; manual URL fallback is acceptable.
