## 1. Extend accounts.Service with password auth (TDD, fakes)

- [x] 1.1 In `internal/accounts`, add the domain `User{ ID int64; Email string; CreatedAt *time.Time }`, the `PasswordHasher` interface (`Hash(plain string) (string, error)`, `Check(hash, plain string) error`), and the sentinels `ErrInvalidEmail`, `ErrPasswordTooShort`, `ErrEmailTaken`, `ErrInvalidCredentials`, `ErrUserNotFound`. Add `minPasswordLen = 8`. Give `Service` a `hasher PasswordHasher` field and update `New` to `New(repo Repository, hasher PasswordHasher) *Service` (update the OAuth construction site in handler.go and existing accounts tests to pass a hasher — a nil/fake is fine where unused).
- [x] 1.2 Extend the `Repository` interface with `CreateUser(ctx, email, passwordHash string) (User, error)` (returns `ErrEmailTaken` on a unique violation), `UserByEmail(ctx, email string) (user User, passwordHash string, hasPassword bool, err error)` (returns `ErrUserNotFound` when absent; `hasPassword=false` for a passwordless/OAuth account), and `UserByID(ctx, id int64) (User, error)` (`ErrUserNotFound` when absent).
- [x] 1.3 Add `normalizeEmail(raw string) (string, error)` to the package (move the `mail.ParseAddress`+lowercase logic from the handler).
- [x] 1.4 Write failing unit tests (fake `Repository` + fake `PasswordHasher`) for: Register — invalid email → `ErrInvalidEmail` (no repo call); short password → `ErrPasswordTooShort` (no repo call); happy path hashes then `CreateUser` with the hash, returns the user; unique → `ErrEmailTaken`. Login — unknown email → `ErrInvalidCredentials`; passwordless account (`hasPassword=false`) → `ErrInvalidCredentials` (hasher.Check NOT called); bad password (hasher.Check returns error) → `ErrInvalidCredentials`; good password → returns the user. UserByID — found → user; absent → `ErrUserNotFound`.
- [x] 1.5 Implement `Service.Register`, `Service.Login`, `Service.UserByID` to pass 1.4 (Register: normalize email → length check → `hasher.Hash` → `repo.CreateUser`; Login: normalize → `repo.UserByEmail` → reject if `!hasPassword` or `hasher.Check` fails → user; UserByID: passthrough). Confirm `go test ./internal/accounts/` green and the package still imports no `auth`/`fiber`/`pgx`/`pgtype`/`oauth`.

## 2. Extend the adapter

- [x] 2.1 In `internal/accounts/repository.go`, implement the three new `Repository` methods on `QueriesRepository`: `CreateUser` (call `q.CreateUser` with `pgtype.Text{String: passwordHash, Valid: true}`; map unique violation → `ErrEmailTaken`; map `db.CreateUserRow` → `User`); `UserByEmail` (call `q.GetUserByEmail`; `pgx.ErrNoRows` → `ErrUserNotFound`; return the `PasswordHash.String`/`.Valid` as `passwordHash`/`hasPassword` and map the row → `User`); `UserByID` (call `q.GetUserByID`; `pgx.ErrNoRows` → `ErrUserNotFound`). Map `pgtype.Timestamptz` → `*time.Time` (reuse/extend a helper) for `User.CreatedAt`.
- [x] 2.2 Confirm `var _ Repository = (*QueriesRepository)(nil)` still compiles; `go build ./...` and `go test ./internal/accounts/` green.

## 3. bcrypt hasher adapter + thin the handler

- [x] 3.1 Add a bcrypt `PasswordHasher` adapter (in `internal/handler`, e.g. `authHasher`) wrapping `auth.HashPassword`/`auth.CheckPassword`; pass it into `accounts.New(accounts.NewQueriesRepository(queries, pool), authHasher{})` in `Register` (handler.go).
- [x] 3.2 Rewrite `Register`/`Login`/`Me` in `auth.go` to: `BodyParser`; call `h.accounts.Register/Login/UserByID`; map sentinels → status (`ErrInvalidEmail`/`ErrPasswordTooShort`→400, `ErrInvalidCredentials`→401, `ErrEmailTaken`→409, `ErrUserNotFound`→401); on success `setSession` (Register/Login) and render `userResponse` (201 for Register, 200 for Login/Me). Keep `Logout` as-is.
- [x] 3.3 Change `userResponse.CreatedAt` to `*time.Time` and map from the service `User`. Remove `normalizeEmail`/`isUniqueViolation` from the handler if no longer used (check OAuth/other handlers first — `isUniqueViolation` may now be unused since OAuth moved; remove if so).
- [x] 3.4 Confirm `go build ./...`, `go vet ./...`, `go test ./internal/handler/ ./internal/accounts/` green. Check the existing auth unit tests (`auth_test.go`) construct a `Handler` with the service wired (or only hit pre-service reject paths) — give `authApp`/similar a service with a fake repo+hasher if a path now reaches `h.accounts`.

## 4. Verify

- [x] 4.1 `go test -tags=integration ./internal/handler/` (real Postgres) — register/login/me end-to-end and the auth integration tests stay green (needs Docker; else CI before merge). Migrate any integration test that called the old `h.queries`-based path or a removed helper.
- [x] 4.2 `gofmt -l` clean; `go build ./... && go vet ./...` clean.
- [x] 4.3 Self-review: `internal/accounts` imports no `auth`/`fiber`/`pgx`/`pgtype`/`oauth`; no validation/credential/normalization rule left in `auth.go`; status codes and the user wire shape unchanged.
