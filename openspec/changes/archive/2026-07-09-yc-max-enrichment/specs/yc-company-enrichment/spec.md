## MODIFIED Requirements

### Requirement: yc-oss entries map to company-info fields

The system SHALL provide a pure mapping from a yc-oss directory entry
(`yc-oss.github.io/api/companies/all.json`) to company-info fields: `one_liner` →
tagline, `long_description` → a `company_info.description`, the union of `industry`,
`industries`, `subindustry`, and `tags` (de-duplicated) → industries, `team_size` →
employee count, `launched_at` → founding year, and `all_locations` → an HQ country
resolved via `internal/location` (the first resolved country, or none). The mapping
SHALL also carry `batch`, `status`, and `stage` through, expose the normalized slug
of each `former_names` entry (for matching), and derive a `flags` set holding
`top_company` when the entry is a YC top company and `hiring` when it is hiring. It
SHALL place `website`, the YC profile url, and the logo into `company_info`. An entry
with a blank name SHALL be skipped.

#### Scenario: A directory entry maps to the expected fields

- **WHEN** the mapper is given an entry with `name`, `one_liner`, `long_description`,
  `team_size`, `launched_at`, `industry`, `industries`, `subindustry`, `tags`,
  `all_locations`, `batch`, `status`, `stage`
- **THEN** it yields a record whose tagline is the `one_liner`, whose industries are
  the de-duplicated union of `industry`/`industries`/`subindustry`/`tags`, whose
  employee count is `team_size`, whose founding year is derived from `launched_at`,
  whose HQ country is resolved from `all_locations`, and whose `batch`/`status`/
  `stage` are carried through

#### Scenario: Former names are exposed as slugs

- **WHEN** the mapper is given an entry whose `former_names` contains `Facebook`
- **THEN** the record's former-name slugs include `facebook` (normalized)

#### Scenario: Flags reflect top-company and hiring

- **WHEN** the mapper is given an entry with `top_company = true` and `isHiring = true`
- **THEN** the record's flags are `{hiring, top_company}`, and an entry with neither
  has empty flags

#### Scenario: Missing optional fields become absent, not empty sentinels

- **WHEN** the mapper is given an entry with an empty `long_description`, unknown
  `all_locations`, and zero `team_size`
- **THEN** the record omits the description, leaves HQ country unset, and leaves
  employee count unset (no zero/empty placeholder)

### Requirement: The YC directory importer enriches existing companies and adds the rest

The system SHALL provide a run-once worker (`cmd/import-yc`) that fetches the
yc-oss directory, maps each entry, and resolves it to a company by its current-name
slug **or any former-name slug** — the first that matches an existing company — and
upserts there: an existing company has its company-info columns and curated YC
facets refreshed, and an entry matching no existing company (by any name) is inserted
as a reference row (`is_reference = true`) with no jobs under its current-name slug,
so the full YC directory is held. The upsert SHALL NOT modify a company's
`job_count`, `collections`, or job-derived facet arrays. The worker SHALL be
idempotent — re-running rewrites the same values — and SHALL report matched vs
inserted counts.

#### Scenario: Existing company is enriched

- **WHEN** the worker processes an entry whose normalized name matches a company row
- **THEN** that company's tagline, industries, employee count, founding year, HQ
  country, `company_info.description`, and curated YC facets are set, and its
  `job_count`/`collections`/job-derived facets are unchanged

#### Scenario: A former name matches an existing company instead of inserting a duplicate

- **WHEN** the worker processes an entry whose current name matches no company but
  whose `former_names` slug matches an existing company
- **THEN** that existing company is enriched (no new reference row is inserted), and
  its display `name` is left unchanged

#### Scenario: Unmatched entry is inserted as a reference row

- **WHEN** the worker processes an entry whose current name and every former name
  match no company
- **THEN** a `companies` row is inserted with `is_reference = true` under the
  current-name slug, carrying the mapped company-info and yc facets, and `job_count = 0`

#### Scenario: Re-running is idempotent

- **WHEN** the worker runs twice over the same directory
- **THEN** the second run writes the same company-info and yc facet values as the first

### Requirement: yc_batch and yc_status are curated facets exempt from the recompute

The system SHALL store `companies.yc_batch`, `yc_status`, `yc_stage`, and `yc_flags`
(each a `TEXT[]`) as curated facets loaded by the YC importer: the company's YC batch
(e.g. `Winter 2012`), status (e.g. `Active`), stage (`Early`/`Growth`), and flags
(`top_company`, `hiring`). These SHALL be filterable by array overlap on the company
list endpoint. The periodic facet recompute (`RefreshCompanyFacets` /
`cmd/recount-companies`) SHALL NOT read or write these columns, so an imported value
survives every recompute. A company never touched by the importer SHALL have all four
empty (`'{}'`).

#### Scenario: Recompute leaves the YC facets untouched

- **WHEN** a company has imported `yc_batch`/`yc_status`/`yc_stage`/`yc_flags` values
  and the facet recompute runs
- **THEN** all four columns are unchanged, regardless of the company's jobs

#### Scenario: A non-YC company has empty YC facets

- **WHEN** a company the importer never matched is read
- **THEN** its `yc_batch`, `yc_status`, `yc_stage`, and `yc_flags` are all empty
