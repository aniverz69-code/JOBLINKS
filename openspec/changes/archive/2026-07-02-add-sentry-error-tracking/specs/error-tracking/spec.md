## ADDED Requirements

### Requirement: Opt-in, env-gated activation

Error tracking SHALL be disabled unless a Sentry DSN is configured, and its absence
SHALL NOT alter any other behavior. The backend/workers read `SENTRY_DSN`; the frontend
reads `PUBLIC_SENTRY_DSN`. When the relevant DSN is empty or unset, the corresponding
surface MUST NOT initialize Sentry, MUST NOT attempt network delivery, and MUST run
exactly as it does today.

#### Scenario: Backend runs unchanged without a DSN

- **WHEN** the HTTP server or any `cmd/*` worker starts with `SENTRY_DSN` empty or unset
- **THEN** it MUST NOT initialize Sentry and MUST start and run with no error-tracking side effects

#### Scenario: Frontend runs unchanged without a DSN

- **WHEN** the SvelteKit app renders (client or SSR) with `PUBLIC_SENTRY_DSN` empty or unset
- **THEN** it MUST NOT initialize Sentry and MUST render with no error-tracking side effects

#### Scenario: Backend activates with a DSN

- **WHEN** a Go entry point starts with `SENTRY_DSN` set to a valid DSN
- **THEN** it MUST initialize Sentry once, tagged with the configured environment, before serving traffic or processing work

### Requirement: Backend panic and unexpected-error capture

When active, the HTTP server SHALL report unhandled panics and unexpected server errors
(HTTP 5xx) to Sentry, while continuing to serve the existing JSON error envelope to the
client. Expected client-facing failures — any `*fiber.Error` (4xx), `pgx.ErrNoRows`
mapped to 404, and foreign-key-violation mapped to 404 — SHALL NOT be reported, so the
Sentry inbox reflects genuine faults rather than routine 4xx traffic.

#### Scenario: Recovered panic is reported

- **WHEN** a handler panics and the recover middleware catches it
- **THEN** the panic MUST be captured to Sentry with a stack trace
- **AND** the client MUST still receive the standard 500 JSON error response

#### Scenario: Unexpected 500 is reported

- **WHEN** a handler returns an error that the central error handler maps to HTTP 500
- **THEN** the error MUST be captured to Sentry

#### Scenario: Routine 4xx is not reported

- **WHEN** a handler returns a `*fiber.Error` with a 4xx status, `pgx.ErrNoRows`, or a foreign-key violation
- **THEN** the error MUST NOT be captured to Sentry
- **AND** the client MUST receive the existing mapped status and JSON envelope

### Requirement: Worker error capture with guaranteed delivery

When active, every `cmd/*` worker SHALL report unhandled panics and explicitly-reported
errors to Sentry, and SHALL flush pending events before the process exits. Because these
are short-lived run-once-and-exit processes, delivery MUST NOT depend on a background
flush that the process outlives.

#### Scenario: Worker panic is captured and flushed

- **WHEN** a worker panics during its run
- **THEN** the panic MUST be captured to Sentry and the event MUST be flushed before the process exits with a non-zero status

#### Scenario: Worker flushes on normal exit

- **WHEN** a worker finishes its run (with or without reported errors) and is about to exit
- **THEN** any buffered Sentry events MUST be flushed within a bounded timeout before exit

### Requirement: Frontend client and server error capture

When active, the SvelteKit frontend SHALL report unhandled errors from both the browser
(client) and SSR (server) to Sentry via the framework error hooks, tagged with the
configured environment.

#### Scenario: Client-side error is reported

- **WHEN** an unhandled error is thrown while the app runs in the browser and `PUBLIC_SENTRY_DSN` is set
- **THEN** the error MUST be captured to Sentry from the client

#### Scenario: SSR error is reported

- **WHEN** an unhandled error is thrown during server-side rendering and `PUBLIC_SENTRY_DSN` is set
- **THEN** the error MUST be captured to Sentry from the server

### Requirement: Content-Security-Policy allows Sentry ingest

The frontend Content-Security-Policy SHALL permit the browser to deliver events to the
Sentry ingest host, so that client-side reporting is not blocked by CSP.

#### Scenario: Browser event is not blocked by CSP

- **WHEN** the active frontend attempts to send an error to the Sentry ingest host
- **THEN** the CSP `connect-src` MUST allow that host and the browser MUST NOT block the request

### Requirement: PII is not sent by default

Error tracking SHALL NOT transmit personally identifiable information by default. The
SDKs MUST be configured with default PII sending disabled so that request bodies,
cookies, auth tokens, and user emails are not shipped to Sentry unless a future,
explicit decision enables scoped context.

#### Scenario: Default PII sending is off

- **WHEN** Sentry is initialized on any surface
- **THEN** default PII sending MUST be disabled (no automatic cookies, auth headers, or request bodies attached to events)
