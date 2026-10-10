## 1. Shared Chrome-fingerprint transport

- [x] 1.1 Extract `internal/sources/metahttp.go` into a provider-neutral
  `internal/sources/fingerprinthttp.go` (`fingerprintHTTP` with `get`/`GetHTML`/`GetXML`
  and the `safehttp` SSRF-guarded dialer). Re-point the Meta adapter at it; move the SSRF
  test (`TestMetaHTTPBlocksInternalTarget` → `TestFingerprintHTTPBlocksInternalTarget`) and
  keep it green.
- [x] 1.2 In `All()`, build the fingerprint transport **once** and share it across
  `meta`, `bayt`, `gulftalent` (nil-client marker/listing path stays transport-free).
  `go build ./...` + `go test ./internal/sources/` pass.

## 2. Bayt adapter (`internal/sources/bayt.go`)

- [x] 2.1 Build faithful inline HTML fixtures (`baytDetailHTML`/`baytListingHTML`) mirroring
  the real Bayt JSON-LD structure captured from a live page — matching the repo's JSON-LD
  adapter pattern (metacareers/globalpayments use inline builders, not `testdata/`), which is
  deterministic and network-free.
- [x] 2.2 JSON-LD detail parse: failing test that a detail fixture yields a `Job` with
  title, HTML description, company (`hiringOrganization.name`), free-text location
  (`jobLocation.address` locality + ISO country), posted-at (`datePosted`), and `ExternalID`
  (numeric id from the URL). A detail with no `JobPosting`/no company is dropped.
- [x] 2.3 Listing pagination: failing test that a listing fixture yields all job-detail
  hrefs and that pagination stops on a page with no new links; implemented the walker.
- [x] 2.4 `Fetch` wiring + marker: `Provider() == "bayt"`, board-based (board = country slug),
  `aggregator()` marker; company from the posting (drop company-less and id-less postings);
  bounded detail fan-out (`baytDetailWorkers = 3`, conservative for Bayt's throttling edge).
  Re-crawl dedup identity is the URL id.
- [x] 2.5 Register `NewBayt(fp)` in `All()` over the shared fingerprint transport; add
  `sources/bayt.yml` with the 8 live-validated country scopes (sa/ae/qa/kw/bh/om/eg/jo).
  Config validation accepts the board file (verified via `cmd/ingest`).

## 3. GulfTalent adapter (`internal/sources/gulftalent.go`)

- [x] 3.1 Build faithful inline fixtures (`gtSitemapIndexXML`/`gtShardXML`/`gtDetailHTML`)
  mirroring the real sitemap index, a `jx` shard, and a detail JobPosting captured live.
- [x] 3.2 Sitemap enumeration: failing test that the index + shard fixtures yield the
  job-detail URLs, following ONLY the `jx` job-posting shards (jl/jc/co skipped); implemented
  the enumerator. Live capture corrected the shard marker from `jl` to `jx`.
- [x] 3.3 Detail JSON-LD parse: failing test that a detail fixture yields a `Job`
  (title/description/company/location/posted-at RFC3339/`ExternalID` from the URL id);
  implemented. A detail with no `JobPosting`/no company is dropped; an unparseable index errors.
- [x] 3.4 `Fetch` wiring + markers (`Provider() == "gulftalent"`, `boardless()`,
  `aggregator()`), company from the posting, drop company-less/id-less, bounded fan-out
  (`gulftalentDetailWorkers = 4`). Dedup identity is the URL id.
- [x] 3.5 Register `NewGulfTalent(fp)` in `All()` over the shared fingerprint transport; add
  `sources/gulftalent.yml`. Config validation accepts the board file (verified via `cmd/ingest`).

## 4. Live validation + mega-employer seed

- [x] 4.1 Live-validate: bayt/bahrain ingested end-to-end into a scratch Postgres — 237
  real jobs, failed=0, company from the posting, location + derived `{bh}` country facet
  correct; re-run added only new live-feed postings with no dedup errors (unique constraint
  holds). **Surfaced and fixed a production bug: the transport's `Chrome_133` profile was 403'd
  by Bayt's Akamai** — bumped to `Chrome_144` (verified 200 for Bayt, GulfTalent, and Meta).
  GulfTalent validated at transport (sitemap 200) + parse (unit tests) level; its full ~23k
  crawl runs on the prod cron rather than in the bounded smoke.
- [x] 4.2 Mega-employer seed — **finding: the keyless-ATS seed is largely redundant; +2 net-new
  Gulf boards**. Live-probed the major ME tech employers (Careem, Tamara, Sylndr, Ziina, Lean,
  Thndr, Unifonic, Rewaa, …) against greenhouse/lever/ashby/smartrecruiters/workable/recruitee.
  Nearly all confirmed keyless-ATS boards were **already in `sources/*.yml`**; net-new and added:
  **Foodics** and **Salla** (both Saudi, Workable — 25 and several live jobs). The large Gulf
  enterprises (Aramco, SABIC, banks, telecoms, Emirates, ADNOC, HungerStation) run enterprise ATS
  (Workday/SuccessFactors/Taleo/Oracle) with complex per-tenant board ids — a separate harvest,
  **noted as a follow-up seam**, not forced into this change. This validates the strategy: the
  MENA breadth comes from the two aggregators, with the seed a thin top-up.
- [x] 4.3 Ops note (recorded in proposal Impact + design Migration Plan): after the first prod
  ingest of `sources/bayt.yml` / `sources/gulftalent.yml`, run `cmd/backfill-derive` +
  `make reindex`, then re-check the `regions=mena` company facet against the 2,357 baseline.
  Two new cron schedules (one per board file) land in `freehire-ops`. **Follow-up seam —
  high-value Gulf enterprises on freehire-SUPPORTED enterprise ATS, needing per-tenant board
  ids validated against the adapter** (harvested tenant hints): NEOM → Eightfold
  (`careers.neom.com`, board=host; empty at check time, re-add when hiring); Almarai →
  SuccessFactors (`career5.successfactors.eu` company `AlMaraiP`); stc → SuccessFactors
  (`careers.stc.com.sa`, tenant `5075ac53`); Aramco (`careers.aramco.com`) / SABIC
  (`jobs.sabic.com`) → SuccessFactors; SNB (`snbtalent.alahli.com`) / Mobily (Oracle Recruiting
  Cloud `CX_2001`) → Oracle; HungerStation → Workday (`deliveryhero.wd3.myworkdayjobs.com`);
  Nana → Workable (`nana-grocery-direct`, empty now). These are the real MENA depth, addable in a
  focused follow-up PR.

## 5. Verification

- [x] 5.1 `go build ./... && go vet ./... && go test ./...` all green; `gofmt` clean
  (full module verified).
- [x] 5.2 `openspec validate mena-aggregators --strict` passes; code review run on the adapter
  diff — 1 Important (spec↔code drop-vs-error) + minors, all resolved. Post-review changes
  (Chrome_144 profile fix, query-strip) are live-verified.
