# Design — add-oauth-sign-in

## Context

`internal/auth` already owns the session primitives (HS256 `Issuer`, httpOnly
`SameSite=Lax` cookie transport, `RequireAuth`). The JWT carries only the user
id, so OAuth sign-in needs no token/middleware change — it only needs a way to
turn "provider says this is person X" into a `users.id`, then reuse
`SetTokenCookie`. The data model anticipated this: `password_hash` is nullable
and `UNIQUE (lower(email))` makes email the canonical account key. The SPA and
API are same-origin (Vite proxy in dev), which makes the redirect flow and the
state cookie straightforward.

## Goals / Non-Goals

**Goals:**

- Sign in / sign up via Google, GitHub, LinkedIn with the standard server-side
  authorization-code flow (full-page redirect, no popup).
- One account per email: provider sign-in with a verified email matching an
  existing account links to it; otherwise it creates a passwordless account.
- Providers are individually enableable via env config; an unconfigured
  provider is invisible (not listed, its routes 404).
- Reuse the existing session exactly (same cookie, same JWT, same `/me`).

**Non-Goals:**

- Magic-link sign-in (separate future change; the model already permits it).
- Account-management UI (listing/unlinking identities) — additive later.
- PKCE — confidential server-side client with a secret + `state` cookie covers
  the threat model; add PKCE if a public client ever appears.
- Refresh of provider tokens / calling provider APIs after sign-in. We discard
  provider tokens once the identity is fetched; nothing stores them.

## Decisions

### Standard authorization-code flow, server-side, full-page redirect

`start` builds the consent URL and 302s the browser to the provider; the
provider 302s back to `callback` with `code` + `state`. The callback is a
top-level GET navigation, so the `SameSite=Lax` state cookie is sent — Lax
works for both the state cookie and the session cookie. A popup/PKCE SPA flow
would add complexity for no benefit when the backend can hold the client
secret.

Routes live under `/api/v1/auth/oauth/:provider/...` so the literal
`register`/`login`/`me`/`logout` routes are untouched and the provider name is
a clean path param validated against the registry (unknown/disabled → 404).

### CSRF: random `state` in a short-lived httpOnly cookie

`start` generates 32 random bytes (base64url), sets them as an
`HttpOnly; SameSite=Lax; Max-Age=600` cookie, and passes the same value as
`state`. `callback` compares cookie vs query and clears the cookie. Mismatch
or absence → reject. This is the standard stateless defense; no server-side
session store is needed.

### Provider abstraction in `internal/auth/oauth`, registry like `internal/sources`

```go
type Identity struct {
    ProviderUserID string
    Email          string // empty if none
    EmailVerified  bool
}

type Provider interface {
    Name() string
    AuthCodeURL(state string) string
    FetchIdentity(ctx context.Context, code string) (Identity, error)
}
```

Each provider wraps an `oauth2.Config` (from `golang.org/x/oauth2`, which
ships Google/GitHub/LinkedIn endpoints) plus its identity fetch:

- **Google** — scopes `openid email`; identity from the OIDC `userinfo`
  endpoint (`sub`, `email`, `email_verified`).
- **GitHub** — scope `read:user user:email`; id from `GET /user`, email from
  `GET /user/emails` picking the primary **verified** email (GitHub's `/user`
  `email` field is often null).
- **LinkedIn** — "Sign In with LinkedIn using OpenID Connect": scopes
  `openid email`; identity from its `userinfo` endpoint (same OIDC shape as
  Google).

The handler maps `:provider` through a `map[string]Provider` built at startup
from config — only providers with both client id and secret present are
registered. Adding a provider later = one type + one registry line, mirroring
how `internal/sources` grows.

### Account resolution: identity → verified email → new user, in one transaction

On callback, with the fetched `Identity`:

1. `GetUserByIdentity(provider, provider_user_id)` → hit: session for that
   user. (Returning user — the common case, one query.)
2. Miss: require a **verified** email. No verified email → fail the sign-in
   (redirect with an error). Auto-linking on an unverified email would let
   anyone claim `victim@example.com` at a provider and take over the matching
   account.
3. `GetUserByEmail(lower(email))` → hit: insert the identity row linking to
   that user (account linking). Miss: create a passwordless user
   (`password_hash` NULL) and the identity row.

Steps 2–3 run in a single pgx transaction (`Queries.WithTx`) so a concurrent
duplicate callback cannot create a half-linked state; the
`PRIMARY KEY (provider, provider_user_id)` and `UNIQUE (lower(email))`
constraints are the backstop — a unique-violation race retries the lookup.

### Data model

`migrations/0010_user_identities.sql`:

```sql
CREATE TABLE user_identities (
    provider         text        NOT NULL,
    provider_user_id text        NOT NULL,
    user_id          bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, provider_user_id)
);
CREATE INDEX user_identities_user_id_idx ON user_identities (user_id);
```

Reference-only link table; no provider tokens, no profile copy — `users` stays
canonical, exactly as the `add-user-auth` design sketched.

### Callback UX: redirect, never JSON

The callback is a browser navigation, so errors must not land the user on a
JSON page. Success → `302 FRONTEND_ORIGIN/`. Any failure (state mismatch,
exchange error, no verified email) → `302 FRONTEND_ORIGIN/?auth_error=oauth`
with the detail logged server-side. The SPA already resolves the session via
`GET /me` on boot, so after the redirect the user is simply signed in — no new
SPA session code, just provider buttons (anchors to the `start` URL) and an
optional error toast for `auth_error`.

### Config

`OAUTH_GOOGLE_CLIENT_ID`/`OAUTH_GOOGLE_CLIENT_SECRET`, same pattern for
`GITHUB` and `LINKEDIN`, as a `map[string]OAuthCredentials` style struct in
`config.Settings`. No new "base URL" variable: redirect URLs derive from the
existing `FRONTEND_ORIGIN` (same-origin — in dev the Vite proxy forwards
`/api`, so `http://localhost:5173/api/v1/auth/oauth/<p>/callback` is what gets
registered at each provider). Missing credentials disable the provider; the
server still boots (unlike `JWT_SECRET`, OAuth is optional).

## Risks / Trade-offs

- **LinkedIn app review** — "Sign In with LinkedIn using OpenID Connect"
  requires enabling that product on the LinkedIn developer app. Operational,
  not code; the provider stays disabled until credentials exist.
- **Email change at the provider** — we link by identity first, so a later
  provider-email change does not re-key the account. Accepted: email is only
  used at first link.
- **Account linking by email** — restricted to verified emails (see above);
  residual risk is a compromised provider account, which is out of scope.
- **No migration runner** (existing gotcha) — `0010` applies on fresh volumes
  only; dev needs `docker compose down -v && make up`.

## Open Questions

- None blocking. Identity unlinking and a magic-link provider are noted seams.
