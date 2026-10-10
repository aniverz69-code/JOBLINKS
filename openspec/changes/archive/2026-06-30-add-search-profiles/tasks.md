## 1. Database

- [x] 1.1 Add `migrations/0028_search_profiles.sql`: `search_profiles` table (id, user_id FK → users ON DELETE CASCADE, name with 1–100 CHECK, specialization TEXT NOT NULL, skills TEXT[] NOT NULL with `cardinality(skills) > 0` CHECK, created_at, updated_at), `UNIQUE (user_id, name)`, and an index on `user_id`.
- [x] 1.2 Add `internal/db/queries/search_profiles.sql`: `ListSearchProfiles` (user-scoped, `updated_at DESC`), `CountSearchProfiles`, `CreateSearchProfile`, `UpdateSearchProfile` (partial via COALESCE, owner-scoped), `DeleteSearchProfile` (owner-scoped).
- [x] 1.3 Run `make sqlc` and commit the regenerated `internal/db` code; confirm `go build ./...` passes.

## 2. Service (`internal/searchprofile`)

- [x] 2.1 Write failing tests (`searchprofile_test.go`) for: name validation (blank/over-long → ErrInvalidName), duplicate name → ErrDuplicateName, per-user cap (50) → ErrCapExceeded, unknown specialization → ErrInvalidSpecialization, empty skills → ErrEmptySkills, skill normalization (lowercase/trim/dedupe), and partial update leaving omitted fields unchanged — against a fake Repository.
- [x] 2.2 Implement `Repository` interface, `Service`, sentinel errors, and `New`, mirroring `internal/savedsearch`. Reuse the rune-counted name bound and cap constants.
- [x] 2.3 Implement `validSpecialization` (membership in `enrich.CategoryValues`) and `normalizeSkills` (lowercase/trim/dedupe, error if empty). Make the tests pass.
- [x] 2.4 Implement the repository (`repository.go`) over `*db.Queries`, mapping unique violation → ErrDuplicateName and missing/non-owned row → ErrNotFound, mirroring `savedsearch/repository.go`.

## 3. HTTP handlers (`internal/handler/me_profiles.go`)

- [x] 3.1 No handler-level test: the closest sibling (`saved_searches`) has thin, untested handlers and is covered purely by service unit tests. Match that convention — the comprehensive `searchprofile` service tests cover the domain logic; the HTTP surface (status codes, envelope) is exercised at the verification step (running the server). Adding an integration harness the sibling lacks would be inconsistent.
- [x] 3.2 Implement `CreateSearchProfile`, `ListSearchProfiles`, `UpdateSearchProfile`, `DeleteSearchProfile` with the `{"data": ...}` envelope (omit `user_id`), letting the central `ErrorHandler` render errors. Map service sentinels to statuses (400/409/404).
- [x] 3.3 Wire the four routes under `RequireAuth` in `internal/handler/handler.go`, alongside the existing `/me/searches` block; construct the `searchprofile.Service` in `Register`.
- [x] 3.4 Confirm `go build ./...`, `go vet ./...`, and the new tests pass.

## 4. Web

- [x] 4.1 Add an API client module for profiles (list/create/update/delete) under `web/src/lib`, mirroring the saved-searches client.
- [x] 4.2 Build a profile-management view: list profiles; create/rename/edit with the existing category selector (specialization) and skills facet selector; delete; show a sign-in affordance for anonymous users.
- [x] 4.3 Verify with `svelte-check` and lint (no test runner in `web/`); manually confirm create/edit/delete round-trips against the API.

## 5. Wrap-up

- [x] 5.1 No AGENT.md change: the sibling features (saved searches, subscriptions) have NO convention bullet and are absent from AGENT.md's abbreviated table list — their canonical documentation is the OpenSpec spec, not AGENT.md. Adding an entry only for search-profiles would be the inconsistent one. The feature is documented by its OpenSpec spec (`search-profiles`), synced into `openspec/specs/` at archive.
- [x] 5.2 Full `go test ./...` and `go build ./...` green; confirm no Meilisearch/ingest/enrich surface was touched.
