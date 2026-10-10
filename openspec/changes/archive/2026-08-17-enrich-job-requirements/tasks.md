## 1. Contract, bounds, and sanitize

- [x] 1.1 Add `Requirement{Text, Priority string}` and
      `Requirements []Requirement` to `internal/enrich/enrichment.go`'s
      `Enrichment` struct, under a new "Stated requirements (job-only, no
      CV)" grouping comment, with a doc comment cross-referencing
      `matchanalysis.Requirement` as a deliberately unrelated sibling type.
- [x] 1.2 Add `maxRequirements`/`maxRequirementTextRunes` consts.
- [x] 1.3 Implement `boundRequirements` (mirrors `boundCities`: cap list
      length, per-item text clip via `llm.TrimTruncateRunes`, drop
      empty-after-trim entries) and a local `coerceRequirementPriority`
      (case/whitespace-insensitive match on `"required"`, else
      `"preferred"`) in `Sanitize()`. Add a code comment on `Requirements`
      warning against adding an `llmschema.Enum` override for `priority`
      (see design.md - Decisions).
- [x] 1.4 Extend `TestRoundTripFidelity` in `enrichment_test.go` with a
      `Requirements` value in the `original` fixture.
- [x] 1.5 Add `TestSanitizeBoundsRequirements` in `enrichment_test.go`:
      list-length cap, per-item text clip, blank-after-trim entries
      dropped, priority coercion (including case/whitespace-insensitive
      `"Required "` → `"required"` and unrecognized `"nice-to-have"` →
      `"preferred"`).

## 2. Tolerant decode

- [x] 2.1 Add `Requirements []Requirement` to the `enrichmentJSON` shadow
      struct in `internal/enrich/enrichment_unmarshal.go` and copy it
      through in `UnmarshalJSON`.
- [x] 2.2 Add a `requirementListOrWrap` type mirroring `sliceOrWrap`'s
      pattern (decode `[]Requirement`, or fall back to decoding one bare
      `Requirement` and wrapping it into a one-element slice).
- [x] 2.3 Add `TestUnmarshalCoercesBareObjectRequirementToArray` in
      `enrichment_parse_test.go`, paralleling the existing scalar→array
      coercion test.

## 3. Request schema

- [x] 3.1 Confirm (with a test, not just reading) that plain reflection in
      `internal/enrich/schema.go`'s `requestSchema` produces a `requirements`
      property with a strict-mode object item shape (`text`/`priority`
      required, non-null) with no code change needed there beyond what
      reflection already does.
- [x] 3.2 Add `"requirements"` to the field list in
      `TestRequestSchema_CarriesTheFieldsThePromptAsksFor` (`schema_test.go`).
- [x] 3.3 Add `TestRequestSchema_RequirementsHasNoEnum` asserting
      `items.properties.priority` on the `requirements` schema node carries
      no `"enum"` key.

## 4. Prompt

- [x] 4.1 Add an unconditional instruction block to `buildSystemPrompt` in
      `internal/enrich/langchain.go` (not inside the `askGeo` branch)
      describing the `requirements` field: array of `{text, priority}`,
      max 30 entries, explicitly framed as "no CV to compare against —
      base this only on the job posting."
- [x] 4.2 Add `"requirements"` to the field list in
      `TestSystemPromptKeepsServedAndHybridFields` (`langchain_test.go`).
- [x] 4.3 Add a test asserting the prompt contains the "no CV to compare
      against" framing.

## 5. Public API pass-through

- [x] 5.1 Add a test in `internal/jobview` asserting a job's `Requirements`
      (from a stubbed `Enrichment`) passes through `FromDomain` unchanged
      into the served `Job` (pins the "let it flow through" design decision;
      expected to pass with no production code change, since `jobview`
      already embeds `Enrichment` wholesale).

## 6. Generated contracts and verification

- [x] 6.1 Run `make gen-contracts` and commit the regenerated
      `web/src/lib/generated/contracts.ts` (new `Requirement` interface,
      `requirements` member on `Enrichment`).
- [x] 6.2 Run `gofmt -l .` (must be clean), `go vet ./...`, `go test ./...`.
- [x] 6.3 Run `go vet -tags=integration ./...`.
- [ ] 6.4 Manually sanity-check one real enrichment call against a
      configured LLM (`go run ./cmd/enrich` against a dev DB with `LLM_*`
      env set) and confirm `requirements` appears in `jobs.enrichment` for a
      freshly-enriched job and in the `GET /api/v1/jobs/:slug` response.
