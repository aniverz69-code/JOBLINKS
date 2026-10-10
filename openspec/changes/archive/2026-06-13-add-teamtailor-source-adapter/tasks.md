## 1. Job-link collection helper (TDD)

- [x] 1.1 Test a helper that, given a parsed `*html.Node`, returns the absolute hrefs of all
  `<a>` elements linking a `/jobs/<digits>(-slug)?` path, de-duplicated and in first-seen order
  (a card linking the same job twice yields one entry; a page with no such links yields none)
- [x] 1.2 Implement the helper as a depth-first walk over `html.Node` collecting matching `href`s

## 2. JobPosting ld+json extraction helper (TDD)

- [x] 2.1 Test a helper that, given a parsed `*html.Node`, decodes the first
  `<script type="application/ld+json">` whose JSON has `"@type":"JobPosting"` into a typed
  struct (title, description, datePosted, employmentType, jobLocationType,
  jobLocation.address.{addressLocality,addressCountry}); returns ok=false when no such block
- [x] 2.2 Implement the helper: walk for `ld+json` script nodes, JSON-decode each, return the
  first JobPosting

## 3. Teamtailor adapter — Provider + listing→detail Fetch (TDD)

- [x] 3.1 Test `Provider()` returns `"teamtailor"`
- [x] 3.2 Test `Fetch` GETs `https://<board>/jobs`, and per job-card link GETs the job page,
  yielding the normalized jobs; assert the listing URL and that each job URL is fetched
- [x] 3.3 Implement `teamtailor.go`: `teamtailor` struct over `HTTPClient`, `NewTeamtailor`,
  paginating the listing with `GetHTML`, then `fetchDetails(urls, workers, detail)` GET-ting
  each job page

## 4. Pagination (TDD)

- [x] 4.1 Test a multi-page board: pages 1..N each yield job links, page N+1 yields none → the
  adapter fetches every page until the empty one and yields the union of jobs
- [x] 4.2 Test the page cap bounds a board that never returns an empty page (stops at the cap)

## 5. Field mapping (TDD)

- [x] 5.1 Test mapping: `ExternalID` = numeric id parsed from the `/jobs/<id>-…` URL path;
  `URL` = the job URL; `Title` from JobPosting `title`; `Company = e.Company`;
  `Description = sanitizeHTML(html.UnescapeString(description))` (double-encoded HTML decoded,
  active content stripped, structure kept); `Location = joinNonEmpty(addressLocality, addressCountry)`
- [x] 5.2 Test `PostedAt` derived from `datePosted` via `parseRFC3339`; nil when absent/unparseable
- [x] 5.3 Test `Remote` true when `jobLocationType == "TELECOMMUTE"` and via `isRemote(location/title)`

## 6. Isolation and empty-board behavior (TDD)

- [x] 6.1 Test a failed job-page fetch (or a page without a JobPosting block) for one link drops
  only that posting and still yields the rest (no board abort)
- [x] 6.2 Test an empty listing (page 1 has no job links) yields zero jobs and no error

## 7. Registration and configuration

- [x] 7.1 Register `NewTeamtailor(c)` in `sources.All`; confirm `reg`'s duplicate-provider guard still passes
- [x] 7.2 Add at least one verified `teamtailor` entry to `sources/teamtailor.yml`
  (`board: <career-site-host>`), validated live (>0 jobs in its listing)

## 8. Verification

- [x] 8.1 `go build ./... && go vet ./... && go test ./internal/sources/...` all green
- [x] 8.2 Focused live check: run the adapter (real `Client`) against the configured board and
  confirm real postings normalize (title + sanitized description + id + url + datePosted +
  location); confirm the validated-registry fail-fast still accepts `teamtailor`
