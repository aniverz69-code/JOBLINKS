## Why

A large share of the Russian-speaking / remote-friendly IT job market is posted
only in Telegram channels and never reaches an ATS board freehire can crawl.
Research over the Telagon channel index (1.13M+ candidate channels surveyed)
found **343 active IT-vacancy channels** with public web previews (96% of the
filtered pool) — curated boards (e.g. `@it_vakansii_jobs`), junior/intern boards
(`@jobforjunior`), role-specific boards (`@forallqa`, `@job_python`), corporate
channels (`@ya_jobs`, `@avito_career`), and authored channels (`@hrlunapark`).
None of this inventory is reachable today: Telegram posts are unstructured text
(a post may hold zero, one, or several vacancies), so they cannot enter the
catalogue through the existing ATS `Source` adapters.

## What Changes

- **New crawl worker `cmd/tg-ingest`** (run-once-and-exit, cron like
  `cmd/ingest`): reads a new `channels.yml` (channel username + `kind:
  authored|board`), validates entries and fails fast, then fetches each
  channel's latest posts from the public web preview (`t.me/s/<channel>`) over
  the shared HTTP client — no Telegram account, no MTProto. Per-channel failures
  are counted, not fatal.
- **New `telegram_posts` table**: one row per `(channel, msg_id)` — the crawl
  dedup key — holding the post text and `posted_at`, plus the same
  lease/retry/dead-letter bookkeeping as `enrichment_outbox`. Rows persist after
  extraction so a re-crawl never re-processes a seen post. A cheap heuristic
  prefilter marks obvious non-vacancy posts done-with-zero-jobs at insert, so
  they never reach the LLM.
- **New extraction worker `cmd/tg-extract`** (run-once-and-exit, cron): claims
  pending posts (`FOR UPDATE SKIP LOCKED` + lease), asks the existing
  provider-agnostic LLM `Provider` to classify and extract **0..N structured
  vacancies per post** (title, company, location, salary text, remote hint,
  description), validates the payload (retry once, then dead-letter), and on
  success writes each vacancy through the existing `UpsertJob` path in one
  transaction with marking the post extracted — so extracted jobs flow into the
  normal enrichment outbox automatically.
- Extracted jobs are stored with `source = "telegram"`, `external_id =
  "<channel>/<msg_id>/<n>"`, `url = https://t.me/<channel>/<msg_id>` (Telegram
  vacancies usually have no apply link — the post is the application surface),
  and the company upserted from the extracted company name via the existing slug
  normalization.
- **Seed `channels.yml`** with the curated tier-1 list from the research
  (~35 verified-active IT boards; the 343-channel pool is the expansion
  backlog).

## Capabilities

### New Capabilities

- `telegram-ingest`: crawling configured Telegram channels via the public web
  preview, storing posts idempotently, LLM-extracting 0..N vacancies per post
  through a durable queue, and writing them through the canonical job upsert.

### Modified Capabilities

<!-- none — source-ingest, job-enrichment, and the public API are untouched;
     telegram jobs enter through the existing UpsertJob write path. -->

## Impact

- **Code**: new `internal/telegram` package (channels.yml config, t.me/s HTML
  parsing, prefilter, extraction contract + runner), `cmd/tg-ingest`,
  `cmd/tg-extract`; new migration + sqlc queries for `telegram_posts`. Reuses
  `enrich.Provider` (LLM), `normalize` (slugs), `db.UpsertJob`, and the shared
  sources HTTP client. No change to handlers, jobview, search, or the SPA.
- **Data**: new `telegram_posts` table; `jobs` gains rows with
  `source = "telegram"`. No change to existing tables.
- **Ops**: two new cron entries. `cmd/tg-extract` needs the LLM env
  (`LLM_BASE_URL`/`LLM_API_KEY`/`LLM_MODEL`); `cmd/tg-ingest` needs only
  `DATABASE_URL`. ~35 channels × 1 fetch/run is far below t.me rate limits;
  fetches are sequential with a polite delay.
- **Known seams (not built)**: MTProto transport behind the same fetch boundary
  for preview-disabled channels (~4% of the pool); cross-channel duplicate
  detection (same vacancy posted in several channels is stored once per
  channel); post edit/delete handling (stored posts are immutable); aggregator
  link resolution (channels reposting from job sites are excluded from
  `channels.yml` instead).
