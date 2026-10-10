<!-- Backend tasks use Go tests (unit; DB-touching handler/queue tests under the
     `integration` build tag). web/ has no unit-test runner: those tasks are
     verified with `svelte-check` + lint and a manual/headless pass, standing in
     for the RED→GREEN step. Each task runs the full RED→GREEN→simplify→review
     micro-cycle before it is checked off. -->

## 1. Persistence: the dismissed interaction

- [x] 1.1 Add migration `ALTER TABLE user_jobs ADD COLUMN IF NOT EXISTS dismissed_at TIMESTAMPTZ` (new migration file, mirroring the `saved_at` add)
- [x] 1.2 Add sqlc queries `DismissJob` (idempotent upsert `dismissed_at = now()`), `UndismissJob` (set `dismissed_at = NULL`), and `ExcludedJobIDs` (job ids where `dismissed_at IS NOT NULL OR saved_at IS NOT NULL`, capped/ordered most-recent) in `queries/user_jobs.sql`; run `make sqlc`
- [x] 1.3 Extend the `jobtracking` service with `Dismiss`/`Undismiss` use cases delegating to the new queries, mirroring save/unsave

## 2. Dismiss endpoints

- [x] 2.1 Add `DismissJob` handler `POST /jobs/:slug/dismiss` (resolve slug→id, idempotent, returns updated interaction) behind `RequireAuthOrKey`; wire the route
- [x] 2.2 Add `UndismissJob` handler `DELETE /jobs/:slug/dismiss` (no-op when not dismissed) behind `RequireAuthOrKey`; wire the route
- [x] 2.3 Integration tests (`integration` build tag): dismiss sets `dismissed_at`, is idempotent, works via API key, 401 unauth, 404 on missing job; undismiss clears and is a no-op when absent

## 3. Search: filterable id + exclusion helper

- [x] 3.1 Add `id` to `FilterableAttributes` in `internal/search/client.go`; update/settle the settings test
- [x] 3.2 Add `NotIn(attr string, ids []int64)` helper to `internal/search/filter.go` emitting `id NOT IN [...]` (escaped, composes through `Filter`); unit-test it

## 4. Swipe deck endpoint

- [x] 4.1 Add `GET /me/jobs/swipe` handler behind `RequireAuth`: reuse the search param parsing, add the `NotIn("id", ExcludedJobIDs(user))` filter group, return the standard list envelope of `jobview` items with `limit`/`offset`
- [x] 4.2 Wire the route; integration test: deck excludes the caller's saved+dismissed jobs, honors filters/`q`, pages via offset without repeats, 401 unauth

## 5. Frontend: API client + route shell

- [x] 5.1 Add `swipeDeck(params, limit, offset)`, `dismissJob(slug)`, `undismissJob(slug)` to `web/src/lib/api.ts`, mirroring `saveJob`/`unsaveJob`
- [x] 5.2 Create the `/jobs/swipe` route (`+page.svelte` + `+page.server.ts` noindex) reading the same URL params as `/jobs`; render the auth-gate prompt for signed-out users
- [x] 5.3 Add the swipe-mode entry button to `JobsView.svelte` toolbar → `goto('/jobs/swipe?' + filtersToParams(applied))`

## 6. Frontend: SwipeDeck component

- [x] 6.1 Create `SwipeDeck.svelte`: fetch the first batch via `swipeDeck`, render the active card (rich layout: logo, title, company+location, salary, facet chips, description excerpt) plus a peeking next card
- [x] 6.2 Desktop input: ✗/♥ buttons and `←`/`→` keys → dismiss/save + advance; `↑`/click opens detail; `U`/button triggers undo
- [x] 6.3 Mobile input: touch drag with commit threshold + exit animation — right = save, left = dismiss
- [x] 6.4 Save/dismiss call `saveJob`/`dismissJob` and advance; undo calls `unsaveJob`/`undismissJob` and restores the last card (single-step)
- [x] 6.5 Prefetch the next `offset` batch below a remaining-cards threshold; render the exhausted-deck empty state with a link back to `/jobs`

## 7. Verification

- [x] 7.1 `go build ./... && go vet ./... && go test ./...` (+ `-tags=integration ./internal/...` for the new DB tests) green
- [x] 7.2 `svelte-check` and lint clean for the touched web files
- [x] 7.3 Manual/headless pass across the spec scenarios: launch from `/jobs` with filters, save/dismiss/open/undo, exclusion of judged jobs, prefetch, empty state, mobile swipe + desktop keys, anon sign-in gate
