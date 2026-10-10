## Context

The `jobs` table stores a raw, source-shaped record: `title`, `company`,
`location`, `remote` (bool), `description` (free text), `posted_at`. The criteria
a job seeker actually filters on live inside `description` as prose. Established
job boards expose them as discrete, structured facets.

This change defines the **target schema** an AI layer will later populate. It is
the contract that phases 2 (extraction) and 3 (filtering/search) depend on, so it
must pin down not just field names but the **allowed values** of each enum field —
because the downstream filter layer (planned: Meilisearch) facets on exact value
matches, and inconsistent values (`senior` vs `Senior` vs `sr`) fragment a facet
into broken buckets.

Relevant constraints: Go + Fiber + sqlc (no ORM); migrations apply via Postgres
initdb only (no versioned runner yet); response shapes are `{"data":…}` /
`{"data":…,"meta":…}`.

## Goals / Non-Goals

**Goals:**

- Define one structured, additive representation of a job's enriched fields and
  where it is stored.
- Define every field's name, type, cardinality, and — for enum fields — the
  exact controlled vocabulary, in a single typed Go contract.
- Track enrichment provenance so phase 2 can find un-enriched rows and re-run on
  a schema bump.
- Shape the payload so it can become the Meilisearch document unchanged.

**Non-Goals:**

- The AI extraction itself (phase 2) — no LLM calls, prompts, or parsing here.
- Any filtering/search API, facets, or Meilisearch index (phase 3).
- Per-field confidence scoring or source attribution beyond a row-level stamp.
- Promoting company descriptors to the `companies` entity (kept as a seam).
- Backfilling existing rows — they default to an empty payload.

## Decisions

### D1. Storage: a single `jobs.enrichment JSONB`, not typed columns

The enriched fields live in one `enrichment JSONB NOT NULL DEFAULT '{}'` column
on `jobs`. Raw source columns are untouched; enrichment is purely additive.

- **Why over typed columns (option A):** filtering will be served by Meilisearch,
  not Postgres, so the main advantage of typed columns (btree/GIN filtering in
  Postgres) does not apply. Postgres becomes the source of truth; the filter
  index is a derived, rebuildable projection. JSONB lets the field set evolve
  during the discovery phase without a migration per field, and the blob maps 1:1
  to a Meilisearch document.
- **Why not pure-untyped JSONB:** the typing discipline is not dropped — it moves
  from the database into Go (D2). The DB stores the blob; the Go contract and the
  enrichment layer guarantee its shape.
- **Alternative (hybrid, option C)** — hot fields as columns + tail in JSONB —
  was rejected for now: with Meilisearch doing all filtering, no enriched field
  needs to be a Postgres column. Promoting a field to a column later is a
  localized migration if a Postgres-side need appears.

### D2. The schema's source of truth is a typed Go contract + vocabularies

A new package `internal/enrich` holds one `Enrichment` struct (JSON-tagged,
matching the blob) and the controlled-vocabulary constants for every enum field.
sqlc reads `enrichment` as `json.RawMessage`; conversion to/from `Enrichment`
happens in Go. The vocabularies are exported constants (e.g. `SeniorityValues`)
so phase 2's prompt and phase 3's facet config both reference the same lists.

- **Why:** keeps a single, code-checkable definition of valid values; prevents
  facet fragmentation at the source; gives the rest of the codebase typed access
  despite JSONB storage.
- **Alternative:** DB-level `CHECK`/enum types — rejected because values live in
  a JSONB blob and are expected to evolve; enforcing them in Go is the natural
  seam and avoids migrations on every vocabulary change.

### D3. Field catalog

All fields are **optional**: omitted (or null) when the source does not state
them. Enum fields accept only their listed values. JSON keys are `snake_case` to
match existing `jobs` JSON tags.

| JSON key | Type | Card. | Allowed values / format |
|---|---|---|---|
| **Work arrangement** | | | |
| `work_mode` | enum | one | `remote` · `hybrid` · `onsite` |
| `employment_type` | enum | one | `full_time` · `part_time` · `contract` · `internship` |
| `relocation` | enum | one | `not_supported` · `supported` · `required` |
| `visa_sponsorship` | bool | one | `true` / `false` |
| **Location / eligibility** | | | |
| `countries` | enum[] | many | ISO 3166-1 alpha-2 (e.g. `US`, `DE`, `UA`) — eligible work countries |
| `cities` | string[] | many | free text (not faceted) |
| `timezone_note` | string | one | free text, e.g. "UTC±2 overlap" (not faceted) |
| **Compensation** | | | |
| `salary_min` | int | one | integer in `salary_currency` units |
| `salary_max` | int | one | integer in `salary_currency` units |
| `salary_currency` | enum | one | ISO 4217 (e.g. `USD`, `EUR`) |
| `salary_period` | enum | one | `year` · `month` · `day` · `hour` |
| **Requirements** | | | |
| `seniority` | enum | one | `intern` · `junior` · `middle` · `senior` · `lead` · `principal` · `c_level` |
| `experience_years_min` | int | one | non-negative integer |
| `english_level` | enum | one | CEFR: `none` · `a1` · `a2` · `b1` · `b2` · `c1` · `c2` · `native` |
| `education_level` | enum | one | `none` · `bachelor` · `master` · `phd` |
| `skills` | string[] | many | normalized lowercase canonical tokens (e.g. `go`, `postgresql`, `kubernetes`) |
| **Classification** | | | |
| `category` | enum | one | `backend` · `frontend` · `fullstack` · `mobile` · `devops` · `data_engineering` · `data_science` · `ml_ai` · `qa` · `security` · `design` · `product` · `project_management` · `management` · `marketing` · `sales` · `support` · `other` |
| `domains` | enum[] | many | `fintech` · `gambling` · `ecommerce` · `crypto` · `healthcare` · `saas` · `gamedev` · `edtech` · `adtech` · `govtech` · `media` · `travel` · `logistics` · `other` |
| `posting_language` | enum | one | ISO 639-1 (e.g. `en`, `uk`, `ru`) |
| **Company descriptors** (job-time observation; seam to `companies`) | | | |
| `company_type` | enum | one | `product` · `startup` · `outsource` · `outstaff` · `agency` · `inhouse` · `government` |
| `company_size` | enum | one | `1-10` · `11-50` · `51-200` · `201-500` · `501-1000` · `1000+` |

Vocabulary provenance: `seniority`, `company_type`, and the work-format and
experience values follow common industry conventions; `english_level` uses the
CEFR scale (`a1`–`c2`); `countries`/`salary_currency`/`posting_language` use ISO
standards so they are unambiguous and machine-checkable. `skills` is intentionally open (no closed
vocabulary), with only a normalization rule (lowercase canonical token) defined
now; an alias/canonicalization map is a phase-2 concern.

### D4. Provenance as two row-level columns

Add `enriched_at TIMESTAMPTZ` (nullable) and `enrichment_version INT NOT NULL
DEFAULT 0` as columns (not in the blob). `enriched_at IS NULL` ⇒ never enriched;
`enrichment_version` lets phase 2 select rows below the current schema version to
re-run. Columns (not JSON) because phase 2 queries them directly to find work.

- **Alternative:** per-field confidence/source — deferred as overengineering for
  phase 1; the row-level stamp is enough to drive re-enrichment.

### D5. Company descriptors stay in the job blob for now

`company_type` and `company_size` describe the company but are captured per job
(the AI reads them from the job's description). They live in `enrichment`, not on
the `companies` table. This satisfies "filter by company criteria" as job facets
without building company-level enrichment infrastructure before there is a need.
Promoting them to `companies` is a noted future change.

### D6. Exposure on the read API

The jobs read responses (`GET /api/v1/jobs`, `/api/v1/jobs/:id`, and the jobs
nested under a company) include `enrichment` (object, `{}` when absent),
`enriched_at`, and `enrichment_version`. Additive only — no field is removed or
renamed, so no existing requirement changes.

### D7. Migration applies via initdb; no runner introduced

The new columns ship as a new `migrations/0003_job_enrichment.sql`, applied on a
fresh volume like the others. This change deliberately does not build the
versioned migration runner (still a noted seam) — it only adds columns a fresh
DB picks up and `sqlc` reads.

## Risks / Trade-offs

- **No DB-level value validation** → the Go `enrichment` write path
  (`UpsertJob`) and phase 2 are solely responsible for vocabulary correctness;
  the contract in `internal/enrich` is the guard. Invalid values would surface as
  fragmented facets downstream, so vocabulary validation belongs in the
  enrichment write path when phase 2 lands.
- **Existing volumes won't get the columns** → recreate with `down -v && make up`
  in dev; production needs the migration-runner seam before persistent rollout
  (unchanged by this change, just reaffirmed).
- **Vocabularies will need to grow** (new `category`/`domain` values) → expected;
  because they are Go constants over a JSONB blob, adding a value is a code
  change with no migration. Keep `other` as the safety valve.
- **Per-job company descriptors can disagree across a company's jobs** →
  acceptable while they are job facets; resolved if/when promoted to `companies`.
- **`json.RawMessage` round-tripping** → marshal/unmarshal through the typed
  `Enrichment` struct at the boundaries; never hand-build the JSON.

## Migration Plan

1. Add `migrations/0003_job_enrichment.sql` (`enrichment`, `enriched_at`,
   `enrichment_version`).
2. Update `internal/db/queries/jobs.sql` (select the new columns;
   `UpsertJob` accepts `enrichment`), run `make sqlc`, commit generated code.
3. Add `internal/enrich` (struct + vocabulary constants).
4. Surface the new fields in the jobs handler responses.
5. Dev: `docker compose down -v && make up` to re-init the volume.
6. Rollback: drop the migration file and the columns; the change is additive, so
   no data migration is required.

## Open Questions

- None blocking. Candidate fields deferred unless requested: `equity_offered`
  (startup-specific), `salary_is_gross`, `benefits[]`, `interview_steps`. They
  fit the JSONB blob additively if needed later.
