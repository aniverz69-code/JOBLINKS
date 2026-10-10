## 1. Job-count lookup

- [x] 1.1 Add a `CompanyJobCountBySlug :one` query (returns `job_count` for a slug,
      `pgx.ErrNoRows` if absent); regenerate `internal/db`.

## 2. Importer guard

- [x] 2.1 RED: extend the `cmd/import-yc` loader test — a matched existing company
      whose job_count exceeds the entry's team_size (above the floor) is skipped as a
      collision (facets not written), counted in a new `collisions` stat; a matched
      company with team_size ≥ job_count is still enriched.
- [x] 2.2 GREEN: in resolve/load, fetch the matched company's job_count and skip the
      upsert when `team_size > 0 && job_count > team_size && job_count >= floor`;
      report `matched/inserted/collisions/skipped`.
- [x] 2.3 REFACTOR + simplify.

## 3. Finish + deploy

- [x] 3.1 `go build/vet/test` green.
- [ ] 3.2 Deploy; re-run `cmd/import-yc` (collisions skipped); one-off prod cleanup
      clearing the four `yc_*` columns on companies the guard rejects; verify Meta et
      al. no longer carry YC facets while legit YC companies keep them.
