## ADDED Requirements

### Requirement: teamtailor is a registered provider

The system SHALL register a `teamtailor` adapter so Teamtailor career sites can be listed in
`sources.yml`. The adapter SHALL treat the configured `board` value as the career-site host and
enumerate jobs from that site's `GET https://<board>/jobs` listing HTML, taking each job-card
anchor to a `/jobs/<id>-<slug>` path as a job (with the job's native id as the leading numeric
segment of that path). The adapter SHALL paginate the listing via `?page=N`, requesting
successive pages until a page yields no job links, so boards larger than one listing page are
fully enumerated. Because the listing carries no description, the adapter SHALL fetch each job
page and extract the posting from the page's schema.org JobPosting `application/ld+json` block,
with bounded concurrency; a single failed page fetch SHALL drop only that posting rather than
abort the board. The adapter SHALL yield the normalized job shape (at least title, url, remote
flag, description, and the platform's native posting id), with the `description` as sanitized
HTML (HTML-unescaped before sanitizing, since the `ld+json` description is double-encoded),
consistent with the existing adapters. The job `location` SHALL come from the JobPosting's
`jobLocation` address (locality and country) when present and MAY be empty otherwise.

#### Scenario: Teamtailor board is enumerated from its listing page

- **WHEN** `sources.yml` lists a board with provider `teamtailor` and a career-site host
- **THEN** the adapter fetches `https://<host>/jobs`, and per job-card anchor fetches the job
  page, yielding each as the normalized job shape with `external_id` set to the numeric id from
  the job URL and `url` set to that job URL

#### Scenario: Title, description, and date come from the JobPosting ld+json

- **WHEN** a Teamtailor job page is fetched
- **THEN** the adapter yields the job's title from the JobPosting `title`, a sanitized HTML
  description from its HTML-unescaped `description`, `posted_at` parsed from `datePosted`, and
  `location` from the `jobLocation` address when present

#### Scenario: A multi-page board is fully enumerated

- **WHEN** a board's listing spans more than one page
- **THEN** the adapter requests successive `?page=N` pages until one yields no job links, and
  yields the jobs from every page

#### Scenario: A failed job-page fetch drops only that posting

- **WHEN** a board's listing yields several jobs and one job page's fetch fails
- **THEN** the failed posting is skipped and every other posting is still yielded, without
  aborting the board

#### Scenario: An empty listing yields no jobs without error

- **WHEN** a board's `/jobs` listing yields no job links
- **THEN** the adapter yields zero jobs and returns no error, so the board is simply skipped
