## 1. Database

- [x] 1.1 Add `migrations/0006_user_jobs.sql`: `user_jobs` table (`user_id` BIGINT FK→`users(id)` ON DELETE CASCADE, `job_id` BIGINT FK→`jobs(id)` ON DELETE CASCADE, `viewed_at` timestamptz NOT NULL default now(), `applied_at` timestamptz NULLABLE) with `PRIMARY KEY (user_id, job_id)`
- [x] 1.2 Add `internal/db/queries/user_jobs.sql`: `RecordJobView` (upsert on `(user_id, job_id)`, `DO UPDATE SET viewed_at = now()`, `RETURNING *`) and `MarkJobApplied` (upsert with `applied_at = now()`, `DO UPDATE SET applied_at = now()`, `RETURNING *`)
- [x] 1.3 Run `make sqlc` and commit the regenerated `internal/db` code
- [x] 1.4 Recreate the dev DB volume (`docker compose down -v && make up`) to apply the new migration

## 2. DB query tests

- [x] 2.1 Integration test (`integration` build tag, testcontainers — mirror the queue tests): `RecordJobView` creates then refreshes one row without duplicating; `MarkJobApplied` sets `applied_at`, works with no prior view, and is idempotent

## 3. HTTP handlers (`internal/handler/user_jobs.go`)

- [x] 3.1 Add an API interaction type (`job_id`, `viewed_at`, `applied_at` with `applied_at` nullable/omitempty) and a mapping from the `db.UserJob` row
- [x] 3.2 `RecordView` handler: read user id from `c.Locals`, parse `:id` (client error on a non-numeric id), `RecordJobView`, return `200 {"data": interaction}`
- [x] 3.3 `MarkApplied` handler: read user id from `c.Locals`, parse `:id`, `MarkJobApplied`, return `200 {"data": interaction}`
- [x] 3.4 Wire routes in `handler.Register`: `POST /api/v1/jobs/:id/view` and `POST /api/v1/jobs/:id/apply`, both guarded by `RequireAuth`; leave `GET /api/v1/jobs/:id` public and unchanged
- [x] 3.5 Handler tests: 401 without an auth cookie; non-numeric `:id` → client error; interaction serialization contract (omits user_id). Happy path + idempotency are covered at the query layer by the `integration` test (2.1), matching the repo's testing split (handlers unit-tested for no-DB logic; DB behavior tested in the `db` package)

## 4. Web (SPA)

- [x] 4.1 Add `UserJob` wire type to `web/src/lib/types.ts`; add `recordJobView(id)` and `markJobApplied(id)` to `web/src/lib/api.ts` (POST with `credentials: 'include'`, returning the interaction)
- [x] 4.2 `JobView.svelte`: on open, when `auth.user` is signed in, call `recordJobView` and keep `applied_at`; render a "You applied" badge when `applied_at` is set
- [x] 4.3 `JobView.svelte`: after the user clicks Apply (external link), reveal an inline "Did you apply? [Yes, save] [No]" block — Yes → `markJobApplied` then show the badge; No → hide the block locally with no request; hide the whole prompt when already applied or signed out

## 5. Verification

- [x] 5.1 `go build ./... && go vet ./... && go test ./...` all pass
- [x] 5.2 Manual e2e (API): signed-in `POST /jobs/:id/view` → 200 with interaction; `POST /jobs/:id/apply` → `applied_at` set; both without a cookie → 401; `GET /jobs/:id` still works with no cookie
- [x] 5.3 Manual e2e (web): open a job signed in → view recorded; follow Apply → confirm Yes → "You applied" badge; reopen → badge persists; No → no record; signed-out view unchanged
- [x] 5.4 Update `AGENT.md` (layout + conventions) to document the `user_jobs` table, the view/apply endpoints, and the SPA interaction surface
