## Why

The `enrichment_outbox` holds ~313k unenriched jobs, but `cmd/enrich` drains it
strictly sequentially (one ~45s LLM call at a time ≈ 1,900 jobs/day) while ingest
adds jobs faster — so the backlog grows without bound. Worse, the claim orders by
insertion id (not freshness) and never excludes closed postings, so fresh, live
jobs sit behind a long tail of stale and dead ones, and LLM budget is spent
enriching vacancies no user will ever see.

## What Changes

- **Prioritize fresh, open jobs.** The claim joins `jobs` and orders by
  `posted_at DESC NULLS LAST, id DESC` (the order the user sees in the feed), so
  the newest postings are enriched first.
- **Stop enriching closed jobs.** Both the enqueue backfill and the claim exclude
  `closed_at IS NOT NULL` jobs. Closed orphan rows already sitting in the outbox
  are simply never claimed (left as harmless cruft — a known seam).
- **Drain with bounded concurrency.** The worker processes a claim wave across
  `ENRICH_CONCURRENCY` goroutines (default 4) instead of one-at-a-time, lifting
  throughput ~×4 (~1,900 → ~7,600 jobs/day). The claim wave is sized to the
  concurrency so each lease window stays well under `LeaseSeconds`, keeping
  exactly-once-per-run guarantees intact even when cron runs overlap.
- **BREAKING (operational only):** `ENRICH_BATCH_SIZE` is removed; the claim wave
  is now sized by `ENRICH_CONCURRENCY`. No API or data change.

## Capabilities

### New Capabilities

(none — this refines an existing capability)

### Modified Capabilities

- `ai-enrichment`: the enqueue and claim now exclude closed jobs and order claims
  by job freshness; the batch command drains a claim wave concurrently rather than
  sequentially, with the wave sized by a configurable concurrency.

## Impact

- **Code:** `internal/db/queries/enrichment.sql` (Claim + Enqueue) → `make sqlc`
  regenerates `internal/db`; `internal/enrich/runner.go` (concurrent drain,
  mutex-guarded stats); `internal/config/enrich.go` (+`enrich_test.go`) (add
  `Concurrency`, drop `BatchSize`); `cmd/enrich/main.go`; `internal/enrich/runner_test.go`
  (thread-safe fake `Store`).
- **Config:** new `ENRICH_CONCURRENCY` (default 4); `ENRICH_BATCH_SIZE` removed.
- **Data/migrations:** none — relies on the existing `jobs_posted_at_id_idx` and
  the `outbox(job_id)` index from `UNIQUE (job_id, target_version)`.
- **No HTTP/API surface change.**
