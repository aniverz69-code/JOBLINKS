## Why

`cmd/import-yc` already loads the yc-oss directory but leaves several high-value
fields on the table. Two improve data quality and discovery at low cost:
`former_names` (present on 48% of entries) lets us match a company we ingest under
a renamed/old name instead of inserting a duplicate reference row; and `stage`
(Early/Growth), `top_company` (YC's curated marquee list), and `isHiring` are
signals users would filter and browse by. Logos are already handled (logo.dev by
name), so this change is data + facets + a couple of company-page badges.

## What Changes

- **Match on `former_names`**: `cmd/import-yc` resolves an entry to an existing
  company by its current name **or any former name** before deciding update vs
  insert — fewer duplicate YC reference rows, more enrichment landing on real
  companies.
- **Richer industries**: the `ycdir` mapping unions `industry` + `industries[]` +
  `subindustry` + `tags` (de-duplicated) instead of just `industry` + `tags`.
- **New curated facets**: `companies.yc_stage text[]` (Early/Growth) and
  `companies.yc_flags text[]` (`top_company`, `hiring`) — importer-owned,
  recompute-exempt — filterable by overlap on `GET /api/v1/companies` and in the
  companies FilterModal.
- **Company-page badges**: surface stage, "YC Top Company", and "Hiring" on the
  company detail view from the stored data.

## Capabilities

### Modified Capabilities

- `yc-company-enrichment`: the importer additionally matches by former name, the
  mapping unions the richer industry fields, and it populates the new `yc_stage`/
  `yc_flags` curated facets.
- `companies`: the list endpoint gains `yc_stage` and `yc_flags` facet parameters.

## Impact

- **Schema**: migration adding `companies.yc_stage text[]` + `yc_flags text[]`.
- **DB access**: `UpsertYCCompany` gains the two facet columns; `ListCompanies`/
  `CountCompanies` gain two facet params.
- **Code**: `internal/ycdir` (former-name slugs, richer industries, stage, flags),
  `cmd/import-yc` (former-name matching), regenerate `internal/db`.
- **Web**: two new company filters + company-page badges.
- **Data/prod**: re-run `cmd/import-yc`; the former-name matching reduces the
  reference-row count on re-run; the recompute leaves the new columns alone.
- **Non-goal**: yc-oss `small_logo_thumb_url` — logos are already served by logo.dev.
