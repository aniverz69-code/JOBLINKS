## ADDED Requirements

### Requirement: join is a registered provider

The system SHALL register a `join` adapter so Join.com career pages can be listed in
`sources.yml`. The adapter SHALL treat the configured `board` value as the numeric Join
company id and enumerate jobs from the public JSON API
`GET https://join.com/api/public/companies/<board>/jobs?page=N&pageSize=<size>`, requesting
successive pages until all pages reported by the response's pagination have been read, so a
company with more jobs than one page is fully enumerated. Because the list response carries no
description, the adapter SHALL fetch each job's detail from
`GET https://join.com/api/public/jobs/<id>` with bounded concurrency; a single failed detail
request SHALL drop only that posting rather than abort the board. The adapter SHALL yield the
normalized job shape (at least title, url, remote flag, description, and the platform's native
posting id), with the `description` rendered from the job's Markdown body to sanitized HTML.
The job `location` SHALL come from the listing item's city (locality and country) when present
and MAY be empty otherwise.

#### Scenario: Join board is enumerated from the public list API

- **WHEN** `sources.yml` lists a board with provider `join` and a numeric company id
- **THEN** the adapter requests `…/companies/<id>/jobs` and, per listed job, fetches
  `…/jobs/<job-id>`, yielding each as the normalized job shape with `external_id` set to the
  API's numeric job id and `url` set to `https://join.com/companies/<company-slug>/<idParam>`

#### Scenario: All pages of a multi-page board are enumerated

- **WHEN** a board's job count spans more than one API page
- **THEN** the adapter requests each page up to the pagination's reported page count and yields
  the jobs from every page

#### Scenario: Description is rendered from Markdown to sanitized HTML

- **WHEN** a Join job's detail is fetched and its body is Markdown
- **THEN** the adapter yields a `description` that is the Markdown rendered to HTML and then
  sanitized (active content stripped, structure such as lists and paragraphs kept)

#### Scenario: A failed detail request drops only that posting

- **WHEN** a board lists several jobs and one job's detail request fails
- **THEN** the failed posting is skipped and every other posting is still yielded, without
  aborting the board

#### Scenario: An empty board yields no jobs without error

- **WHEN** a board's list API reports zero jobs
- **THEN** the adapter yields zero jobs and returns no error, so the board is simply skipped
