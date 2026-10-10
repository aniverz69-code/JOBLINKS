## Context

The backend already has an enrichment worker (`internal/enrich` + `cmd/enrich`)
and a job read API, but no ingest: jobs only arrive hand-seeded. The CLAUDE.md
seam reserves `internal/sources/` (parsers as interface + registry) and
`internal/pipeline/` (fetch → normalize → dedup → upsert) for exactly this, and
notes that ingest should enqueue into `enrichment_outbox` in the same transaction
as the upsert. The enrichment system is the structural model to mirror: a small
domain interface (`Provider`), a `Runner`, a `Store` interface decoupling the DB,
and a thin scheduled binary.

The chosen source model is the **ATS-platform-adapter** model: a `Source` is a
platform (Greenhouse, Lever, Ashby); the unit of work is a company board listed in
config. One adapter serves many boards. This first change ships the framework plus
three adapters.

## Goals / Non-Goals

**Goals:**

- A parser framework where adding a platform is one new file plus one registration
  line, and adding a company is a config edit with no code.
- An end-to-end ingest path: config → fetch → normalize → dedup-upsert →
  transactional enrichment enqueue → standalone scheduled command.
- Preserve enrichment across re-ingests; never let ingest wipe enrichment.
- Mirror the enrichment module's seams (interface + Store + thin binary) so the two
  pipelines stay symmetric and independently testable.

**Non-Goals:**

- Incremental/delta fetch (every run fetches all postings; dedup is by upsert).
- Per-source rate limiting, headless browsers, platform authentication.
- Per-posting description detail-fetch (these three platforms carry the description
  in their list endpoints).
- DB-backed source config or an admin API for boards.
- Re-enrichment when a posting's content changes (bump `enrich.Version` for a
  global re-enrich instead).

## Decisions

### Source contract and registry

```go
type CompanyEntry struct { Company, Provider, Board string }

type Source interface {
    Provider() string
    Fetch(ctx context.Context, e CompanyEntry) ([]Job, error)
}

type Job struct {
    ExternalID, URL, Title, Company, Location, Description string
    Remote   bool
    PostedAt *time.Time
}
```

Adapters take an `HTTPClient` in their constructor (`NewGreenhouse(c)`), so tests
inject a fake and never touch the network. The registry is an **explicit
constructor** `sources.All(c) map[string]Source` — chosen over `init()`
self-registration because the rest of the codebase wires dependencies explicitly
(e.g. `NewLangChainProvider`), explicit assembly is trivially testable, and it
avoids hidden global state. Trade-off: adding a platform touches one central line;
accepted as the cheaper, clearer option.

### Namespaced dedup key

`source` stays the bare platform (`greenhouse`) so `jobs_source_idx` and
`source`-filtered queries keep working. The pipeline (not the adapter) namespaces
`external_id` as `"<board>:<native-id>"`, keeping namespacing policy in one place
and adapters dumb. This removes the cross-company collision that a bare platform
`source` + native ATS id would create, since ATS ids are unique only within a
board.

### Write path: preserve enrichment, enqueue transactionally

`UpsertJob`'s `ON CONFLICT DO UPDATE` currently full-replaces the enrichment
columns from `EXCLUDED`. Ingest carries no enrichment, so re-ingest would wipe it.
Decision: drop the enrichment columns from the conflict update so re-ingest leaves
`enrichment`/`enriched_at`/`enrichment_version` untouched; the enrichment worker's
`SetJobEnrichment` remains their sole writer. On first insert the columns take their
table defaults (`'{}'` / NULL / 0).

The pipeline's `Store.Save` runs in one pgx transaction: `UpsertJob` returns the
row, then a new gated enqueue query inserts into `enrichment_outbox` only when
`enriched_at IS NULL OR enrichment_version < target`, `ON CONFLICT (job_id,
target_version) DO NOTHING`. This honors the transactional-outbox seam while
reusing the same "needs enrichment" predicate the backfill enqueue already uses, so
already-enriched jobs are not re-queued on every run.

### Run model

`cmd/ingest` mirrors `cmd/enrich`: load config, build the pool, build
`sources.All(httpClient)`, run the pipeline once, log `ingested`/`failed`, exit.
Boards run through a bounded worker pool; a per-board failure is recorded and
skipped (mirrors `enrich`'s per-entry `fail`), never aborting the run.

### HTTP transport

A shared client in `internal/sources/http.go`: a `net/http.Client` with a timeout,
a project User-Agent, and a small retry-with-backoff on transient (5xx/timeout)
responses, behind an `HTTPClient` interface the adapters depend on. No SSRF
allowlist: URLs are built from our own trusted `sources.yml` plus known platform
hostnames, not user input.

## Risks / Trade-offs

- **Re-enrichment on content change is unaddressed** → A posting whose description
  changes after enrichment keeps stale enrichment until `enrich.Version` bumps.
  Accepted as an explicit non-goal; the version bump is the escape hatch.
- **Full-fetch every run** → Re-downloads all postings each run. Fine at MVP board
  counts; revisit with incremental fetch when volume warrants.
- **`UpsertJob` signature change** → Dropping enrichment from the conflict update
  changes generated code and any caller. Mitigation: `UpsertJob` has no production
  caller yet (enrichment writes via `SetJobEnrichment`); verify callers during
  implementation and adjust tests.
- **List-endpoint description assumption** → If a platform's list endpoint omits
  descriptions, enrichment input is empty for it. Mitigation: verified true for
  Greenhouse (`?content=true`), Lever, and Ashby; a future platform needing a
  detail-fetch is a localized adapter change.

## Migration Plan

No schema migration: reuses `jobs` and `enrichment_outbox`. Steps: edit
`queries/jobs.sql` (conflict update + new enqueue query), `make sqlc`, build the
sources/pipeline packages and `cmd/ingest`, add `sources.yml`. Rollback is reverting
the change; the only persistent behavior change is that re-ingest no longer wipes
enrichment, which is strictly safer.

## Open Questions

None outstanding — the source model, config location, scope (three adapters),
registry style, and the enrichment-preservation decision were settled during
brainstorming.
