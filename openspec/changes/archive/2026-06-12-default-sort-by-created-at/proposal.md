## Why

The SPA jobs list always goes through the search endpoint, and a query with no
text and no explicit sort falls back to Meilisearch "relevance" — which for an
empty query is effectively arbitrary order. Newly ingested jobs (including the
new Telegram source) don't surface on top; the catalogue reads as stale. The
DB-backed list orders by `posted_at`, which buries fresh ingests whose platform
date is old or missing.

## What Changes

- Default ordering is **time added to the catalogue** (`created_at DESC`):
  - Search endpoint: when no valid `sort` param is given **and the query text is
    empty**, sort by `created_at:desc`. A non-empty query keeps relevance order;
    an explicit `sort` always wins.
  - `created_at` joins the sortable allowlist (`?sort=created_at`) and the
    index's sortable attributes (the document already carries it via the job
    view).
  - DB-backed list (`GET /api/v1/jobs`): `ORDER BY created_at DESC, id DESC`.
- Requires a settings update + reindex (sortable attributes change); the
  reindex command already applies settings idempotently.

## Capabilities

### Modified Capabilities

- `job-search`: sortable attributes gain `created_at`; default ordering for a
  no-text, no-sort search is newest-first by `created_at`.

## Impact

- `internal/search/client.go` (sortable attributes), `internal/handler/search.go`
  (allowlist + default), `internal/db/queries/jobs.sql` (ListJobs order) + sqlc.
- Ops: run `cmd/reindex` once after deploy.
