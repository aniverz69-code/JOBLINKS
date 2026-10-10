## 1. DB layer — ledger query changes

- [x] 1.1 In `internal/db/queries/user_job_analysis.sql`, drop `created_at = now()` from `UpsertUserJobAnalysis`'s `ON CONFLICT DO UPDATE SET` so a recompute preserves the first-analysis time (update the query comment to match).
- [x] 1.2 Add `CountRecentUserJobAnalyses(user_id, since) :one` returning `bigint` — count of the user's rows with `created_at >= since`.
- [x] 1.3 Run `make sqlc` and commit the regenerated `internal/db` (new method + preserved-`created_at` behaviour).

## 2. Backend enforcement

- [x] 2.1 Add `fitAnalysisLimit = 10` and `fitAnalysisWindow = 30 * 24h` constants and extend the `jobFitStore` interface with `CountRecentUserJobAnalyses`; update the DB-less test fake to implement it and track a per-`(user, job)` row set.
- [x] 2.2 `GetJobFit`: compute `used` via `CountRecentUserJobAnalyses(user, now-window)` and include `quota { used, limit, remaining }` (`remaining = max(0, limit-used)`) in `jobFitResponse`; keep the read LLM-free. Test: quota reported; remaining floors at 0.
- [x] 2.3 `PostJobFit`: before the LLM call, look up the existing `(user, job)` row — a hit is a recompute (allow); a miss is a new job and, when `used >= limit`, return `fiber.NewError(429, …)` without invoking the LLM or persisting. Tests: new job under limit runs; new job over limit → 429; recompute over limit still runs; failed/unconfigured LLM leaves quota unchanged.
- [x] 2.4 `StreamJobFit`: run the same existence + quota check while the fiber ctx is valid (before `SetBodyStreamWriter`) and return `429` before opening the event stream for an over-limit new job. Test: over-limit new-job stream → 429, no LLM call.

## 3. Frontend

- [x] 3.1 Add `quota { used: number; limit: number; remaining: number }` to `JobFitResponse` in `web/src/lib/types.ts`.
- [x] 3.2 `JobFitAnalysis.svelte` (sidebar): show "N/10 used"; when `remaining == 0` and no analysis exists for this job (new job), show a limit notice instead of the compute CTA; a recompute of an already-analysed job stays available.
- [x] 3.3 `jobs/[slug]/fit/+page.svelte`: show usage; when it would be a new-job analysis and `remaining == 0`, render the limit message and do NOT open the EventSource; recompute of a cached analysis stays allowed.

## 4. Verification

- [x] 4.1 Backend: `go build ./... && go vet ./... && go test ./...` green. Web: `svelte-check`, web unit tests, and lint clean; visual-verify the sidebar + fit page usage/limit states.
