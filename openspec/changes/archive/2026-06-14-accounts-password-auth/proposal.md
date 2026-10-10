## Why

Password sign-in (`Register`/`Login`/`Me` in `internal/handler/auth.go`) still mixes use-case rules with transport and persistence: the handler reaches `h.queries` directly and owns the rules — email normalization/validation, the minimum-password-length check, password hashing, the unique-violation → 409 mapping, the generic-401 "never reveal which factor failed" login rule, and the "valid token but user gone → 401" rule. This is the third slice of finding #2 (after `jobtracking` and the OAuth `accounts.Service`): fold these rules into the existing `accounts.Service` so the whole account surface is one testable use-case package and the auth handlers become thin. It also sets up the later `Register` parameter-list slimming.

## What Changes

- Extend `internal/accounts.Service` with `Register`, `Login`, and `UserByID` use cases, reusing the package built for OAuth resolution.
- Extend `internal/accounts.Repository` with `CreateUser`, `UserByEmail` (returns the stored password hash for verification — the hash never leaves the service), and `UserByID`, implemented by the existing `QueriesRepository` adapter.
- Introduce a narrow `PasswordHasher` interface (`Hash(plain) (string, error)`, `Check(hash, plain) error`) in `internal/accounts`. The bcrypt implementation is a thin adapter over `internal/auth` provided by the handler at construction, so `accounts` stays free of `auth`/`fiber`/`pgx` (which `internal/auth` would pull transitively).
- Move email normalization (`mail.ParseAddress` + lowercase) into the service as part of the matching rule.
- Rules become explicit sentinels: `ErrInvalidEmail`, `ErrPasswordTooShort`, `ErrEmailTaken`, `ErrInvalidCredentials` (the generic login failure — unknown email, wrong password, and passwordless account all collapse to it), `ErrUserNotFound`.
- The service returns a domain `User{ ID int64; Email string; CreatedAt *time.Time }`. `userResponse` switches to `*time.Time` for `created_at` (same wire string as the current `pgtype` marshaling; the auth integration tests are the guard).
- `internal/handler/auth.go` thins to transport: `BodyParser`, call the service, map sentinels → HTTP status (400/401/409), `setSession` (cookie), render `userResponse`. `Register`/`Login`/`Me` no longer touch `h.queries`. `isUniqueViolation` and `normalizeEmail` leave the handler if nothing else uses them.
- Unit tests exercise every rule against fake `Repository` + fake `PasswordHasher` (no DB, no bcrypt cost). No HTTP behavior change, no schema/SQL change.

## Capabilities

### New Capabilities
<!-- None. Internal refactor. -->

### Modified Capabilities
<!-- None. The `user-auth` spec's REQUIREMENTS are unchanged — register/login/me
     keep identical validation, status codes (201/200/400/401/409), the generic-401
     rule, the session cookie, and the user wire shape. Pure internal restructuring
     (no spec delta; archive with --skip-specs). -->

## Impact

- **Modified:** `internal/accounts` (add `Register`/`Login`/`UserByID` to `Service`, extend `Repository` + adapter, add `PasswordHasher`, move `normalizeEmail`); `internal/handler/auth.go` (thinned); `internal/handler/handler.go` (construct the service with the bcrypt hasher adapter — supersedes the OAuth-only construction).
- **Unchanged:** routes/middleware, `setSession`/cookie transport, `users` table + queries, the `auth.HashPassword`/`CheckPassword` primitives, `Logout`.
- **Out of scope (later):** reducing `Register`'s parameter count; `JobQueryService`; #3 phase 2 wire types.
