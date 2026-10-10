## Why

Browsing the catalogue one row at a time is slow, especially on mobile: a user
applies filters on `/jobs`, then scrolls a long list, opening each posting to
decide. A Tinder-style swipe deck turns triage into single-gesture decisions —
save the ones worth a look, discard the rest — and clears already-judged jobs
out of the way so the next session starts fresh.

## What Changes

- Add a **swipe triage mode**: a full-screen card deck reachable from a new
  button in the `/jobs` toolbar. The deck honors the page's current filters and
  free-text query (carried over via the URL).
- Two decisions per card: **swipe right = save** (reuses the existing save
  interaction), **swipe left = dismiss** (a new persisted mark). On desktop the
  same decisions are driven by ✗/♥ buttons and the `←`/`→` arrow keys; `↑`/click
  opens the job in detail, and a one-step **undo** reverts the last action.
- The deck excludes the caller's already **saved** and **dismissed** jobs, and
  is filled by a new authenticated endpoint that runs the same Meilisearch query
  as `/jobs/search` with a server-side `id NOT IN (excluded)` filter.
- Add a new **dismissed** interaction to per-user job tracking: `dismissed_at`
  on `user_jobs`, written by `POST /jobs/:slug/dismiss` and cleared by
  `DELETE /jobs/:slug/dismiss` (matching the existing `save`/`unsave` routes).
  Dismissal only affects the swipe deck — the job still appears in the normal
  `/jobs` list and search.
- Make the Meilisearch `id` attribute **filterable** so the deck can exclude
  jobs by id. This requires a one-time reindex on deploy (reindex first, or
  `/jobs` 500s on the new attribute until the index is rebuilt).

## Capabilities

### New Capabilities

- `job-swipe`: The swipe triage mode — its entry point and route, the
  authenticated deck endpoint (filter/query parity with search, exclusion of
  saved + dismissed jobs, batched with prefetch), the card presentation, the
  save/dismiss/open/undo actions, gesture vs. button+keyboard input across
  mobile and desktop, and the empty/auth states.

### Modified Capabilities

- `user-job-tracking`: Adds a **dismissed** interaction (`dismissed_at`) with
  idempotent set/clear endpoints, alongside the existing view/apply/save marks.

## Impact

- **Backend (`internal/`)**: one migration (`user_jobs.dismissed_at`), new sqlc
  queries (`DismissJob`/`UndismissJob`/`ExcludedJobIDs`), two dismiss handlers +
  one deck handler in `handler/user_jobs.go`, a `NotIn` filter helper in
  `search/filter.go`, and adding `id` to `search` filterable attributes.
- **Search index**: a one-time reindex is required on deploy for the new
  filterable `id` attribute.
- **Frontend (`web/`)**: a new `/jobs/swipe` route, a `SwipeDeck.svelte`
  component, a toolbar entry button in `JobsView.svelte`, and three new
  `api.ts` client functions (`swipeDeck`, `dismissJob`, `undismissJob`).
- No changes to the public catalogue wire shape; reuses `jobview` and the
  existing save interaction.
