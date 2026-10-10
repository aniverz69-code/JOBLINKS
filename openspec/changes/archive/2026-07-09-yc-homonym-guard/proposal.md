## Why

`cmd/import-yc` matches yc-oss entries to our companies by `normalize.Slug(name)`,
which conflates homonyms: a tiny/defunct YC startup literally named "Meta"
(metavision.com, team 11, Inactive) collided with Facebook's Meta, stamping it
with a bogus YC batch/status/badge — as did "Benchmark" (YC team 7 ≠ Benchmark
Capital), "Archer" (team 4 ≠ Archer Aviation), "Corsair" (team 2 ≠ Corsair
Gaming). These show wrong YC facets on prominent non-YC companies.

## What Changes

- **Add a homonym guard to `cmd/import-yc`**: when an entry matches an **existing
  job-backed** company, skip applying YC data if the company plainly dwarfs the YC
  startup — `job_count > team_size` (with a small floor) means we hold far more
  open jobs than the YC company has employees, so it is not the same entity. The
  count is reported (`collisions`). Reference-row inserts (unambiguous, exist only
  because of YC) are never guarded.
- **One-off prod cleanup**: clear `yc_batch`/`yc_status`/`yc_stage`/`yc_flags` on
  companies the guard now rejects.

## Capabilities

### Modified Capabilities

- `yc-company-enrichment`: the importer's matching gains a homonym guard that
  skips enriching an existing company that dwarfs the matched YC startup.

## Impact

- **Code**: `cmd/import-yc` (guard + a `CompanyJobCountBySlug` lookup), new query,
  regenerate `internal/db`.
- **Data/prod**: re-run `import-yc` (collisions now skipped) + a cleanup pass.
- **Known limit**: two comparably-sized homonyms (e.g. zip.co vs YC ziphq, team
  500) are not separated by the size heuristic — a rarer case, deferred to a
  curated blocklist if it matters.
