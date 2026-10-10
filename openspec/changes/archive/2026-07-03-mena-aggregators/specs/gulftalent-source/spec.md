## ADDED Requirements

### Requirement: GulfTalent sitemap-driven crawl

The system SHALL provide a `gulftalent` source adapter that crawls GulfTalent.com into the
catalogue by enumerating its sitemap index. The adapter is a **boardless aggregator**: it
fetches `https://www.gulftalent.com/sitemap.xml`, follows the job-posting shards
(`sitemap_jx0NN.xml` — the `jl`/`jc`/`co` shards are category, company, and course pages that
carry no `JobPosting`), collects the job-detail URLs, fetches each detail page, and parses the
embedded `JobPosting` JSON-LD. The crawl is keyless.

#### Scenario: Sitemap enumeration yields all listed postings

- **WHEN** the adapter crawls GulfTalent
- **THEN** it returns one `Job` per job URL enumerated across the job-posting shards, each
  populated from the detail page's `JobPosting` JSON-LD (title, HTML description, company,
  free-text location, apply/detail URL, posted-at)

#### Scenario: Only job-posting shards are followed

- **WHEN** the sitemap index lists both job-posting shards (`jx`) and category/company shards
  (`jl`/`jc`/`co`)
- **THEN** the adapter follows only the `jx` job-posting shards, so category and company pages
  never become jobs, and every `jx` shard is followed so postings in later shards are not
  dropped

### Requirement: Self-contained aggregator company identity

The adapter MUST read each posting's company from the JSON-LD `hiringOrganization.name`,
because GulfTalent is a multi-company aggregator where each posting carries its own
employer. The adapter is registered as an aggregator-marker source so its jobs are stored
under their own company identity.

#### Scenario: Company comes from the posting

- **WHEN** a detail page's `JobPosting` names a `hiringOrganization`
- **THEN** that organization name is the job's company

#### Scenario: Posting without an employer is dropped

- **WHEN** a detail page has no resolvable `hiringOrganization.name`
- **THEN** the adapter omits that posting rather than persisting a company-less job

### Requirement: Stable dedup identity

The adapter SHALL derive each job's `ExternalID` from the stable GulfTalent posting id in
the detail URL, so re-crawling dedups to the same catalogue row.

#### Scenario: Re-crawl dedups

- **WHEN** the same posting is enumerated on a later crawl
- **THEN** it maps to the same `ExternalID` and updates the existing row rather than
  creating a duplicate; a URL with no extractable id is dropped rather than persisted with
  an empty key

### Requirement: Anti-bot fingerprint transport

The adapter MUST issue its requests over the shared Chrome-fingerprint HTTP transport
(TLS + HTTP/2 spoofing), because GulfTalent's Akamai edge rejects the default Go
fingerprint with a 403. Requests still dial through the SSRF-guarded dialer.

#### Scenario: Requests are served, not blocked

- **WHEN** the adapter fetches the sitemap or a detail page
- **THEN** it uses the Chrome-fingerprint transport so the edge serves a 200 rather than
  an anti-bot 403

### Requirement: Resilient parsing fails loudly on the index but isolates per-detail failures

The adapter SHALL error the whole crawl when the **sitemap index** cannot be fetched or parsed
(the index is the loud, catalogue-wide signal), while dropping an individual detail page that
carries no `JobPosting` JSON-LD (a single re-templated posting must not abort a healthy crawl).
An empty sitemap shard is valid and yields no jobs; a single unreadable shard is skipped so it
does not abort the run.

#### Scenario: Unparseable sitemap index errors the crawl

- **WHEN** the sitemap index cannot be fetched or parsed
- **THEN** `Fetch` returns an error rather than an empty success

#### Scenario: Missing JobPosting on a detail drops only that posting

- **WHEN** a fetched detail page has no `JobPosting` JSON-LD
- **THEN** the adapter omits that posting and keeps the other postings from the crawl
