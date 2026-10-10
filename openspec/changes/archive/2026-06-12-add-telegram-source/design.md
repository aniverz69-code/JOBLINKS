# Design — add-telegram-source

## Context

Telegram channels are a first-class vacancy source in the RU/remote IT market,
but posts are free text: zero, one, or many vacancies per message; the company
is named in the text (not the channel); often no apply URL (the post itself, or
a TG handle, is the application surface). This is structurally different from
the ATS adapters, which yield one normalized job per posting.

Research basis (2026-06-12, via the Telagon channel index + live preview
checks): 443 filtered IT-vacancy channels → 424 (96%) have the public
`t.me/s/<channel>` preview enabled → 343 posted within the last 30 days.
Channel typology: *authored* (storytelling, 1 post = N jobs, e.g. hrlunapark),
*board* (semi-structured, 1 post = 1 job, e.g. job_web3), *aggregator bots*
(repost their own job site, e.g. remoteyeah — excluded: they duplicate the ATS
boards we already crawl).

## Goals / Non-Goals

**Goals**

- Ingest vacancies from a curated, configured list of public Telegram channels.
- Idempotent, cron-driven, run-once-and-exit workers, consistent with
  `cmd/ingest` / `cmd/enrich`.
- LLM extraction with the same durability discipline as enrichment (durable
  queue, lease, retry-once, dead-letter; never persist an invalid payload).
- Extracted jobs are ordinary `jobs` rows: they flow through the existing
  enrichment outbox, search indexing, and public API untouched.

**Non-Goals (recorded seams)**

- MTProto/account-based reading (only needed for the ~4% preview-disabled
  channels; the fetch boundary keeps the swap local).
- Cross-channel content dedup (same vacancy in 5 channels → 5 jobs at MVP).
- Post edit/delete tracking (a stored post is immutable).
- Channel auto-discovery (the 343-channel pool is curated by hand into
  `channels.yml`; discovery tooling lives outside this repo).

## Decisions

### D1. Transport: public web preview, not MTProto

`t.me/s/<channel>` returns the ~20 latest posts (id, ISO datetime, HTML text)
with no auth; pagination via `?before=<id>` exists but is not needed for an
hourly crawl of curated channels. Pure HTTP fits the existing shared client,
needs no session/account, and carries no account-ban risk. MTProto (gotd) was
rejected for MVP: it adds session management, FLOOD_WAIT discipline, and an
account as an operational dependency, for a 4% coverage gain. The HTML fetch +
parse lives behind one function boundary so a future MTProto implementation
replaces only that.

Trade-off accepted: the preview markup is not a contracted API. The parser is
fixture-tested; a markup change breaks the crawl loudly (zero posts parsed →
channel counted failed), not silently.

### D2. Telegram does not implement `sources.Source`

The `Source` interface yields normalized jobs; Telegram yields raw posts that
need asynchronous LLM work before jobs exist. Forcing it through the registry
would either put LLM calls inside the crawl (coupling ingest to the LLM and its
latency/cost) or fake empty jobs. Instead Telegram gets its own pipeline:
crawl → `telegram_posts` → extraction → `UpsertJob`. The write path converges
on the same canonical upsert, so everything downstream (enrichment, search,
API) is untouched.

### D3. Extraction before `UpsertJob` (post ≠ job)

A post maps to 0..N vacancies, so classification/extraction must happen before
rows enter `jobs` — `jobs` stays canonical and clean (no "maybe a vacancy"
rows). `telegram_posts` is the durable queue *and* the crawl-dedup record:

- PK `(channel, msg_id)`; insert is `ON CONFLICT DO NOTHING` — re-crawling is
  free and idempotent.
- Same bookkeeping columns as `enrichment_outbox` (`attempts`, `claimed_at`,
  `failed_at`, `last_error`) plus `extracted_at` (non-NULL = done). Rows are
  kept after extraction (unlike outbox deletion) because the row *is* the
  "already seen" record for the next crawl.
- Extraction runs at most once per post; the `/n` suffix in `external_id` is
  assigned then and never re-generated, so index stability across LLM runs is
  a non-issue.

### D4. Cheap heuristic prefilter at insert

Channels mix vacancies with digests, ads, and memes. A keyword/pattern gate
(vacancy/salary/hiring markers, RU+EN) runs at insert: posts that clearly are
not vacancies are stored with `extracted_at = now()` and zero jobs — recorded
(so never re-fetched into the queue) but never sent to the LLM. The filter is
deliberately permissive (recall over precision); the LLM is the real
classifier and may itself return zero vacancies.

### D5. Extraction contract mirrors the enrichment discipline

A typed `Extraction{Jobs []ExtractedJob}` is the contract; `Validate` requires
a non-empty title per job and sane field shapes. Flow per claimed post: LLM →
validate → on invalid, retry once, then dead-letter (`failed_at`) — an invalid
payload is never persisted. On success, one transaction: `UpsertJob` per
extracted job + mark the post extracted. `UpsertJob` already upserts the
company (slug from the extracted company name via `normalize`; empty company
name → jobless company upsert is skipped by the existing query) and enqueues
enrichment atomically. Zero extracted jobs is a normal success (post was not a
vacancy).

The LLM goes through the existing `enrich.Provider` abstraction — same
OpenAI-compatible config, no vendor coupling. The prompt is channel-kind-aware
(`board`: expect exactly one vacancy; `authored`: expect 0..N).

### D6. `channels.yml`, not `sources.yml`

A channel is not a company board: `company` is meaningless (the company is in
the post), and the entry shape differs (`kind` instead of `provider`/`board`
semantics). Overloading `sources.yml` would weaken its validation. Separate
file, same rigor: parsed and validated at `cmd/tg-ingest` startup, unknown
`kind` or duplicate channel fails fast. Path overridable via `CHANNELS_FILE`
(mirrors `SOURCES_FILE`).

### D7. Two binaries, not one

`cmd/tg-ingest` (crawl, DB-only) and `cmd/tg-extract` (LLM) have different
dependencies, failure modes, and natural schedules (crawl hourly; extraction
as often as LLM budget allows). Mirrors the existing `cmd/ingest` /
`cmd/enrich` split. Both reuse `config.Load`.

## Data

```sql
CREATE TABLE telegram_posts (
    channel     TEXT        NOT NULL,
    msg_id      BIGINT      NOT NULL,
    text        TEXT        NOT NULL,           -- plain text extracted from preview HTML
    posted_at   TIMESTAMPTZ NOT NULL,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- extraction bookkeeping (lease/retry/dead-letter, as enrichment_outbox)
    attempts     INT         NOT NULL DEFAULT 0,
    claimed_at   TIMESTAMPTZ,
    failed_at    TIMESTAMPTZ,
    last_error   TEXT        NOT NULL DEFAULT '',
    extracted_at TIMESTAMPTZ,                   -- non-NULL = done (jobs written or none found)
    PRIMARY KEY (channel, msg_id)
);
```

Job mapping: `source = "telegram"`, `external_id = "<channel>/<msg_id>/<n>"`
(n = 0-based index within the post), `url = "https://t.me/<channel>/<msg_id>"`,
`posted_at` = post timestamp, `remote` = extracted hint, `description` =
extracted vacancy text (sanitized, plain-paragraph HTML).

## Risks

- **Preview markup drift** — loud failure mode (parse yields zero posts →
  channel counted failed in stats); fixture-based parser tests document the
  expected shape.
- **LLM extraction quality** — bounded by validate + dead-letter; bad posts
  never corrupt `jobs`. Prompt iteration is cheap (no version bump needed:
  extraction runs once per post, and the post pool is append-only).
- **t.me rate limiting** — sequential fetch with delay; ~35 channels/run is
  negligible. Scaling to the full 343-channel pool stays within ~1 req/10s
  budget at hourly cadence.
