## 1. Schema

- [x] 1.1 Add migration `migrations/0005_company_remote_regions.sql` adding
      `companies.remote_regions text[] NOT NULL DEFAULT '{}'`
- [x] 1.2 Regenerate `internal/db` via `make sqlc`; confirm `db.Company` gains the
      `RemoteRegions` field and the build passes

## 2. Region mapping dictionary (`internal/remoteregion`)

- [x] 2.1 RED: write table-driven unit tests for `Map(raw string) []string`
      covering clean labels (`Europe→[eu]`, `Worldwide→[global]`,
      `USA`/`North America→[north_america]`), composite (`Americas→{north_america,
      latam}`), timezone/narrow-geo best-effort (`Pacific Time Zone→[north_america]`,
      `CET…→[eu]`, `Western Asia→[mena]`), unrecognized→`[]`, dedup, and
      vocabulary-confinement (every output in `enrich.RegionValues`)
- [x] 2.2 GREEN: implement `remoteregion.Map` as a pure curated dictionary keyed on
      a normalized (lowercased/trimmed) label, splitting comma-separated composites
- [x] 2.3 REFACTOR + simplify; tests stay green (reviewed: fixed TZ-span both-edge
      logic, `pacific`→`pacific time`, gated boundary tz codes, dropped dead rules)

## 3. Persistence query (`SetCompanyRemoteRegions`)

- [x] 3.1 Add `SetCompanyRemoteRegions` to `internal/db/queries/companies.sql`:
      UPDATE-only by slug, setting `remote_regions` and merging
      `remote_regions_raw` into `company_info` (jsonb), bumping `updated_at`;
      `:execrows` so callers see matched vs unmatched. Do NOT touch name/job_count/
      collections/is_reference/job-derived facets
- [x] 3.2 Regenerate `internal/db` via `make sqlc`
- [x] 3.3 Integration test (`//go:build integration`, testcontainers): existing
      company gets `remote_regions` + `remote_regions_raw` and nothing else changes;
      an unmatched slug returns 0 rows and inserts no company

## 4. Dataset (`sources/remote-companies.csv`)

- [x] 4.1 Place the externally-provided CSV at `sources/remote-companies.csv`
      (header columns `Name`, `Website`, `Region`); document the source
      (transcribed from the Atul Kumar remote-companies directory, 734 rows)
- [x] 4.2 Add a test that loads the checked-in CSV and asserts every row parses and
      every region string maps (dataset↔dictionary guard: skipped=0, unmapped=0)

## 5. Backfill worker (`cmd/backfill-remote-regions`)

- [x] 5.1 RED: unit-test the loader against a fake store — matched record updates,
      unmatched record is a no-op counted as unmatched, unmapped region yields empty
      set with raw preserved, idempotent re-run, stat tallies (matched/unmatched/
      mapped/unmapped)
- [x] 5.2 GREEN: implement `cmd/backfill-remote-regions` using `worker.Main`/
      `worker.Bootstrap`, reading the CSV via `encoding/csv`, `remoteregion.Map` +
      `normalize.Slug`, `SetCompanyRemoteRegions`, mirroring `cmd/backfill-company-info`
      structure
- [x] 5.3 REFACTOR + simplify; tests stay green

## 6. Recompute guard

- [x] 6.1 Integration test asserting `RefreshCompanyFacets` leaves a company's
      backfilled `remote_regions` untouched (and confirm the query does not
      reference the column)

## 7. Company list facet (API)

- [x] 7.1 Add a `remote_regions text[]` facet param to `ListCompanies` and
      `CountCompanies` in `companies.sql` (identical `&&` overlap + empty short-circuit
      as the existing facets; keep both WHEREs identical); regenerate via `make sqlc`
- [x] 7.2 Wire `remote_regions` through the companies list handler (parse repeatable
      query param, pass to both queries)
- [x] 7.3 Handler test: `GET /api/v1/companies?remote_regions=eu` filters by the
      column and reports the filtered `meta.total`; independence from `regions`

## 8. Web filter UI

- [x] 8.1 Add a `remote_regions` facet to the companies filter (FilterModal): pills
      from the macro-region vocabulary, URL-synced param, counted toward the active
      filter total, mirroring the existing company `regions` facet
- [x] 8.2 Verify (svelte-check + visual/manual) the facet selects and filters the
      companies list

## 9. Finish

- [x] 9.1 `go build ./... && go vet ./... && go test ./...` green; run the
      backfill against a dev DB and confirm matched/unmatched counts are sane
- [x] 9.2 Update `CLAUDE.md`/`AGENT.md` layout + conventions for the new column,
      package, dataset, and worker
