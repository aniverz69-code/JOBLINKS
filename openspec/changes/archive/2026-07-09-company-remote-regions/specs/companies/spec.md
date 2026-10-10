## MODIFIED Requirements

### Requirement: Company list is served without joining jobs

The system SHALL expose `GET /api/v1/companies` returning companies read from the
`companies` table. Each company's job count SHALL be read from the denormalized
`companies.job_count` column (open jobs only), not computed at query time, so the
read path performs no join to the `jobs` table. The list SHALL be ordered by
`job_count` descending, then `name` ascending, so the most active companies
surface first.

The endpoint SHALL accept an optional `q` query parameter that filters companies
by a case-insensitive substring match on the company `name`. An absent or empty
`q` SHALL return the unfiltered list.

The endpoint SHALL additionally accept repeatable facet query parameters —
`collections`, `regions`, `countries`, `domains`, `company_type`, `company_size`,
and `remote_regions` — each filtering against the company's corresponding
denormalized array by **array overlap**: a company matches a facet when its array
shares at least one value with the requested values (OR within a facet), and a
company must match every provided facet (AND across facets). The `remote_regions`
facet filters against the curated `companies.remote_regions` column (see the
`company-remote-regions` capability), independent of the job-derived `regions`
facet. Facet filters SHALL compose with the `q` name search. An absent facet
parameter SHALL not constrain the list.

When any filter (`q` or a facet) is applied, the list `meta.total` SHALL report
the count of companies matching the full filter combination, so pagination over
the filtered results is correct.

#### Scenario: Listing companies most-active first

- **WHEN** a client requests `GET /api/v1/companies`
- **THEN** the response contains companies under `data` with list `meta`,
  ordered by `job_count` descending (ties broken by `name`), each carrying its
  denormalized `job_count`

#### Scenario: Searching companies by name

- **WHEN** a client requests `GET /api/v1/companies?q=acme`
- **THEN** the response contains only companies whose name matches `acme`
  case-insensitively, ordered by `job_count` descending, and `meta.total` is the
  count of matching companies

#### Scenario: Empty query returns the full list

- **WHEN** a client requests `GET /api/v1/companies?q=` (empty or absent)
- **THEN** the response is the unfiltered company list, identical to omitting the
  parameter

#### Scenario: Filtering by a single facet

- **WHEN** a client requests `GET /api/v1/companies?regions=europe`
- **THEN** the response contains only companies whose `regions` array contains
  `europe`, and `meta.total` is the count of such companies

#### Scenario: Multiple values within one facet are OR-ed

- **WHEN** a client requests `GET /api/v1/companies?regions=europe&regions=asia`
- **THEN** the response contains companies whose `regions` overlap
  `{europe, asia}` (in Europe **or** Asia)

#### Scenario: Different facets are AND-ed and compose with search

- **WHEN** a client requests
  `GET /api/v1/companies?collections=yc&company_type=startup&q=lab`
- **THEN** the response contains only companies that are in the `yc` collection
  **and** have `startup` among their `company_types` **and** whose name matches
  `lab`

#### Scenario: Filtering by remote-hiring regions

- **WHEN** a client requests `GET /api/v1/companies?remote_regions=eu`
- **THEN** the response contains only companies whose `remote_regions` array
  contains `eu`, and `meta.total` is the count of such companies

#### Scenario: The remote-regions facet is independent of the job-derived regions facet

- **WHEN** a company has `remote_regions` `{eu}` but its open jobs derive
  `regions` `{north_america}`
- **THEN** the company matches `remote_regions=eu` and does not match
  `regions=eu`

### Requirement: Company job counts are denormalized and periodically recomputed

The system SHALL store each company's count of open jobs (`closed_at IS NULL`) in
a denormalized `companies.job_count` column, and its derived facet arrays
(`regions`, `countries`, `domains`, `company_types`, `company_sizes`) in
denormalized columns. Both SHALL be maintained by the same periodic recompute (a
scheduled worker), not by a synchronous write on the job ingest/close paths, so
they are eventually consistent with the `jobs` table within the recompute
interval. A company with no open jobs SHALL have `job_count = 0` and empty facet
arrays. The recompute SHALL NOT read or write the curated `remote_regions` column
(owned by the `company-remote-regions` backfill), so a backfilled value survives
every recompute.

#### Scenario: Recompute reflects only open jobs

- **WHEN** the recompute runs and a company has 3 open jobs and 2 closed jobs
  (`closed_at` set)
- **THEN** that company's `job_count` is set to 3 and its facet arrays reflect
  only the 3 open jobs

#### Scenario: Recompute zeroes a company whose jobs all closed

- **WHEN** every job of a company has been closed since the last recompute and the
  recompute runs again
- **THEN** that company's `job_count` is set to 0 and its facet arrays are emptied

#### Scenario: Counts are eventually consistent

- **WHEN** a new job is ingested for a company between recompute runs
- **THEN** the company's `job_count` and facet arrays do not change until the next
  recompute, which then includes the new job

#### Scenario: Recompute rewrites nothing when already current

- **WHEN** the recompute runs and a company's `job_count` and every facet array
  already equal the freshly computed values
- **THEN** that company's row is not rewritten (the recompute reports it as
  unchanged)

#### Scenario: Recompute leaves curated remote_regions untouched

- **WHEN** a company has a backfilled `remote_regions` value and the facet
  recompute runs
- **THEN** the company's `remote_regions` column is unchanged, regardless of its
  open jobs' derived `regions`
