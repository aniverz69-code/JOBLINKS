## 1. Schema migration

- [x] 1.1 Add `migrations/0003_job_enrichment.sql` adding to `jobs`:
  `enrichment JSONB NOT NULL DEFAULT '{}'`, `enriched_at TIMESTAMPTZ` (nullable),
  `enrichment_version INT NOT NULL DEFAULT 0`.
- [x] 1.2 Recreate the dev volume (`docker compose down -v && make up`) and
  confirm the three columns exist via `make psql` (`\d jobs`). _(verified on
  OrbStack: `enrichment jsonb NOT NULL DEFAULT '{}'`, `enriched_at timestamptz`
  nullable, `enrichment_version integer NOT NULL DEFAULT 0`.)_

## 2. Enrichment contract (`internal/enrich`)

- [x] 2.1 Create `internal/enrich/enrichment.go` with the `Enrichment` struct:
  one JSON-tagged, `omitempty`/pointer field per the design catalog
  (work arrangement, location/eligibility, compensation, requirements,
  classification, company descriptors). All fields optional.
- [x] 2.2 Add controlled-vocabulary constants and exported value sets for every
  enum field (`SeniorityValues`, `WorkModeValues`, `EmploymentTypeValues`,
  `RelocationValues`, `EnglishLevelValues`, `EducationLevelValues`,
  `SalaryPeriodValues`, `CategoryValues`, `DomainValues`, `CompanyTypeValues`,
  `CompanySizeValues`).
- [x] 2.3 Add `Validate()` that checks each enum field against its vocabulary and
  reports the first offending field; non-enum fields are unconstrained.
- [x] 2.4 Add `internal/enrich/enrichment_test.go`: JSON round-trip fidelity,
  `omitempty` on undetermined fields, and `Validate` accept/reject cases.

## 3. DB access (sqlc)

- [x] 3.1 Edit `internal/db/queries/jobs.sql`: select `enrichment`,
  `enriched_at`, `enrichment_version` in `GetJob`, `ListJobs`,
  `ListJobsByCompany`; add `enrichment` (and provenance) to `UpsertJob`.
  _(`SELECT *`/`RETURNING *` pick up the new columns automatically; only
  `UpsertJob`'s INSERT/ON CONFLICT were edited.)_
- [x] 3.2 Confirm `enrichment` is generated as `json.RawMessage` (sqlc.yaml
  override added: `jobs.enrichment` → `encoding/json.RawMessage`); ran sqlc
  v1.31.1 and committed the regenerated `internal/db/*.go`.

## 4. API exposure

- [x] 4.1 Include `enrichment`, `enriched_at`, `enrichment_version` in the job
  objects returned by `GET /api/v1/jobs`, `GET /api/v1/jobs/:id`, and the jobs
  nested under `GET /api/v1/companies/:slug`. _(Handlers return `db.Job`
  directly; the additive generated fields surface with no handler edits.)_
- [x] 4.2 Ensure an absent payload serializes as `{}` (not `null`) in responses.
  _(`json.RawMessage` override: a `{}` blob marshals through verbatim; verified
  via a serialization check — output `"enrichment":{}`, `"enriched_at":null`,
  `"enrichment_version":0`.)_

## 5. Verification

- [x] 5.1 `go build ./... && go vet ./...` clean.
- [x] 5.2 `go test ./internal/enrich/...` passes.
- [x] 5.3 Insert one enriched job via `make psql`, then `curl` the list, detail,
  and company-nested endpoints to confirm enrichment + provenance are exposed and
  that an un-enriched job shows `enrichment: {}`, `enriched_at: null`,
  `enrichment_version: 0`. _(verified on OrbStack: list, `/jobs/:id`, and
  `/companies/acme` all expose enrichment + provenance; un-enriched job returns
  `enrichment: {}`, `enriched_at: null`, `enrichment_version: 0`; enriched job
  returns the raw JSON payload, not base64.)_
