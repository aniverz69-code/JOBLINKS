## Why

Errors are currently only visible in per-process stdout logs (`log.Printf`, Fiber's
recover middleware) and the browser console. There is no central place to see that a
panic fired in a cron worker, a 500 escaped a handler, or the SPA threw for a user —
so failures are noticed late, if at all. A single error-tracking backend (Sentry) that
receives errors from the HTTP server, every `cmd/*` worker, and the SvelteKit frontend
gives us one inbox for production failures with stack traces, release, and environment
context.

## What Changes

- Add a small backend Sentry initializer (`internal/observability`) that every Go entry
  point calls: the HTTP server and all `cmd/*` workers. It captures unhandled panics and
  explicitly-reported errors, tags each event with the process/environment, and flushes
  on shutdown so short-lived cron workers deliver their events before exiting.
- Wire Sentry into the Fiber server: report panics recovered by the recover middleware
  and unexpected 5xx errors surfaced by the central error handler.
- Wire Sentry into the SvelteKit frontend (`@sentry/sveltekit`): capture client-side and
  server-side (SSR) errors via the SvelteKit error hooks, gated on a public DSN.
- Make the whole integration **opt-in and env-gated** (no DSN ⇒ disabled, no regression),
  matching the existing optional-integration pattern (Meili/LLM/S3/Telegram). Errors-only
  at start: no performance tracing, no session replay.
- Extend the frontend Content-Security-Policy `connect-src` to allow the Sentry ingest
  host so browser events are not blocked.
- Document the new env vars and DSN/host setup for `freehire-ops`.

## Capabilities

### New Capabilities

- `error-tracking`: Centralized, opt-in error reporting to Sentry across all three
  surfaces (Go HTTP server, Go `cmd/*` workers, SvelteKit frontend). Defines how the
  integration is configured, when it is active, what gets captured, how PII is handled,
  and how short-lived workers guarantee delivery before exit.

### Modified Capabilities

<!-- None: CSP host allowance and worker/server wiring are implementation details of the
     new capability, not changes to existing spec-level behavior. -->

## Impact

- **New dependency (Go):** `github.com/getsentry/sentry-go` (+ its Fiber middleware).
- **New dependency (web):** `@sentry/sveltekit`.
- **New code:** `internal/observability` (backend init/flush helper); Sentry wiring in
  `cmd/server/main.go`, every `cmd/*/main.go`, `internal/handler` error handler; frontend
  `hooks.client.ts` / `hooks.server.ts` (+ instrumentation) and Vite config.
- **Config:** new env vars — `SENTRY_DSN`, `SENTRY_ENVIRONMENT` (backend/workers) and
  `PUBLIC_SENTRY_DSN`, `PUBLIC_SENTRY_ENVIRONMENT` (frontend). All optional.
- **CSP:** `web/svelte.config.js` gains a `connect-src` entry for the Sentry ingest host.
- **Ops:** `freehire-ops` must inject the DSNs; two Sentry projects (frontend + backend)
  created in sentry.io.
- **No API/schema changes**, no breaking changes.
