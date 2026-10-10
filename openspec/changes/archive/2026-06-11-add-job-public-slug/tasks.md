## 1. Slug builder

- [x] 1.1 Add a deterministic `JobSlug(title, company, source, externalID)` to
  `internal/normalize` that joins the non-empty `Slug(title)`/`Slug(company)`
  segments and an 8-char lowercased base32 `sha256(source + 0x00 + externalID)`
  shortcode with `-`. Test-first: same inputs → same slug; stable when only the
  description would change (it is not an input); empty title and/or company still
  yield a non-empty slug; distinct `(source, external_id)` → distinct shortcode.

## 2. Schema and DB access

- [x] 2.1 Add `migrations/0007_job_public_slug.sql` adding
  `jobs.public_slug TEXT NOT NULL UNIQUE`.
- [x] 2.2 In `queries/jobs.sql`, add `public_slug` as an arg to `UpsertJob`
  (written on INSERT; deliberately NOT in `DO UPDATE` — slug is mint-once) and
  add a `GetJobBySlug` query selecting by `public_slug`.
- [x] 2.3 Run `make sqlc` and commit the regenerated `internal/db`. Verify
  `go build ./...` and `go vet ./...`.

## 3. Read serialization

- [x] 3.1 Introduce a handler `jobResponse` DTO that carries `public_slug`,
  omits the numeric `id`, and preserves enrichment passthrough (`enrichment` raw
  object / `{}` not null, `enriched_at`, `enrichment_version`). Move the
  guarantees in `serialization_test.go` onto the DTO (test-first).
- [x] 3.2 Map `db.Job` → `jobResponse` in `ListJobs` and the single-job read.

## 4. Slug-based routing

- [x] 4.1 Change `GetJob` to fetch via `GetJobBySlug` using the `:slug` param
  (unknown slug → 404 via the central handler). Verified via db-integration
  (GetJobBySlug ErrNoRows) + errors_test (ErrNoRows→404).
- [x] 4.2 Change `interactionParams` to resolve the `:slug` param to the internal
  job id via `GetJobBySlug` (unknown slug → 404), keeping `user_jobs` writes
  keyed by the `BIGINT` id. Removed the now-obsolete non-numeric-id tests; kept
  the auth-gate tests on slug routes.
- [x] 4.3 Update route registration in `handler.go`: `/jobs/:slug`,
  `/jobs/:slug/view`, `/jobs/:slug/apply`.

## 5. SPA

- [x] 5.1 Update the Svelte SPA under `web/` to address jobs by `public_slug`
  (job links, detail route, view/apply calls) instead of the numeric id.

## 6. Verification

- [x] 4.4 (from code review) `GetCompany` also returns jobs — route them through
  the DTO so the id does not leak there either. Made type-safe via
  `companyDetailResponse{Jobs []jobResponse}`; regression test
  `TestCompanyDetailHidesJobID`.
- [x] 6.1 `go build ./...`, `go vet ./...`, `go test ./...`, and the DB
  integration tests; recreate the volume and smoke-check `GET /api/v1/jobs/:slug`
  (200, no id, slug present), unknown slug → 404, list + company jobs → no id.
