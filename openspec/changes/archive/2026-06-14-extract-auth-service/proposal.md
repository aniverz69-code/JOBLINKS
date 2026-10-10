## Why

The OAuth account-resolution logic — the riskiest, most business-heavy code the HTTP layer holds — lives directly in `internal/handler/oauth.go`. `resolveOAuthUser` owns the security-critical policy (identity-first lookup, the "only a **verified** email may link or create an account" rule, and the concurrent-callback race retry), and `linkOrCreateUser` opens a raw `pgx` transaction (`h.pool.Begin`, `queries.WithTx`) to link-or-create the account atomically. None of this can be unit-tested without a real Postgres and a full OAuth callback, so the account-takeover-prevention rule and the race handling are only exercised by integration tests. This is the highest-value slice of finding #2 (and the natural successor to the job-tracking extraction): move the account-resolution use case behind a service + repository so the policy is unit-testable and the transaction is an implementation detail of the adapter.

## What Changes

- Introduce `internal/accounts` — a `Service` owning `ResolveOAuthAccount`: the identity-first → verified-email-gate → link-or-create → race-retry policy, depending on a narrow `Repository` interface.
- The transaction moves into the repository adapter: a coarse `LinkOrCreateByEmail` operation runs the `GetUserByEmail` → (`CreateUser` passwordless | reuse) → `CreateUserIdentity` sequence in one `pgx` transaction. The service never sees `pgx`/`pgtype`/`*db.Queries`.
- The service takes primitives (`provider, providerUserID, email string, emailVerified bool`), not `oauth.Identity`, so it does not couple to the `oauth` package; the handler unpacks the identity.
- Security rules become explicit, testable sentinels: `ErrNoVerifiedEmail` (unverified/empty email → never link or create), and an internal `ErrIdentityConflict` (unique-violation on the identity insert → the service retries the identity lookup once, preserving the current concurrent-callback behavior).
- `internal/handler/oauth.go` keeps the transport: provider lookup, CSRF state + return-path cookies, code exchange, calling the service, `setSession`, and the redirect/`auth_error` behavior. `resolveOAuthUser`/`linkOrCreateUser` are deleted from the handler.
- Unit tests for the service exercise every branch (identity found; verified-email link to existing; verified-email create passwordless; unverified/empty email rejected; race → retry resolves) against a fake `Repository`, with no DB.
- No HTTP behavior change, no schema change, no SQL change; the OAuth callback's redirects, statuses, and account semantics are identical.

## Capabilities

### New Capabilities
<!-- None. Internal refactor; no new user-facing capability. -->

### Modified Capabilities
<!-- None. The `user-auth` spec's REQUIREMENTS are unchanged — OAuth sign-in
     keeps identical identity-first resolution, the verified-email-only link/create
     rule, the race behavior, the session cookie, and the redirect/auth_error flow.
     Pure internal restructuring (no spec delta; archive with --skip-specs). -->

## Impact

- **New package:** `internal/accounts` (`Service`, `Repository` interface, sentinels, a tx-owning `*db.Queries`+`*pgxpool.Pool` adapter, unit tests).
- **Modified:** `internal/handler/oauth.go` (thinned — `resolveOAuthUser`/`linkOrCreateUser` removed, `OAuthCallback` calls the service), `internal/handler/handler.go` (constructs/holds the service). `isUniqueViolation` moves to the adapter (or stays shared if still used by `Register`).
- **Unchanged:** routes/middleware, the `oauth` provider registry + CSRF/return cookies, `setSession`/`SetTokenCookie`, the `users`/`user_identities` tables and their queries, and the `user-auth` wire/redirect behavior.
- **Out of scope (later slices):** extracting password `Register`/`Login`/`Me` into the same service (they are thin and low-risk); reducing `Register`'s parameter count; `JobQueryService`.
