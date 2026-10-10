## MODIFIED Requirements

### Requirement: Company list is served without joining jobs

The system SHALL expose `GET /api/v1/companies` returning companies read from the
`companies` table. Job counts, when included, SHALL be computed at query time over
**open jobs only** (`closed_at IS NULL`); no denormalized counter is required.

The endpoint SHALL accept an optional `q` query parameter that filters companies
by a case-insensitive substring match on the company `name`. An absent or empty
`q` SHALL return the unfiltered list (today's behavior). When `q` is non-empty,
the list `meta.total` SHALL report the count of companies matching `q`, so
pagination over the filtered results is correct.

#### Scenario: Listing companies

- **WHEN** a client requests `GET /api/v1/companies`
- **THEN** the response contains companies under `data` with list `meta`,
  following the existing list response shape

#### Scenario: Searching companies by name

- **WHEN** a client requests `GET /api/v1/companies?q=acme`
- **THEN** the response contains only companies whose name matches `acme`
  case-insensitively, and `meta.total` is the count of matching companies

#### Scenario: Empty query returns the full list

- **WHEN** a client requests `GET /api/v1/companies?q=` (empty or absent)
- **THEN** the response is the unfiltered company list, identical to omitting the
  parameter

#### Scenario: Closed jobs leave the job count

- **WHEN** a company's job is closed
- **THEN** the company's `job_count` no longer counts it

### Requirement: Company detail returns the company with its jobs

The system SHALL expose `GET /api/v1/companies/:slug` returning the company and
its **open** jobs (`closed_at IS NULL`). The company SHALL be read from
`companies` and its jobs from a single-table filter on `jobs.company_slug` —
without a SQL join between the two tables.

#### Scenario: Existing company

- **WHEN** a client requests `GET /api/v1/companies/:slug` for an existing slug
- **THEN** the response contains the company and its open jobs ordered like the
  main jobs listing

#### Scenario: Unknown company

- **WHEN** a client requests `GET /api/v1/companies/:slug` for a slug with no
  company row
- **THEN** the system responds with HTTP 404

#### Scenario: Closed job leaves the company page

- **WHEN** a company's job is closed
- **THEN** the company detail no longer lists it
