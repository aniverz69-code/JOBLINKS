## 1. Backend — analysed-jobs endpoint

- [x] 1.1 Add `ListUserJobAnalyses(user_id) :many` in `internal/db/queries/user_job_analysis.sql` joining `user_job_analysis`→`jobs` (slug, title, company, closed_at, content_hash, analysis, model, cv_uploaded_at, job_content_hash, created_at), `ORDER BY created_at DESC`; run `make sqlc`.
- [x] 1.2 Add the `myAnalysisItem` wire shape `{slug,title,company,closed,overall_score,verdict,analysed_at,stale}` and `ListMyAnalyses` handler: parse each blob for overall/verdict, compute `stale` via `stampsFresh` (live CV time + `jobFit.ModelID()`), return `{data, meta:{quota}}` via `fitQuotaFor`. RED test first (fake store) for shape + stale + closed + quota-in-meta.

## 2. Backend — rename to Tracking (+ aliases)

- [x] 2.1 Rename `internal/handler/me_jobs.go` → `me_tracking.go`; rename methods `ListMyJobs`→`ListTrackedJobs`, `MyPipeline`→`TrackingPipeline` (keep `ListViewedSlugs`, `SwipeDeck`); update all references.
- [x] 2.2 In `handler.go` register canonical `/me/tracking`, `/me/tracking/{viewed,pipeline,swipe,analyses}`, and keep `/me/jobs*` as aliases to the same handlers (commented back-compat block naming freehire-cli). Update handler unit/integration tests to the new names.

## 3. Frontend — Tracking section + AI fit tab

- [x] 3.1 Move `web/src/routes/my/jobs/*` → `web/src/routes/my/tracking/*`; update `+layout.svelte` label to "Tracking" and add the "AI fit" tab (`/my/tracking/analyses`).
- [x] 3.2 Add `reroute` in `web/src/hooks.ts` mapping `/my/jobs` and `/my/jobs/<rest>` → `/my/tracking/<rest>` (preserve subpath). RED unit test on the pure reroute helper first.
- [x] 3.3 `api.ts`: switch tracking paths to `/me/tracking/*`, add `myAnalyses()` → `/me/tracking/analyses`; add the `MyAnalysisItem` type (generated). Update all `resolve('/my/jobs...')` refs (TopBar, HeaderMenu, HomeView, CliView, swipe & recommendations loaders).
- [x] 3.4 Build the AI-fit tab page (`my/tracking/analyses/+page.server.ts` + `+page.svelte`): quota banner "N/limit used this month" + list rows (score chip, verdict, company, analysed date, stale badge, Closed badge, link to `/jobs/[slug]/fit`); empty state when none.

## 4. Docs + verification

- [x] 4.1 Update the api-docs endpoint list (`web/scripts/gen-api-docs.mjs` data) for `/me/tracking*` (+ note `/me/jobs` alias) and the new `/me/tracking/analyses`.
- [x] 4.2 Verify: `go build/vet/test ./...` (+ integration) green; web `svelte-check`, `vitest`, lint clean on changed files; visual-check the Tracking tabs + AI-fit list.
