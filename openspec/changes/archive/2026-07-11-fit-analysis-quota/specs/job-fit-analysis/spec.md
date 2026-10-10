## ADDED Requirements

### Requirement: Per-user monthly fit-analysis quota

The system SHALL limit each user to at most **10 AI fit analyses per rolling 30-day window**, enforced BEFORE the LLM prompt-chain runs on both the synchronous `POST /api/v1/jobs/:slug/fit` endpoint and the streaming `GET /api/v1/jobs/:slug/fit/stream` endpoint. Only the FIRST analysis of a distinct `(user, job)` pair consumes quota: a recompute of a pair the user has already analysed, and re-running the same job, MUST be allowed regardless of the count. Consumption is counted from the persisted `user_job_analysis` rows whose first-analysis timestamp falls within the last 30 days, so an analysis that fails or is never persisted MUST NOT consume quota. The limit applies to every role — there is no staff exemption.

#### Scenario: New job under the limit

- **WHEN** a user who has analysed fewer than 10 distinct jobs in the last 30 days requests an analysis for a job they have not analysed
- **THEN** the system runs the chain, persists the result, and the run counts toward the 30-day window

#### Scenario: New job over the limit

- **WHEN** a user who has already analysed 10 distinct jobs in the last 30 days requests an analysis for a job they have not analysed
- **THEN** the system responds `429`, never invokes the LLM, and persists nothing

#### Scenario: Recompute is always free

- **WHEN** a user at or above the limit requests an analysis for a job they have already analysed (a recompute of an existing `(user, job)` pair)
- **THEN** the system runs the chain and does not reject the request on quota grounds

#### Scenario: Streaming endpoint enforces the same cap

- **WHEN** a user over the limit opens the SSE stream for a job they have not analysed
- **THEN** the system responds `429` before opening the event stream and never invokes the LLM

#### Scenario: Failed analysis does not consume quota

- **WHEN** an under-limit new-job analysis is attempted but the LLM is unconfigured or errors, so no row is persisted
- **THEN** the user's remaining quota is unchanged

### Requirement: Quota state on the read endpoint

The read endpoint `GET /api/v1/jobs/:slug/fit` SHALL return a `quota` object carrying `used`, `limit`, and `remaining` (where `remaining = max(0, limit - used)`) computed over the caller's last 30 days, without invoking the LLM, so the client can display usage and pre-block a new-job analysis when no quota remains.

#### Scenario: Quota reported on read

- **WHEN** a signed-in caller reads `GET /api/v1/jobs/:slug/fit`
- **THEN** the response includes `quota` with `used`, `limit` (10), and `remaining` reflecting the caller's distinct-job analyses in the last 30 days, and no LLM call is made

#### Scenario: Remaining never negative

- **WHEN** a caller has analysed 10 or more distinct jobs in the window
- **THEN** `remaining` is reported as `0`, not a negative number
