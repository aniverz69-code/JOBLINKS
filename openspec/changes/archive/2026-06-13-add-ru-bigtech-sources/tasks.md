# Tasks

Each adapter follows the same shape as the existing detail-fetching
adapters; tests use a fake `HTTPClient` over canned JSON — no network in unit tests.

## 1. Infra — boardless config (TDD)

- [x] 1.1 Test: in `config_test.go`, `Validate` accepts an entry with an empty `board`
      when its provider implements the `boardless` marker, and still rejects an empty
      `board` for a board-based provider (e.g. `greenhouse`) and an unknown provider
- [x] 1.2 Implement: add `type boardless interface{ boardless() }` in `source.go`; in
      `config.go` `Validate`, skip the empty-board check when
      `registry[c.Provider]` implements `boardless`. No change to existing adapters
- [x] 1.3 Test: a `boardless`-implementing fake source registered under a key validates
      with an empty board; confirm `reg`'s duplicate-provider guard still holds

## 2. Infra — header-aware HTTP client (TDD)

- [x] 2.1 Test: in `http_test.go`, a request issued via `GetJSONWithHeaders` /
      `PostJSONWithHeaders` carries the custom headers alongside `User-Agent`/`Accept`;
      a request via the existing `GetJSON`/`PostJSON` sends no custom header (unchanged)
- [x] 2.2 Implement: add `GetJSONWithHeaders(ctx, url, headers map[string]string, v)` and
      `PostJSONWithHeaders(ctx, url, headers, body, v)` to the `HTTPClient` interface and
      `Client`; have `GetJSON`/`PostJSON` delegate with `nil` headers; set custom headers
      in `do` without overriding `User-Agent`/`Accept`
- [x] 2.3 Update the shared test fake(s) to implement the two new interface methods and to
      record received headers (so adapter tests can assert them)

## 3. Yandex (`yandex`, board ru/com — NOT boardless) (TDD)

- [x] 3.1 Test: cursor pagination follows `next` (extract `cursor`, re-issue on the public
      host) until null; skips `redirect_url != null` and `fast_track != null`; per posting
      fetches `…/publications/{id}` detail; maps `external_id=id`,
      `url=https://yandex.{board}/jobs/vacancies/{publication_slug_url}`, location from
      `vacancy.cities[].name`, description = concat
      `short_summary+duties+key_qualifications+additional_requirements+conditions`,
      `posted_at` from `modified`; empty list → no jobs, no error
- [x] 3.2 Implement `yandex.go` (`Provider()="yandex"`, board ∈ {ru,com}); register
      `NewYandex(c)` in `sources.All`; add `sources/yandex.yml` with `ru` and `com` entries

## 4. Ozon (`ozon`, boardless) (TDD)

- [x] 4.1 Test: page pagination to `meta.totalPages` (`?page=N&limit=50`); keeps only
      `vacancyType=="external_vacancy"`; detail `…/vacancy/{hhId}` →
      `descr`, `slug`, `publishedAt` (`2006-01-02 15:04:05`), `workFormat` (strip nbsp),
      `city`; maps `external_id=hhId`,
      `url=https://career.ozon.ru/vacancy/{slug}/`, `posted_at` parsed; empty → no error
- [x] 4.2 Implement `ozon.go` (boardless marker); register; add `sources/ozon.yml`

## 5. Wildberries (`rwb`, boardless) (TDD)

- [x] 5.1 Test: offset pagination to `data.range.count` (`?limit=200&offset=N`); detail
      `…/pub/vacancies/{id}` → description = concat
      `description+duties_arr+requirements_arr+conditions_arr`; location `city_title`;
      remote from `employment_types[].title` ("Удаленно"); `external_id=id`,
      `url=https://career.rwb.ru/vacancies/{id}`, `posted_at=nil`
- [x] 5.2 Implement `rwb.go` (boardless); register; add `sources/rwb.yml`

## 6. Sber (`sber`, boardless) (TDD)

- [x] 6.1 Test: `skip/take=200` pagination to `data.total`; body inline (no detail call) =
      concat `introduction+duties+requirements+conditions`; `external_id=requisitionId`,
      `url=https://rabota.sber.ru/search/{requisitionId}`, `company` from the row field,
      location `city`, `posted_at=publicationDate` (ISO); remote via `isRemote(location)`
- [x] 6.2 Implement `sber.go` (boardless), throttling between pages (relies on client
      429/5xx backoff for 503s); register; add `sources/sber.yml`

## 7. Alfa-Bank (`alfabank`, boardless) (TDD)

- [x] 7.1 Test: `skip/take=100` pagination to `total`; body inline (`description` HTML);
      cities resolved once via `…/api/vacancies/options?listId=cities`; `external_id=id`,
      `url=https://job.alfabank.ru{slug}`, location = resolved city,
      `posted_at=createdAt` (ISO); remote = `slug` contains `/remote-job/`
- [x] 7.2 Implement `alfabank.go` (boardless); register; add `sources/alfabank.yml`

## 8. Lamoda (`lamoda`, boardless) (TDD)

- [x] 8.1 Test: offset pagination `pagination[limit]/[start]` to `meta.total`
      (URL-encode brackets); detail `…/vacancies/{id}` → `data.attributes` concat
      `introduction+duties+requirements+conditions`; `external_id=id`,
      `url=https://job.lamoda.ru/vacancies/{slug}`, location `location.name`,
      `posted_at=externalPublicationDate` (ISO); remote via `isRemote(location)`
- [x] 8.2 Implement `lamoda.go` (boardless); register; add `sources/lamoda.yml`

## 9. Kuper (`kuper`, boardless) (TDD)

- [x] 9.1 Test: page pagination using `pages` from the `category:"pagination"` block; body
      inline (`description`); `external_id=id`, location `city`, remote from `wf[]`,
      `posted_at=nil`; URL synthesized best-effort (confirm in §18)
- [x] 9.2 Implement `kuper.go` (boardless); register; add `sources/kuper.yml`

## 10. Aviasales (`aviasales`, boardless) (TDD)

- [x] 10.1 Test: single-call list (plain array); detail loader
      `…/about/vacancies/{id}?__loader=about/vacancies/(id)/page&__ssrDirect=true`
      (encode the literal `(id)` parens) → `{vacancy}` concat
      `description+todo+requirements+conditions`; `external_id=id`,
      `url=https://www.aviasales.ru/about/vacancies/{id}`, location/remote from `workPlace`
- [x] 10.2 Implement `aviasales.go` (boardless); register; add `sources/aviasales.yml`

## 11. Dodo (`dodo`, boardless) (TDD)

- [x] 11.1 Test: single-call list, flatten `data[].items[]`; detail
      `…/api/v1/pages/vacancy/{id}` → concat `.text` of typed blocks `vacancy_text`,
      `vacancy_expectation`, `vacancy_you_will`, `vacancy_benefits`; `external_id=id`,
      `company` from `brand`, location `vacancy_location`, remote from `work_format[]`
- [x] 11.2 Implement `dodo.go` (boardless); register; add `sources/dodo.yml` (URL best-effort, §18)

## 12. DomClick (`domclick`, boardless) (TDD)

- [x] 12.1 Test: single-call list `https://career.domclick.ru/api/v1/vacancy/` (`result[]`);
      detail `…/api/v1/vacancy/detail/{slug}/` → `vacancycontent.branded_description`
      (fallback `.description`); `external_id=slug`,
      `url=https://career.domclick.ru/vacancy/{slug}`, location `area.name`, remote from
      `vacancycontent.work_format[]` (`REMOTE/HYBRID/ON_SITE`)
- [x] 12.2 Implement `domclick.go` (boardless); register; add `sources/domclick.yml`

## 13. MTS Link (`mtslink`, boardless) (TDD)

- [x] 13.1 Test: single-call list `…/api/huntflow/vacancies` (keep `state=="OPEN"`); detail
      `…/api/huntflow/vacancy/{id}` (singular) → concat `body+requirements+conditions`;
      `external_id=id`, `url=https://mts-link.ru/vacancies/{id}/`, location/remote from
      `workFormat` ("Удаленный"), `posted_at=created.date` (`2006-01-02 15:04:05.000000`)
- [x] 13.2 Implement `mtslink.go` (boardless); register; add `sources/mtslink.yml`

## 14. T-Bank (`tbank`, boardless — POST) (TDD)

- [x] 14.1 Test (body-aware fake): list `POST …/papi/getVacancies` body
      `{"pagination":{"publisher":{"offset":N}},"limit":20,"filters":{}}`, loop until
      `nextPagination.publisher.isFinished`; detail `POST …/papi/getVacancyDescription`
      body `{"source":"publisher","urlSlug":…,"options":{"category":…}}`, walk the mixed
      `description[]` array (HTML `{key,content}` and structured `{title,content:[…]}`);
      `external_id=urlSlug`, location `subtitle`, remote from `tags[]`
- [x] 14.2 Implement `tbank.go` (boardless, uses `PostJSON`); register; add `sources/tbank.yml`
      (URL best-effort, §18)

## 15. MTS (`mts`, boardless — POST + x-api-key) (TDD)

- [x] 15.1 Test: list `POST …/v1/vacancies/filtered/career` body `{"offset":N,"limit":200}`
      with header `x-api-key`, loop to `data.pageInfo.total`; detail
      `GET …/v1/vacancy/{id}` with `x-api-key` → `data.vacancy.detailText` concat
      `descriptionOfProject+description+requirements+conditions`; `external_id=id`,
      `url=https://job.mts.ru/vacancy/{id}`, `company=info.brand`, location `info.city`,
      remote from `info.worktype`; assert the `x-api-key` header is sent
- [x] 15.2 Implement `mts.go` (boardless): harvest the public key from the Nuxt runtime
      config at `Fetch` time; a harvest failure fails only this board. Register; add
      `sources/mts.yml`

## 16. Tochka (`tochka`) — DEFERRED (fragile RSC flight-payload; see proposal Out of scope)

- [ ] 16.1 Test: list `…/api/v2/hr/vacancies/?page=N&page_size=20` to `meta.total`; detail
      `GET …/vacancies/catalog/{slug}/` with header `RSC: 1`, parse the flight payload for
      the vacancy object (fixture), concat
      `detail+aboutTeam+requirement+responsibility+expectation+stack`; `external_id=slug`,
      `url=https://hr.tochka.com/vacancies/catalog/{slug}/`, location `city`, remote from
      `workFormat` (`remotely`), `posted_at=publishedAt` (`2006-01-02`)
- [ ] 16.2 Implement `tochka.go` (boardless): isolate the RSC flight-payload parse; a parse
      failure drops only that posting. Register; add `sources/tochka.yml`

## 17. VK (`vk`, boardless — HTML scrape) (TDD)

- [x] 17.1 Test: list `…/career/api/v2/vacancies/?limit=50&offset=0` follows `next`/offset
      to `count`; per posting the HTML extractor (over a captured page fixture, via
      `golang.org/x/net/html`) returns the description sections, sanitized; `external_id=id`,
      `url=https://team.vk.company/vacancy/{id}/`, location `town.name`, remote from the
      `remote` bool; a failed fetch/extract drops only that posting
- [x] 17.2 Implement `vk.go` (boardless) with an isolated HTML-extraction helper; register;
      add `sources/vk.yml`

## 18. Registration sweep, URL confirmation, verification

- [x] 18.1 Confirm all 14 `NewX(c)` lines are in `sources.All` and `reg`'s
      duplicate-provider guard passes; confirm all 14 `sources/<provider>.yml` files exist
      and validate (boardless entries with empty board accepted; `yandex` requires board)
- [ ] 18.2 Live-confirm the best-effort public vacancy URLs for `kuper`, `dodo`, `tbank`;
      fix the URL builder (or store the source URL and flag) per what resolves
- [x] 18.3 `go build ./... && go vet ./... && go test ./internal/...` all green
- [ ] 18.4 Smoke-run `go run ./cmd/ingest` against a small subset (override `SOURCES_FILE`)
      to confirm real postings normalize and upsert and the fail-fast registry accepts the
      new providers
