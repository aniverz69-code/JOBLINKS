## ADDED Requirements

### Requirement: Dismissing a job

The system SHALL let an authenticated user dismiss a job, idempotently, recording
it as a distinct interaction alongside view/apply/save. Authentication MAY be by
session cookie or by API key; either identifies the acting user identically.
Dismissing sets `user_jobs.dismissed_at`; it works whether or not a view was
recorded first, and repeating it does not create a duplicate or error. Dismissal
is a private triage signal used only to keep a job out of the swipe deck — it
SHALL NOT remove the job from the public `/jobs` list or search. The endpoint
SHALL return the updated interaction record.

#### Scenario: Dismiss a job

- **WHEN** an authenticated user sends `POST /api/v1/jobs/:slug/dismiss`
- **THEN** the job's `dismissed_at` is set
- **AND** the response is `200` with the updated interaction record

#### Scenario: Dismiss is idempotent

- **WHEN** an authenticated user dismisses the same job twice
- **THEN** the row is updated in place each time
- **AND** no duplicate row is created and no error is returned

#### Scenario: Dismiss authenticated by an API key

- **WHEN** a request to `POST /api/v1/jobs/:slug/dismiss` carries a valid
  `Authorization: Bearer <key>` and no cookie
- **THEN** the system dismisses the job for the key's owning user exactly as a
  cookie session would and responds `200`

#### Scenario: Dismiss requires authentication

- **WHEN** a request to `POST /api/v1/jobs/:slug/dismiss` carries neither a valid
  auth cookie nor a valid API key
- **THEN** the system responds `401` and records nothing

#### Scenario: Dismiss does not hide the job elsewhere

- **WHEN** an authenticated user dismisses a job and then loads the `/jobs` list
  or search with filters that match it
- **THEN** the job still appears in the list and search results

### Requirement: Clearing a dismissal

The system SHALL let an authenticated user clear a job's dismissal, idempotently,
so it can re-enter the swipe deck. Clearing SHALL unset `dismissed_at`; clearing
a job that is not currently dismissed SHALL be a no-op that returns success. This
is the undo path for a swipe-left decision.

#### Scenario: Clear a dismissal

- **WHEN** an authenticated user sends `DELETE /api/v1/jobs/:slug/dismiss` for a
  job they previously dismissed
- **THEN** the job's `dismissed_at` is cleared
- **AND** the job is eligible to appear in the swipe deck again

#### Scenario: Clearing a non-dismissed job is a no-op

- **WHEN** an authenticated user sends `DELETE /api/v1/jobs/:slug/dismiss` for a
  job they never dismissed
- **THEN** the system responds successfully and no row is created or errored
