# Tasks — add-oauth-sign-in

## 1. Database

- [x] 1.1 Add `migrations/0010_user_identities.sql`: `user_identities (provider, provider_user_id, user_id FK ON DELETE CASCADE, created_at, PRIMARY KEY (provider, provider_user_id))` + index on `user_id`
- [x] 1.2 Add `internal/db/queries/user_identities.sql`: `GetUserByIdentity` (join to users), `CreateUserIdentity`, and `CreateUserPasswordless` (or reuse `CreateUser` with NULL hash) — whatever the resolution flow needs
- [x] 1.3 Run `make sqlc`, commit generated code (dev volume recreate: `docker compose down -v && make up` — pending on the next local run)

## 2. Config & dependency

- [x] 2.1 `go get golang.org/x/oauth2`
- [x] 2.2 Add per-provider OAuth credentials to `config.Settings` (`OAUTH_GOOGLE_CLIENT_ID`/`_SECRET`, `GITHUB`, `LINKEDIN`); a provider is enabled only when both values are set

## 3. OAuth providers (`internal/auth/oauth`)

- [x] 3.1 Define `Identity` + `Provider` interface and the registry built from config (only configured providers registered); state helpers (random state generation, state cookie set/clear/verify) with unit tests
- [x] 3.2 Google provider: `oauth2.Config` (scopes `openid email`), identity via OIDC userinfo (`sub`, `email`, `email_verified`); unit test against a stub userinfo server
- [x] 3.3 GitHub provider: scopes `read:user user:email`, id via `GET /user`, primary **verified** email via `GET /user/emails`; unit test against a stub API server
- [x] 3.4 LinkedIn provider: OIDC userinfo flow (scopes `openid email`); unit test against a stub userinfo server

## 4. HTTP handlers (`internal/handler/oauth.go`)

- [x] 4.1 `GET /api/v1/auth/oauth/providers`: `{"data": [enabled provider names]}`
- [x] 4.2 `GET /api/v1/auth/oauth/:provider/start`: 404 for unknown/disabled provider; set the state cookie; 302 to the provider consent URL
- [x] 4.3 Account resolution (identity → verified-email link → new passwordless user) in one transaction, with unique-violation race handling
- [x] 4.4 `GET /api/v1/auth/oauth/:provider/callback`: verify + clear state cookie, exchange code, fetch identity, resolve account, set the session cookie, 302 to `FRONTEND_ORIGIN/`; all failures 302 to `FRONTEND_ORIGIN/?auth_error=oauth` (logged)
- [x] 4.5 Wire routes + registry in `handler.Register`; thread config from `cmd/server`

## 5. Web (SPA)

- [x] 5.1 `api.ts`: fetch enabled providers; `AuthDialog.svelte`: "Continue with <Provider>" buttons (anchors to the start URL) above the email/password form, shown only for enabled providers
- [x] 5.2 Surface `?auth_error=oauth` after a failed callback redirect (inline message), and clean the param from the URL

## 6. Verification & docs

- [x] 6.1 `go build ./... && go vet ./... && go test ./...` pass
- [x] 6.2 Manual e2e with at least one real provider (Google or GitHub): new-user sign-in, returning sign-in, and email-linking to an existing password account
- [x] 6.3 Update `AGENT.md` (auth convention + config) with the OAuth surface
