## Why

The HTTP layer mixes three responsibilities. Handlers hold the `pgxpool.Pool` and a concrete `*db.Queries`, and the per-user job-tracking rules — stage validation, the "nil field left unchanged" partial-update semantics, the slug→id resolution, and the `pgtype` ↔ JSON conversions — live directly in `internal/handler/user_jobs.go`. Those rules can only be exercised through a full Fiber request, so they are awkward to unit-test and impossible to reuse outside HTTP. This is the first, lowest-risk slice of finding #2: extract one use-case service behind a narrow repository interface to establish the pattern (service + repo interface + thin handler) before touching the riskier auth/OAuth code.

## What Changes

- Introduce `internal/jobtracking` — a `Service` that owns the per-user job-interaction use cases (record view, mark applied, save/unsave, track stage+notes) and their rules, depending on a narrow `Repository` interface rather than `*db.Queries` directly.
- Move the business rules out of the handler into the service: stage validation (already delegating to `userjob.ValidStage`), the partial-update "nil leaves unchanged" semantics, slug→id resolution, and the unsaved-row → zero-interaction idempotency.
- The service returns plain Go result structs; the `pgtype` ↔ wire-shape conversion stays a thin mapping at the handler boundary (transport concern).
- `internal/handler/user_jobs.go` becomes thin: parse request, read auth, call the service, render the JSON envelope. The handler no longer references `*db.Queries` for these endpoints.
- A `Repository` adapter over `*db.Queries` wires the existing generated queries to the new interface (no SQL changes).
- Unit tests for the service exercise the rules against a fake `Repository`, with no Fiber and no database.
- No HTTP API behavior change, no schema change, no SQL change.

## Capabilities

### New Capabilities
<!-- None. This is an internal refactor; no new user-facing capability. -->

### Modified Capabilities
<!-- None. The `user-job-tracking` spec's REQUIREMENTS are unchanged — the
     view/apply/save/track endpoints keep identical request/response behavior,
     auth, idempotency, and the controlled stage vocabulary. This change is a
     pure internal restructuring (no spec delta; archive with --skip-specs). -->

## Impact

- **New package:** `internal/jobtracking` (`Service`, `Repository` interface, result structs, a `*db.Queries` adapter, unit tests).
- **Modified:** `internal/handler/user_jobs.go` (thinned to transport), `internal/handler/handler.go` (the `Handler` constructs/holds the service; `RecordView`/`MarkApplied`/`SaveJob`/`UnsaveJob`/`TrackJob` call it). The `Handler`'s direct `*db.Queries` use for these endpoints is removed; other handlers are untouched in this slice.
- **Unchanged:** routes and middleware wiring, the `user_jobs` table and its queries, the public wire shape (`interactionResponse`), auth (`RequireAuthOrKey`), and the `userjob.Stages` vocabulary.
- **Out of scope (later slices of #2):** `AuthService` (OAuth transactions / account-linking) and `JobQueryService` (list/search/company reads); reducing `Register`'s parameter count.
