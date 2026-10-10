## 1. Pin current behavior

- [x] 1.1 Confirm the existing `internal/handler/user_jobs_integration_test.go` (`//go:build integration`, real Postgres) covers the success-path wire shape (job_id + RFC3339/null timestamps, save/unsave contract, 404). It is the end-to-end no-drift guard; record that it must pass via `go test -tags=integration ./internal/handler/` before merge (needs Docker). Do NOT rewrite it.
- [x] 1.2 After the mapping function is extracted (task 4.x), add a DB-free unit test pinning `Interaction → interactionResponse` JSON: exact field names, RFC3339 timestamp strings, and `null` for absent viewed/saved/applied/stage/notes. (Sequenced here for context; implement once the mapping exists.)

## 2. jobtracking package — contract and rules (TDD, fake repo)

- [x] 2.1 Create `internal/jobtracking` with the domain `Interaction` struct (`JobID int64`, `ViewedAt/SavedAt/AppliedAt *time.Time`, `Stage/Notes *string`) and the sentinel errors `ErrJobNotFound`, `ErrInvalidStage`, `ErrEmptyTrack`.
- [x] 2.2 Define the narrow `Repository` interface (`JobIDBySlug`, `RecordView`, `MarkApplied`, `SaveJob`, `UnsaveJob`, `TrackJob`) in the package.
- [x] 2.3 Write failing unit tests for `Service` against a fake `Repository`: record view, mark applied, save, each returning the mapped `Interaction`.
- [x] 2.4 Implement `Service.RecordView/MarkApplied/SaveJob` (resolve slug→id via repo; `ErrJobNotFound` on unknown slug) to pass 2.3.
- [x] 2.5 Write failing tests for `Unsave` idempotency (no row → zero `Interaction{JobID}`, nil error) and implement it.
- [x] 2.6 Write failing tests for `Track`: invalid stage → `ErrInvalidStage`; neither stage nor notes → `ErrEmptyTrack`; stage-only / notes-only / both leave the other field unchanged. Implement `Service.Track` (using `userjob.ValidStage`) to pass.

## 3. Persistence adapter

- [x] 3.1 Add a `Repository` adapter in `internal/jobtracking` wrapping `*db.Queries`, converting `db.UserJob` (pgtype) → `Interaction` and `db` errors → the package sentinels (`pgx.ErrNoRows` → `ErrJobNotFound` for slug lookup; no-row on unsave handled in the service).
- [x] 3.2 Confirm the package builds and `go test ./internal/jobtracking/` is green.

## 4. Thin the handler

- [x] 4.1 Construct the service in `handler.Register` (`jobtracking.New(jobtracking.NewQueriesRepository(queries))`) and store it on `Handler`.
- [x] 4.2 Rewrite `RecordView/MarkApplied/SaveJob/UnsaveJob/TrackJob` to: read `auth.UserID`, parse the body (track only), call the service with `(userID, slug, …)`, map domain errors (`ErrJobNotFound`→404, `ErrInvalidStage`/`ErrEmptyTrack`→400), and render `Interaction` → the existing `interactionResponse` JSON.
- [x] 4.3 Remove the now-dead inline rules from `user_jobs.go` (`validStages` is already gone; remove `interactionParams`/`trackRequest` plumbing superseded by the service, keeping only what transport still needs). The handler no longer calls `h.queries` for these endpoints.

## 5. Verify

- [x] 5.1 Run `go test -tags=integration ./internal/handler/` (real Postgres) and confirm the user_jobs wire contract from 1.1 stays green. If Docker is unavailable in the dev loop, note that this must pass in CI before merge; the mapping unit test (1.2) is the in-loop guard.
- [x] 5.2 `go build ./...`, `go vet ./...`, `go test ./...` all green.
- [x] 5.3 Self-review: confirm no `pgtype`/`pgx`/`fiber` import in `internal/jobtracking`, and no remaining business rule (stage check, partial-update, idempotency, slug resolution) in `user_jobs.go`.
