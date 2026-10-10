## Why

Comparing a job's stated requirements against a candidate is currently
re-derived from scratch every time it's needed: `internal/hardconstraint`
checks a handful of fixed categories (experience years, education, visa,
language, work mode) against job columns, and `internal/matchanalysis` Stage 1
separately re-extracts the posting's requirements via the LLM on every first
analysis of a (user, job, CV) triple. Neither produces a reusable, job-only,
open-ended view of what a posting actually asks for — `internal/jobfacts`'
regex/dict matchers cover only those same fixed categories and miss anything
stated in prose (a specific tool, a domain background, a soft skill). Since
`cmd/enrich` already runs an LLM pass over every open job's description, it is
the natural place to extract this list once per job instead of paying for it
again on every downstream comparison.

## What Changes

- Add a new `Requirements []Requirement` field (`Requirement{Text, Priority}`,
  `Priority` ∈ `required`/`preferred`) to the `internal/enrich` `Enrichment`
  contract, extracted by the LLM from the job description alongside the
  existing enrichment fields.
- The list is job-only: it reflects what the posting states, with no
  comparison against any candidate — distinct from, and not wired to,
  `matchanalysis.Requirement`, which additionally classifies a requirement
  against one candidate's CV.
- Bound and sanitize the new field the same way other free-text enrichment
  fields are (list-length cap, per-item text-length cap, priority coerced
  into the controlled vocabulary rather than rejected).
- The field is served like every other enrichment field: it flows through
  `internal/jobview` into the public jobs read API and the generated TS
  contracts, with no suppression.
- **Out of scope**: no change to `internal/hardconstraint` or
  `internal/matchanalysis` — nothing yet reads or acts on the new field, it is
  extract-and-store only. No new database column or migration (stored in the
  existing `jobs.enrichment` JSONB). No `enrich.Version` bump (so existing
  already-enriched jobs do not retroactively gain the field as part of this
  change).

## Capabilities

### New Capabilities

(none — this extends two existing capabilities)

### Modified Capabilities

- `job-enrichment`: the typed `Enrichment` contract gains a `requirements`
  field, and the jobs read API's `enrichment` payload includes it like any
  other served field.
- `ai-enrichment`: the LLM provider additionally extracts the posting's
  stated requirements as a freeform `{text, priority}` list, bounded and
  sanitized the same way other free-text/enum fields are.

## Impact

- `internal/enrich`: `enrichment.go` (contract, bounds, sanitize),
  `schema.go` (request schema — no schema-level enum on the nested
  `priority`, prompt + sanitize only), `langchain.go` (prompt instruction),
  `enrichment_unmarshal.go` (tolerant-decode shadow struct).
- `internal/jobview`: no code change, but the field now passes through
  `FromDomain` into the public `Job` shape — pinned by a new test.
- `cmd/gen-contracts`: regenerated `web/src/lib/generated/contracts.ts` gains
  a `Requirement` interface and a `requirements` member on `Enrichment`.
- No database migration; no change to `internal/hardconstraint`,
  `internal/jobfacts`, or `internal/matchanalysis`.
