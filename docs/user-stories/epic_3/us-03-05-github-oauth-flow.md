# US-03-05 — GitHub OAuth Flow

**Epic:** 03
**Depends on:** US-03-04 closed; US-03-06 in progress

---

## Summary

Implement the server-side GitHub OAuth 2.0 flow via Goth. Mirrors the Google flow in
structure and shares all user lookup and creation logic.

---

## Acceptance Criteria

**Configuration**

- `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` required at startup — absence causes
  startup to fail with a clear error.

**Endpoints**

- `GET /v1/auth/oauth/github` — redirects to GitHub with scope `read:user user:email`.
- `GET /v1/auth/oauth/github/callback` — validates state, exchanges code, retrieves
  profile.

**State parameter**

- Identical behaviour to US-03-04. Uses the same server-side state store.

**User lookup and creation**

- Calls `ResolveOrCreateUser` from US-03-04 with `provider = 'github'` and
  `provider_uid = <GitHub user ID as string>`.
- If the primary email is absent from the profile, make a secondary call to GitHub
  `/user/emails` to retrieve the primary verified email. If none found, return
  `422 Unprocessable Entity`.
- Same account-linking behaviour as US-03-04.

**Session issuance and error handling**

- Identical to US-03-04.

---

## Implementation Notes

- GitHub's OAuth token has no `id_token` or `sub` claim. GitHub numeric user ID coerced
  to string is the canonical `provider_uid`.
- `ResolveOrCreateUser` must exist before this story begins. If it does not yet exist when
  this story starts, extract it as a prerequisite within this PR — there must be no period
  where two independent user resolution implementations coexist.
