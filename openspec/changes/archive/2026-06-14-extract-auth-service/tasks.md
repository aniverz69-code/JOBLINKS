## 1. accounts package — policy and rules (TDD, fake repo)

- [x] 1.1 Create `internal/accounts` with the sentinels `ErrIdentityNotFound`, `ErrIdentityConflict`, `ErrNoVerifiedEmail`, the `Repository` interface (`UserIDByIdentity(ctx, provider, providerUserID string) (int64, error)`; `LinkOrCreateByEmail(ctx, provider, providerUserID, email string) (int64, error)`), and `type Service struct{ repo Repository }` + `func New(repo Repository) *Service`.
- [x] 1.2 Write failing unit tests against a fake `Repository` for: (a) identity found → returns that id, no link call; (b) not found + verified email + repo links/creates → returns the new id; (c) not found + `emailVerified=false` → `ErrNoVerifiedEmail` and `LinkOrCreateByEmail` NOT called; (d) not found + empty email → `ErrNoVerifiedEmail`, no link call; (e) race: `LinkOrCreateByEmail` returns `ErrIdentityConflict`, second `UserIDByIdentity` returns the winner id → service returns it; (f) race where the retry also returns `ErrIdentityNotFound` → service returns an error.
- [x] 1.3 Implement `Service.ResolveOAuthAccount(ctx, provider, providerUserID, email string, emailVerified bool) (int64, error)`: identity-first via `UserIDByIdentity`; on `ErrIdentityNotFound` apply the verified-email gate (lower/trim the email as part of matching); call `LinkOrCreateByEmail`; on `ErrIdentityConflict` retry `UserIDByIdentity` once. Make 1.2 pass.
- [x] 1.4 Confirm `go test ./internal/accounts/` is green and the package imports NO `pgx`/`pgtype`/`fiber`/`oauth`.

## 2. Persistence adapter (transaction owner)

- [x] 2.1 Add `QueriesRepository` in `internal/accounts` wrapping `*db.Queries` + `*pgxpool.Pool`. `UserIDByIdentity`: call `q.GetUserByIdentity`; `pgx.ErrNoRows` → `ErrIdentityNotFound`; success → the user id.
- [x] 2.2 Implement `LinkOrCreateByEmail` with the verbatim transaction from the current `handler.linkOrCreateUser`: `pool.Begin` → `defer Rollback` → `q.WithTx(tx)` → `GetUserByEmail` (reuse) or on `pgx.ErrNoRows` `CreateUser{Email, PasswordHash: pgtype.Text{}}` (passwordless) → `CreateUserIdentity` → `Commit`. If `CreateUserIdentity` (or commit) fails with a unique violation, return `ErrIdentityConflict` (port the existing `isUniqueViolation` SQLSTATE 23505 check into the adapter).
- [x] 2.3 Add `var _ Repository = (*QueriesRepository)(nil)`; confirm `go build ./...` and `go test ./internal/accounts/` green.

## 3. Thin the OAuth handler

- [x] 3.1 Add `accounts *accounts.Service` to `Handler` (handler.go) and construct it in `Register` from the shared `queries` + `pool` (`accounts.New(accounts.NewQueriesRepository(queries, pool))`).
- [x] 3.2 Rewrite `OAuthCallback` to call `h.accounts.ResolveOAuthAccount(c.Context(), p.Name(), identity.ProviderUserID, identity.Email, identity.EmailVerified)` in place of `h.resolveOAuthUser`; keep the surrounding transport (state/return cookies, `FetchIdentity`, `setSession`, success/`oauthFail` redirects) unchanged. Any service error routes through the existing `oauthFail`.
- [x] 3.3 Delete `resolveOAuthUser` and `linkOrCreateUser` from `oauth.go`. Remove now-unused imports (`pgx`, `pgtype`, `db` if no longer referenced in oauth.go). Leave `isUniqueViolation` in `handler` only if `Register`/another handler still uses it; otherwise remove it.
- [x] 3.4 Confirm `go build ./...`, `go vet ./...`, and `go test ./internal/handler/ ./internal/accounts/` are green.

## 4. Verify

- [x] 4.1 Run `go test -tags=integration ./internal/handler/` (real Postgres) — the OAuth callback / account-resolution end-to-end path must stay green (needs Docker; if unavailable in the loop, note it must pass in CI before merge).
- [x] 4.2 `gofmt -l` clean on the new/changed files; `go build ./... && go vet ./...` clean.
- [x] 4.3 Self-review: no `pgx`/`pgtype`/`fiber`/`oauth` import in `internal/accounts`; no account-resolution policy (verified-email gate, race retry, identity-first) left in `oauth.go`; the OAuth redirects/statuses/security rule are unchanged.
