## 1. Prioritized, open-only claim & enqueue (SQL)

- [x] 1.1 Write a failing integration test (`internal/db`, `-tags=integration`) asserting `ClaimEnrichmentBatch` returns entries ordered by job `posted_at DESC NULLS LAST, id DESC` and excludes entries whose job has `closed_at IS NOT NULL`.
- [x] 1.2 Update `ClaimEnrichmentBatch` in `internal/db/queries/enrichment.sql`: `JOIN jobs`, `AND j.closed_at IS NULL`, `ORDER BY j.posted_at DESC NULLS LAST, j.id DESC`, `FOR UPDATE OF o SKIP LOCKED`.
- [x] 1.3 Update `EnqueuePendingJobs`: add `AND closed_at IS NULL`. Add/extend an integration test asserting a closed job is not enqueued.
- [x] 1.4 Run `make sqlc`; commit regenerated `internal/db`. Confirm `go build ./...` and the integration tests pass.

## 2. Concurrency config

- [x] 2.1 Write a failing test in `internal/config/enrich_test.go` for `ENRICH_CONCURRENCY` (default 4, override honored) and remove the `BatchSize` assertions/`ENRICH_BATCH_SIZE` case.
- [x] 2.2 In `internal/config/enrich.go`: add `Concurrency` (env `ENRICH_CONCURRENCY`, default 4); remove `BatchSize`/`ENRICH_BATCH_SIZE`.

## 3. Concurrent drain (runner)

- [x] 3.1 Make the fake `Store` in `internal/enrich/runner_test.go` thread-safe (mutex) and add a failing test that a multi-entry wave is drained concurrently with correct enriched/failed/dead-lettered tallies.
- [x] 3.2 In `internal/enrich/runner.go`: replace `RunOptions.BatchSize` with `Concurrency`; pass `opt.Concurrency` as the claim limit; replace the inner sequential `for entry { process }` with a bounded pool of `Concurrency` goroutines; guard `run.stats` with a `sync.Mutex`.
- [x] 3.3 Re-run `go test ./internal/enrich/...`; confirm green and `go test -race` clean.

## 4. Wire the worker

- [x] 4.1 In `cmd/enrich/main.go`: pass `Concurrency` into `RunOptions`; drop `BatchSize`.
- [x] 4.2 `go build ./... && go vet ./...`; confirm the worker compiles and `dbStore.Claim` receives the concurrency-sized limit.

## 5. Verify

- [x] 5.1 Run `go test ./...` and `go test -tags=integration ./internal/db/`; confirm all green.
- [x] 5.2 Grep the repo for stale `ENRICH_BATCH_SIZE` / `BatchSize` references (deploy docs, compose, AGENT.md) and update them.
