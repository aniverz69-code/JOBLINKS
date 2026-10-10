## ADDED Requirements

### Requirement: Jobs store an additive enrichment payload

The system SHALL store a structured enrichment payload per job in a
`jobs.enrichment` JSONB column that defaults to an empty object. Enrichment SHALL
be additive: writing it SHALL NOT modify any raw source field (`title`,
`company`, `location`, `remote`, `description`, `posted_at`, `company_slug`).

#### Scenario: New job defaults to an empty payload

- **WHEN** a job is upserted without an enrichment payload
- **THEN** its `enrichment` reads back as an empty object (`{}`) and its raw
  fields are stored unchanged

#### Scenario: Enrichment is stored without altering raw fields

- **WHEN** a job is upserted with an enrichment payload
- **THEN** the payload is persisted under `enrichment` and the job's raw fields
  remain exactly as supplied

### Requirement: Enrichment fields follow a typed contract with controlled vocabularies

The system SHALL define the enrichment payload as a single typed Go contract in
`internal/enrich` whose fields and allowed values are the schema's source of
truth. Every field SHALL be optional and omitted when not determined. Enum
fields SHALL accept only their defined vocabulary values; `skills`, `cities`, and
`countries` SHALL be arrays; `skills` values SHALL be normalized lowercase
tokens. The contract SHALL provide validation of a payload against the
vocabularies.

#### Scenario: Payload round-trips through the typed contract

- **WHEN** an `Enrichment` value (e.g. `seniority=senior`, `work_mode=remote`,
  `skills=[go, postgresql]`) is marshalled to JSON, stored, read back, and
  unmarshalled
- **THEN** the resulting value equals the original

#### Scenario: Undetermined fields are omitted, not zero-filled

- **WHEN** an enrichment payload does not determine salary
- **THEN** the `salary_min`, `salary_max`, `salary_currency`, and
  `salary_period` keys are absent from the stored JSON rather than present with
  zero/empty values

#### Scenario: A value outside a vocabulary is reported invalid

- **WHEN** the contract validates a payload whose `seniority` is `"sr"` (not a
  defined value)
- **THEN** validation reports the payload as invalid, identifying the offending
  field

### Requirement: Enrichment provenance is tracked per job

The system SHALL track enrichment provenance with two job columns: `enriched_at`
(nullable timestamp) and `enrichment_version` (integer, default 0). A job that
has never been enriched SHALL have `enriched_at` null and `enrichment_version` 0,
so un-enriched rows are identifiable for later processing.

#### Scenario: Un-enriched job is identifiable

- **WHEN** a job has never been enriched
- **THEN** its `enriched_at` is null and its `enrichment_version` is 0

#### Scenario: Provenance reflects a completed enrichment

- **WHEN** a job is written with an enrichment payload produced at schema
  version N
- **THEN** its `enriched_at` is set and its `enrichment_version` equals N

### Requirement: Company descriptors are captured as job enrichment fields

The system SHALL capture company descriptors (`company_type`, `company_size`) as
fields of the job's enrichment payload, not as columns on the `companies` table.
Writing them SHALL NOT alter any `companies` row.

#### Scenario: Company descriptors live in the job payload

- **WHEN** a job is upserted with enrichment including `company_type=product`
- **THEN** the value is stored in that job's `enrichment` and no `companies` row
  is created or modified by it

### Requirement: The jobs read API exposes enrichment and provenance

The system SHALL include `enrichment`, `enriched_at`, and `enrichment_version` in
the job objects returned by the jobs read endpoints (`GET /api/v1/jobs`,
`GET /api/v1/jobs/:id`, and jobs nested under a company). The addition SHALL be
backward compatible: no existing field is removed or renamed.

#### Scenario: Job detail includes enrichment and provenance

- **WHEN** a client requests `GET /api/v1/jobs/:id` for an existing job
- **THEN** the returned object under `data` includes `enrichment`,
  `enriched_at`, and `enrichment_version` alongside the existing fields

#### Scenario: Empty enrichment serializes as an object

- **WHEN** a job that has not been enriched is returned by a read endpoint
- **THEN** its `enrichment` is serialized as an empty object (`{}`), not null
