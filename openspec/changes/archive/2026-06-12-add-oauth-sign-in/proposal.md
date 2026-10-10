# Add OAuth sign-in (Google, GitHub, LinkedIn)

## Why

Today the only way to sign in is email + password. The auth foundation was
deliberately shaped for passwordless providers (nullable `password_hash`,
email as the canonical account key, JWT carrying only `sub`), but no provider
flow exists. OAuth sign-in removes sign-up friction and was the announced next
step of the user-auth change.

## What Changes

- Add a `user_identities` table linking external provider identities
  (`provider`, `provider_user_id`) to `users` — the seam designed in
  `add-user-auth`, now built.
- Add a server-side OAuth2 authorization-code flow for three providers —
  Google, GitHub, LinkedIn — behind a provider registry (mirroring
  `internal/sources`):
  - `GET /api/v1/auth/oauth/providers` — list of enabled providers (the SPA
    renders buttons from it).
  - `GET /api/v1/auth/oauth/:provider/start` — sets a CSRF `state` cookie and
    redirects to the provider's consent page.
  - `GET /api/v1/auth/oauth/:provider/callback` — verifies `state`, exchanges
    the code, fetches the user's identity (id + verified email), resolves or
    creates the account, sets the existing JWT session cookie, and redirects
    back to the SPA.
- Account resolution: existing identity → that user; otherwise a **verified**
  provider email matching an existing account links a new identity to it;
  otherwise a new passwordless user is created. Unverified emails never link.
- Config: per-provider `OAUTH_<PROVIDER>_CLIENT_ID` / `OAUTH_<PROVIDER>_CLIENT_SECRET`
  env vars; a provider is enabled only when both are set. Redirect URLs derive
  from `FRONTEND_ORIGIN` (same-origin deployment).
- SPA: the auth dialog gains "Continue with Google / GitHub / LinkedIn"
  buttons (only for enabled providers) above the email/password form.
- Existing password auth, session transport (httpOnly JWT cookie), and all
  public endpoints are unchanged.

## Capabilities

### Modified Capabilities

- `user-auth`: adds OAuth provider sign-in (external identities, the
  start/callback flow, provider listing, SPA provider buttons) on top of the
  existing password + cookie-session requirements.

## Impact

- New migration `0010_user_identities.sql` (additive; needs a dev volume
  recreate, same as all migrations).
- New queries `internal/db/queries/user_identities.sql` + regenerated sqlc.
- New Go dependency: `golang.org/x/oauth2`.
- New package `internal/auth/oauth` (provider abstraction + google/github/
  linkedin implementations + registry); new handler file
  `internal/handler/oauth.go`; config additions.
- SPA: `AuthDialog.svelte` (provider buttons), `api.ts` (providers fetch).
