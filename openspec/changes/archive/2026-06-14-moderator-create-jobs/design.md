## Context

All jobs today arrive through automated sources (ATS adapters, Telegram extraction,
link-following). The single write path is `UpsertJob` + `EnqueueJobEnrichment` in one
transaction, adapted to `pipeline.Store` and duplicated across `cmd/ingest`,
`cmd/tg-extract`, and `linksource`. Authentication is a stateless JWT (`sub` = user id)
in an httpOnly cookie, or a hashed API key via `RequireAuthOrKey`; the user id lands in
`c.Locals` and there is no notion of a role. A recent change extracted per-user job
tracking into `internal/jobtracking` as a `Service` + `Repository` pair — the pattern
this change mirrors.

## Goals / Non-Goals

**Goals:**

- Let a `moderator` create (`POST /api/v1/jobs`) and edit (`PATCH /api/v1/jobs/:slug`)
  hand-curated vacancies, authenticated by API key (CLI) or cookie.
- Add a persisted user `role` and a `RequireRole` authorization layer.
- Record `created_by` / `updated_by` authorship audit on jobs.
- Reuse the canonical write path (enqueue enrichment on create) without disturbing the
  working ingest path.

**Non-Goals:**

- Re-running enrichment after an edit (`PATCH` does not re-enqueue).
- Changing a manual job's URL / identity (re-key).
- Self-service role management or an `admin` capability (column is forward-compatible).
- Synchronous Meilisearch indexing (manual jobs reach search via `make reindex`, as ingest).
- The `freehire-cli` command itself (separate repo, tracked as follow-up).

## Decisions

### Role lives in the DB, checked by middleware after auth

`role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user','moderator','admin'))` on `users`.
`RequireRole(q, "moderator")` runs after `RequireAuthOrKey`, reads `userID` from `c.Locals`,
loads the role via `GetUserByID`, and 403s on mismatch.

- **Why not put the role in the JWT?** The JWT deliberately carries only `sub`; a DB-backed
  role applies immediately on role change and survives the planned swap to opaque sessions.
  Cost is one indexed lookup per guarded request — negligible for a low-traffic moderator
  endpoint.
- **Alternative considered**: a boolean `is_moderator` — rejected as non-extensible; the
  enum already accommodates `admin`.

### New `internal/moderation` service, ingest `UpsertJob` left untouched

A `Service` + `Repository` pair mirroring `internal/jobtracking`. The `Repository` adapts
`*db.Queries` + the pool, running `UpsertManualJob` + `EnqueueJobEnrichment` (create) or
`UpdateManualJob` (edit) in one transaction. The handler in `internal/handler/jobs_moderation.go`
is thin: parse body → call service → `{ "data": job }`.

- **Why a dedicated `UpsertManualJob` rather than adding `created_by` to the shared
  `UpsertJob`?** Keeps the moderator concern out of the working ingest path (surgical, zero
  regression risk). The cost is near-duplicate SQL; the benefit is isolation. `UpsertJob`
  has no actor, so threading a nullable `created_by` through it would pollute every ingest
  call site.
- **Alternative considered**: refactor all four write sites into one shared store
  (Approach 2). Rejected for blast radius and risk to a working pipeline; the project is
  MVP-fluid but ingest is the most load-bearing path.

### Shared derivation helper

The geo/skills/slug derivation currently inline in `pipeline.normalizeJob` (calls to
`location.Parse`, `skilltag.Parse`, `normalize.JobSlug`/`Slug`, work-mode precedence) is
factored into a small helper so the manual write path derives identically. The manual path
differs only in source identity (`source='manual'`, `external_id=url`, no board namespacing).

### Edit is load-merge-derive-write, scoped to `source='manual'`

`PATCH` is partial at the API, but the partial merge happens in the service, not in SQL:
the service loads the manual job, overlays the supplied (nil-means-unchanged) fields, and
**re-derives the deterministic facets** (geography, skills, company slug) from the merged
content via the shared helper — so a location/description/company edit never leaves a stale
facet (AI enrichment is the only facet not refreshed). The write is then a full-field
`UPDATE ... WHERE public_slug = $1 AND source = 'manual'` returning the row; a slug that
resolves to a non-manual (or missing) job returns no row → `404`. That `source = 'manual'`
guard is the security invariant: the moderator write path can never rewrite an ATS/Telegram
vacancy. The source identity (`url`/`external_id`/`public_slug`) is never recomputed, so the
public slug stays stable across edits. The load-then-write is not wrapped in one transaction
(low-frequency, single-actor edits; last-write-wins is acceptable).

- **Alternative considered**: a `COALESCE($field, column)` partial `UPDATE` (no read). Rejected
  because it cannot recompute the derived facets from the merged values, leaving geography/
  skills inconsistent after an edit.

### Audit fields stay off the wire

`created_by` / `updated_by` are not added to `jobview` (the shared public shape), exactly as
`user_id` is omitted from `user_jobs` responses. Read paths are unchanged.

## Risks / Trade-offs

- **Near-duplicate write SQL across four sites** → Accepted deliberately to protect the ingest
  path; the shared derivation helper removes the more error-prone duplication (slug/geo logic).
- **Migration number collision**: `0017` is the next free local number, but an unmerged
  classification change may also claim `0017` → Reconcile/bump the number at merge time; the
  migration's contents are independent.
- **Re-POST overwrites a moderator's prior edits** for the same URL (idempotent upsert by
  design) → Acceptable; the URL is the stated identity and the actor is a trusted moderator.
- **A leaked moderator API key can create/edit jobs** → Accepted: that is the intended CLI
  use; the key acts as its owner, and only moderators carry the role. Key *management* stays
  cookie-only (unchanged).

## Migration Plan

1. Ship `0017_jobs_moderation.sql` (`users.role`, `jobs.created_by`, `jobs.updated_by`).
   On a persistent DB this is applied manually via `psql` (no versioned runner yet).
2. Regenerate sqlc (`make sqlc`) after adding `UpsertManualJob` / `UpdateManualJob` and the
   `role` column on the user reads.
3. Grant the role: `UPDATE users SET role='moderator' WHERE email='...'`.
4. Deploy backend; manual jobs appear in `/jobs` immediately and in search after the next
   `make reindex`.
5. Rollback: the columns are additive and nullable/defaulted; reverting the code leaves them
   harmlessly unused.

## Open Questions

- Final migration number pending the classification change's merge order (see Risks).
