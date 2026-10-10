## 1. Schema + queries (telegram_posts)

- [x] 1.1 Migration `migrations/0008_telegram_posts.sql` (or next number): table per design — PK (channel, msg_id), text, posted_at, fetched_at, attempts/claimed_at/failed_at/last_error/extracted_at
- [x] 1.2 Write failing integration tests (`-tags=integration`, testcontainers, mirroring the outbox tests): InsertPost is ON CONFLICT DO NOTHING; ClaimPendingPosts leases with FOR UPDATE SKIP LOCKED and skips done/dead/fresh-leased rows; expired lease is reclaimable; MarkPostExtracted / MarkPostFailed / RecordPostError behave per design
- [x] 1.3 Hand-write `internal/db/queries/telegram_posts.sql`, run `make sqlc`, make tests green

## 2. channels.yml config

- [x] 2.1 Write failing tests in `internal/telegram`: parses channel+kind entries; fails on unknown kind, empty channel, duplicate channel
- [x] 2.2 Implement config parsing/validation (mirror `internal/sources/config.go` style); make green

## 3. Preview fetch + parse

- [x] 3.1 Write failing parser tests against a saved `t.me/s/<channel>` HTML fixture: yields msg_id, timestamp, plain text per post; entity decoding and <br> → newline; page with zero posts → error/empty distinguishable
- [x] 3.2 Implement fetch (shared HTTP client, polite delay between channels) + parse behind one boundary (the MTProto seam); make green

## 4. Prefilter

- [x] 4.1 Write failing tests: vacancy-marker posts pass (RU + EN markers, salary patterns); obvious non-vacancy (short ad/digest with no markers) is filtered; bias to recall documented in cases
- [x] 4.2 Implement the heuristic; make green

## 5. Crawl runner + cmd/tg-ingest

- [x] 5.1 Write failing runner tests (fake fetcher + fake store): every configured channel crawled once; new posts inserted, filtered posts inserted as done; one failing channel counted, run continues; stats (stored/filtered/failed) reported
- [x] 5.2 Implement the crawl runner in `internal/telegram`; make green
- [x] 5.3 Implement `cmd/tg-ingest/main.go`: config.Load + CHANNELS_FILE, fail-fast validation, run once, log stats, exit

## 6. Extraction contract

- [x] 6.1 Write failing tests: `Extraction.Validate` rejects empty-title jobs and malformed shapes; accepts zero jobs; ExtractedJob fields per design
- [x] 6.2 Implement the typed contract in `internal/telegram`; make green

## 7. Extraction runner + cmd/tg-extract

- [x] 7.1 Write failing runner tests (fake Provider + real/fake store): claimed post → N UpsertJob calls with source=telegram, namespaced external_id `<channel>/<msg_id>/<n>`, t.me URL, post timestamp; post marked extracted in the same transaction; zero-vacancy result marks done with no jobs; invalid payload retries once then dead-letters; company name → slug via normalize
- [x] 7.2 Implement the extraction runner (reuse `enrich.Provider`; kind-aware prompt: board = exactly one vacancy expected, authored = 0..N); make green
- [x] 7.3 Implement `cmd/tg-extract/main.go`: config.Load incl. LLM env, drain bounded batch, log processed/jobs/failed, exit

## 8. Seed data

- [x] 8.1 Create `channels.yml` with the curated tier-1 list (~35 verified-active channels from the 2026-06-12 research, kinds assigned; aggregator-bot channels excluded)

## 9. Verify

- [x] 9.1 `go build ./... && go vet ./... && go test ./...` green; `go test -tags=integration ./internal/db/` green
- [x] 9.2 Live smoke: `go run ./cmd/tg-ingest` against dev DB stores posts from 2–3 channels; `go run ./cmd/tg-extract` extracts real vacancies (spot-check hrlunapark multi-vacancy post and a board channel single-vacancy post); extracted jobs visible via the API and queued for enrichment
