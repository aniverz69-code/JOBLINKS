## ADDED Requirements

### Requirement: Analysed-jobs list endpoint

The system SHALL provide an authenticated `GET /api/v1/me/tracking/analyses` endpoint that lists the jobs the caller has run the AI fit analysis on, newest first, without invoking the LLM. Each item MUST carry the job's public slug, title, company, a `closed` flag, the analysis `overall_score` and `verdict`, the analysed timestamp, and a `stale` flag (true when the caller's CV, the job content, or the model changed since the analysis was computed). The response MUST include the caller's fit-analysis `quota` (used/limit/remaining) in `meta`. The endpoint accepts a session cookie or an API key.

#### Scenario: List returns analysed jobs with quota

- **WHEN** a signed-in caller who has analysed two jobs requests `GET /api/v1/me/tracking/analyses`
- **THEN** the response is `{ "data": [<two items newest first>], "meta": { "quota": { "used": 2, "limit": 10, "remaining": 8 } } }`, each item carrying slug/title/company/closed/overall_score/verdict/analysed-at/stale, and no LLM call is made

#### Scenario: Closed analysed job is retained with a flag

- **WHEN** the caller analysed a job that has since closed
- **THEN** it still appears in the list with `closed: true`

#### Scenario: Stale analysis is flagged

- **WHEN** the caller's CV was re-uploaded after an analysis was computed
- **THEN** that item is returned with `stale: true`

### Requirement: Tracking routes moved to /me/tracking

The per-user tracking endpoints SHALL be served under `/api/v1/me/tracking` (`""`, `/viewed`, `/pipeline`, `/swipe`, `/analyses`), replacing the previous `/api/v1/me/jobs*` paths, which MUST no longer be registered. This is a breaking API change: clients (the freehire-cli) MUST migrate to `/me/tracking`.

#### Scenario: Canonical tracking path

- **WHEN** a caller requests `GET /api/v1/me/tracking`
- **THEN** it returns the caller's tracked jobs (the listing previously served at `/api/v1/me/jobs`)

#### Scenario: Old path is gone

- **WHEN** a client requests `GET /api/v1/me/jobs`
- **THEN** the system returns `404` — the path is no longer registered

### Requirement: Tracking section renamed with URL redirects

The frontend personal-jobs section SHALL be presented as **Tracking** and served under `/my/tracking/*` (Board, Pipeline, History, AI fit). Requests to the previous `/my/jobs/*` URLs MUST redirect (HTTP 308) to the corresponding `/my/tracking/*` path so existing bookmarks and inbound links keep working.

#### Scenario: Old URL redirects to the new section

- **WHEN** a user opens `/my/jobs/pipeline`
- **THEN** the app redirects to `/my/tracking/pipeline`

#### Scenario: Section labelled Tracking

- **WHEN** a signed-in user opens the tracking section
- **THEN** the navigation and heading read "Tracking", with tabs for Board, Pipeline, History, and AI fit
