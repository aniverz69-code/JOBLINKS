## ADDED Requirements

### Requirement: Companies carry a curated remote-hiring-regions facet

The system SHALL store, on each `companies` row, a `remote_regions` array
(`TEXT[]`, values drawn from the macro-region vocabulary `enrich.RegionValues`)
recording the regions where the company hires remotely. This value SHALL be a
curated fact loaded by the remote-regions backfill — NOT derived from the
company's jobs — and SHALL be independent of the job-derived `regions` facet. A
company with no loaded value SHALL have an empty `remote_regions` array (`'{}'`),
which is the column default. The raw source region string SHALL be retained under
`company_info.remote_regions_raw` for mapping audit.

#### Scenario: Default is an empty array

- **WHEN** a company row exists that the remote-regions backfill has never touched
- **THEN** its `remote_regions` is the empty array `'{}'`

#### Scenario: The curated value is independent of job geography

- **WHEN** a company has `remote_regions` `{eu}` loaded and its open jobs derive
  `regions` `{north_america}`
- **THEN** both values coexist on the row, unchanged by each other

### Requirement: A curated dataset maps company names to remote-hiring region strings

The system SHALL keep a checked-in CSV dataset (`sources/remote-companies.csv`)
with header columns `Name`, `Website`, and `Region`, exported from the source
remote-companies directory. The dataset SHALL be the reproducible input to the
backfill worker, so the load is repeatable and the source is documented in the
repository. The `Region` cell SHALL be preserved verbatim from the source (its
normalization to macro-region codes is the mapping dictionary's responsibility).

#### Scenario: Dataset record shape

- **WHEN** the dataset is read
- **THEN** each row parses to a record exposing a company `Name` and a free-text
  `Region` string (with an optional `Website`)

### Requirement: A pure dictionary maps region strings to macro-region codes

The system SHALL provide a pure function `remoteregion.Map(raw string) []string`
that maps a free-text region label to zero or more macro-region codes from
`enrich.RegionValues`. The mapping SHALL be a curated, best-effort dictionary:
clean labels map directly (`Worldwide → [global]`, `Europe → [eu]`,
`USA`/`North America → [north_america]`), composite labels expand
(`Americas → [north_america, latam]`), and timezone or narrow-geography labels map
to the nearest macro region (`Pacific Time Zone → [north_america]`,
`CET… → [eu]`, `Western Asia → [mena]`). A label the dictionary cannot place
SHALL map to an empty slice (never a guess). The returned codes SHALL be
de-duplicated and SHALL contain only values in `enrich.RegionValues`.

#### Scenario: A clean label maps to one macro region

- **WHEN** `Map("Europe")` is called
- **THEN** it returns `["eu"]`

#### Scenario: A composite label expands to several macro regions

- **WHEN** `Map("Americas")` is called
- **THEN** it returns the set `{north_america, latam}`

#### Scenario: A worldwide label maps to global

- **WHEN** `Map("Worldwide")` is called
- **THEN** it returns `["global"]`

#### Scenario: An unrecognized label maps to nothing

- **WHEN** `Map` is called with a label the dictionary does not cover
- **THEN** it returns an empty slice (no region is guessed)

#### Scenario: Output is confined to the region vocabulary

- **WHEN** `Map` is called with any input
- **THEN** every returned code is a member of `enrich.RegionValues`, with no
  duplicates

### Requirement: The backfill annotates existing companies only, by slug

The system SHALL provide a run-once-and-exit worker
(`cmd/backfill-remote-regions`) that reads the dataset, maps each record's region
string via `remoteregion.Map`, resolves the company by `normalize.Slug(name)`, and
updates the matched company's `remote_regions` column and
`company_info.remote_regions_raw`. The update SHALL target existing companies
only: a record whose slug matches no company row SHALL be counted as unmatched and
SHALL NOT create a company (no reference row). The write SHALL NOT modify the
company's `name`, `job_count`, `collections`, `is_reference`, or any job-derived
facet array. The worker SHALL be idempotent — re-running the same dataset rewrites
the same values — and SHALL report matched, unmatched, mapped, and unmapped counts.

#### Scenario: Existing company is annotated

- **WHEN** the worker processes a record whose normalized name matches a company
  row, with a region string that maps to `{eu}`
- **THEN** that company's `remote_regions` is set to `{eu}` and its
  `company_info.remote_regions_raw` holds the source string, and no other company
  column changes

#### Scenario: Unmatched record creates no company

- **WHEN** the worker processes a record whose normalized name matches no company
  row
- **THEN** no company row is inserted and the record is tallied as unmatched

#### Scenario: A record whose region does not map is annotated with an empty set

- **WHEN** the worker processes a matched company whose region string maps to an
  empty slice
- **THEN** that company's `remote_regions` is set to `{}` and its
  `company_info.remote_regions_raw` still records the unmapped source string

#### Scenario: Re-running is idempotent

- **WHEN** the worker runs twice over the same dataset
- **THEN** the second run writes the same `remote_regions` values as the first
