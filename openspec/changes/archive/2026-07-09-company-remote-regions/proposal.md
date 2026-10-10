## Why

We have a curated public list (the Atul Kumar "remote companies" directory, 25
pages, ~700 companies) mapping each company to the regions where it hires
remotely. This is a *declared company fact* ("this company hires remotely in
EU") — distinct from the job-derived `companies.regions` facet ("regions our
current postings sit in"), which the periodic recompute owns and would overwrite.
Capturing it lets users filter the company catalogue to employers open to remote
hiring in their region — a signal we cannot derive from the jobs we happen to
have crawled.

## What Changes

- Add a **`companies.remote_regions text[]`** column (values from
  `enrich.RegionValues`) that the periodic facet recompute (`recount-companies` /
  `RefreshCompanyFacets`) **never touches** — a curated, backfilled facet
  independent of the job-derived arrays. The raw source string is retained in
  `company_info.remote_regions_raw` for mapping audit.
- Add a checked-in dataset **`sources/remote-companies.jsonl`** — the transcribed
  PDF as `{name, website, region}` records — so the backfill is reproducible and
  the source is documented.
- Add **`internal/remoteregion`**: a pure, best-effort curated dictionary
  `Map(raw string) []string` that maps a free-text region label to macro-region
  codes (`Worldwide→[global]`, `Europe→[eu]`, `Americas→[north_america,latam]`,
  timezone/narrow-geo to the nearest macro region, unrecognized → `[]`).
- Add **`cmd/backfill-remote-regions`**: a run-once-and-exit worker that reads the
  dataset, maps each region string, matches companies by `normalize.Slug(name)`,
  and updates **existing companies only** (no reference rows for unmatched
  entries), reporting matched / unmatched / mapped / unmapped counts.
- Add a **`remote_regions` facet param** to `GET /api/v1/companies` (array
  overlap, empty = no-op), composing with the existing facets exactly like
  `regions`/`countries`.
- Add a **`remote_regions` filter facet** to the companies filter UI in `web/`.

## Capabilities

### New Capabilities

- `company-remote-regions`: the curated remote-hiring-regions dataset, the
  best-effort region-string mapping dictionary, the backfill worker, and the
  ownership rules for the `remote_regions` column (backfilled, not job-derived,
  and excluded from the facet recompute).

### Modified Capabilities

- `companies`: the company list endpoint (`GET /api/v1/companies`) gains a
  `remote_regions` facet query parameter, and the denormalized-facet recompute
  requirement is amended to exclude the new curated column.

## Impact

- **Schema**: new migration adding `companies.remote_regions`.
- **DB access**: new `SetCompanyRemoteRegions` (UPDATE-only) query; the
  `ListCompanies`/`CountCompanies` queries gain a `remote_regions` facet param.
- **New code**: `internal/remoteregion`, `cmd/backfill-remote-regions`,
  `sources/remote-companies.jsonl`.
- **HTTP**: `GET /api/v1/companies` accepts `remote_regions`.
- **Web**: companies filter UI adds a remote-regions facet.
- **Ops**: a one-time (repeatable) worker run to load the dataset; the recompute
  worker is unaffected (must not clobber the new column).
- **Seam (out of scope)**: matching is slug-only — no website/domain fallback for
  companies whose normalized name differs from ours.
