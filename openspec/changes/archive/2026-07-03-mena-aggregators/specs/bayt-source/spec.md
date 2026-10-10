## ADDED Requirements

### Requirement: Bayt per-country listing crawl

The system SHALL provide a `bayt` source adapter that crawls Bayt.com's paginated
per-country job listings into the catalogue. The adapter is a **board-based multi-company
aggregator**: each configured entry is a COUNTRY scope carried in the board field (e.g.
`saudi-arabia`, `uae`, `qatar`, `kuwait`, `bahrain`, `oman`, `egypt`, `jordan`), not a
per-company board. For each configured country it walks the listing pages, collects
job-detail URLs, fetches each detail page, and parses the embedded `JobPosting` JSON-LD. The
crawl is keyless (no API key).

#### Scenario: Country scope yields all listed postings

- **WHEN** the adapter crawls a configured country scope
- **THEN** it returns one `Job` per job-detail link discovered across that country's
  listing pages, each populated from the detail page's `JobPosting` JSON-LD (title,
  HTML description, company, free-text location, apply/detail URL, posted-at)

#### Scenario: Pagination is followed to exhaustion

- **WHEN** a country's listings span multiple pages
- **THEN** the adapter advances through the pages until a page yields no new job-detail
  links, so postings beyond the first page are not silently dropped

### Requirement: Self-contained aggregator company identity

The adapter MUST read each posting's company from the JSON-LD `hiringOrganization.name`
rather than a configured board company, because Bayt is a multi-company aggregator where
each posting carries its own employer. The adapter is registered as an aggregator-marker
source so its jobs are stored under their own company identity.

#### Scenario: Company comes from the posting

- **WHEN** a detail page's `JobPosting` names a `hiringOrganization`
- **THEN** that organization name is the job's company

#### Scenario: Posting without an employer is dropped

- **WHEN** a detail page has no resolvable `hiringOrganization.name`
- **THEN** the adapter omits that posting rather than persisting a company-less job

### Requirement: Stable dedup identity

The adapter SHALL derive each job's `ExternalID` from the stable Bayt posting id (the
numeric id in the detail URL / JSON-LD `identifier`), so re-crawling the same country
dedups to the same catalogue row.

#### Scenario: Re-crawl dedups

- **WHEN** the same posting is seen on a later crawl
- **THEN** it maps to the same `ExternalID` and updates the existing row rather than
  creating a duplicate; a posting with no extractable id is dropped rather than persisted
  with an empty key

### Requirement: Anti-bot fingerprint transport

The adapter MUST issue its requests over the shared Chrome-fingerprint HTTP transport
(TLS + HTTP/2 spoofing), because Bayt's Akamai/Cloudflare edge rejects the default Go
fingerprint with a 403. Requests still dial through the SSRF-guarded dialer.

#### Scenario: Requests are served, not blocked

- **WHEN** the adapter fetches a Bayt listing or detail page
- **THEN** it uses the Chrome-fingerprint transport so the edge serves a 200 with real
  markup rather than an anti-bot 403

### Requirement: Resilient parsing isolates per-detail failures but fails loudly on the listing

The adapter SHALL drop an individual detail page that carries no `JobPosting` JSON-LD (a single
re-templated or malformed posting must not abort an otherwise healthy crawl — the same
bounded-fan-out isolation the other detail-fetching adapters use), while a **first listing page**
that fails to fetch SHALL error the whole board (a broken listing is the loud signal, not a
per-posting miss). A listing page with zero job links is valid and simply ends pagination; a
later listing page failing just ends pagination with the earlier pages' jobs intact.

#### Scenario: Missing JobPosting on a detail drops only that posting

- **WHEN** a fetched detail page has no `JobPosting` JSON-LD
- **THEN** the adapter omits that posting and keeps the other postings from the crawl

#### Scenario: A broken first listing page errors the board

- **WHEN** the first listing page for a country cannot be fetched
- **THEN** `Fetch` returns an error rather than an empty success
