## MODIFIED Requirements

### Requirement: The YC directory importer enriches existing companies and adds the rest

The system SHALL provide a run-once worker (`cmd/import-yc`) that fetches the
yc-oss directory, maps each entry, and resolves it to a company by its current-name
slug **or any former-name slug** — the first that matches an existing company — and
upserts there: an existing company has its company-info columns and curated YC
facets refreshed, and an entry matching no existing company (by any name) is inserted
as a reference row (`is_reference = true`) with no jobs under its current-name slug,
so the full YC directory is held. To avoid homonym collisions (a well-known
non-YC company sharing a normalized name with a small YC startup), the worker SHALL
NOT enrich a matched **existing** company when that company plainly dwarfs the YC
entry — specifically when the company's open-job count exceeds the YC entry's team
size (above a small floor) — and SHALL count such skips separately; reference-row
inserts are never guarded. The upsert SHALL NOT modify a company's `job_count`,
`collections`, or job-derived facet arrays. The worker SHALL be idempotent —
re-running rewrites the same values — and SHALL report matched vs inserted vs
skipped-collision counts.

#### Scenario: Existing company is enriched

- **WHEN** the worker processes an entry whose normalized name matches a company row
  whose open-job count does not exceed the YC entry's team size
- **THEN** that company's tagline, industries, employee count, founding year, HQ
  country, `company_info.description`, and curated YC facets are set, and its
  `job_count`/`collections`/job-derived facets are unchanged

#### Scenario: A former name matches an existing company instead of inserting a duplicate

- **WHEN** the worker processes an entry whose current name matches no company but
  whose `former_names` slug matches an existing company
- **THEN** that existing company is enriched (no new reference row is inserted), and
  its display `name` is left unchanged

#### Scenario: A homonym collision is skipped, not enriched

- **WHEN** the worker matches an entry to an existing company whose open-job count
  exceeds the (known, non-zero) YC entry team size above the floor — e.g. a company
  with 620 open jobs matching a YC startup with 11 employees
- **THEN** the company's YC facets are left untouched and the entry is counted as a
  skipped collision, not applied

#### Scenario: Unmatched entry is inserted as a reference row

- **WHEN** the worker processes an entry whose current name and every former name
  match no company
- **THEN** a `companies` row is inserted with `is_reference = true` under the
  current-name slug, carrying the mapped company-info and yc facets, and `job_count = 0`

#### Scenario: Re-running is idempotent

- **WHEN** the worker runs twice over the same directory
- **THEN** the second run writes the same company-info and yc facet values as the first
