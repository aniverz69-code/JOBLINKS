## 1. Markdown→HTML helper (TDD)

- [x] 1.1 Add `github.com/yuin/goldmark` as a direct dependency (`go get`); `go mod tidy`
- [x] 1.2 Test a helper `markdownToHTML(md string) string` that renders Markdown to HTML:
  a `*   ` bulleted list becomes `<ul><li>…`, paragraphs become `<p>…</p>`, and empty input
  yields empty output
- [x] 1.3 Implement the helper over `goldmark.Convert` into a buffer

## 2. Join adapter — Provider + list→detail Fetch (TDD)

- [x] 2.1 Test `Provider()` returns `"join"`
- [x] 2.2 Test `Fetch` GETs `…/companies/<board>/jobs?page=1&pageSize=…`, and per listed item
  GETs `…/jobs/<id>`, yielding the normalized jobs; assert the list URL and that each job URL
  is fetched
- [x] 2.3 Implement `join.go`: `join` struct over `HTTPClient`, `NewJoin`, the list request
  decoding `{items, pagination}`, then `fetchDetails(items, workers, detail)` GET-ting each
  `…/jobs/<id>`

## 3. Pagination (TDD)

- [x] 3.1 Test a multi-page board: pagination reports `pageCount > 1`; the adapter requests each
  page (`page=1..pageCount`) and yields the union of jobs
- [x] 3.2 Test a single-page board (`pageCount == 1`) issues exactly one list request

## 4. Field mapping (TDD)

- [x] 4.1 Test mapping: `ExternalID` = API job `id`; `URL` =
  `https://join.com/companies/<company.domain>/<idParam>`; `Title` = item `title`;
  `Company = e.Company`; `Location = joinNonEmpty(city.cityName, city.countryName)`;
  `Description = sanitizeHTML(markdownToHTML(detail.description))` (lists/paragraphs kept,
  active content stripped)
- [x] 4.2 Test `PostedAt` from `createdAt` via `parseRFC3339`; nil when absent/unparseable
- [x] 4.3 Test `Remote` true when `workplaceType == "REMOTE"` and via `isRemote(location/title)`

## 5. Isolation and empty-board behavior (TDD)

- [x] 5.1 Test a failed job-detail request for one item drops only that posting and still yields
  the rest (no board abort)
- [x] 5.2 Test an empty board (list reports zero items / `pageCount == 0`) yields zero jobs and
  no error

## 6. Registration and configuration

- [x] 6.1 Register `NewJoin(c)` in `sources.All`; confirm `reg`'s duplicate-provider guard passes
- [x] 6.2 Add at least one verified `join` entry to `sources/join.yml` (`board: <numeric company
  id>`, slug/name in a comment), validated live (>0 jobs from the list API)

## 7. Verification

- [x] 7.1 `go build ./... && go vet ./... && go test ./internal/sources/...` all green
- [x] 7.2 Focused live check: run the adapter (real `Client`) against the configured board and
  confirm real postings normalize (title + sanitized-HTML description + id + url + createdAt +
  location); confirm the validated-registry fail-fast still accepts `join`
