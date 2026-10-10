## 1. Shared sanitizer

- [x] 1.1 Add `github.com/microcosm-cc/bluemonday` dependency (`go get`, tidy)
- [x] 1.2 Write a failing test for a `sanitizeHTML` helper in `internal/sources`: asserts `<script>` and inline event handlers (`onclick`) are stripped while structural tags (`<h2>`, `<p>`, `<ul>/<li>`, `<strong>`, `<a href>`) survive
- [x] 1.3 Implement `sanitizeHTML` over a package-level compiled `UGCPolicy`; make the test green

## 2. Greenhouse adapter

- [x] 2.1 Write a failing test: given entity-encoded `content` (`&lt;h2&gt;Role&lt;/h2&gt;`), the adapter yields decoded, sanitized HTML (`<h2>Role</h2>`)
- [x] 2.2 Decode with `html.UnescapeString` then `sanitizeHTML`; make the test green

## 3. Lever adapter

- [x] 3.1 Write a failing test for the multi-field assembly: `description` + per-list `<h3>{text}</h3>{content}` + `additional` concatenated, including the case where `descriptionPlain` is empty but `description`/`lists`/`additional` are present
- [x] 3.2 Add the HTML source fields (`description`, `lists[].text/content`, `additional`) to the lever response struct, assemble + `sanitizeHTML`; make tests green (cover both a multi-list posting and a description-only posting)

## 4. Ashby adapter

- [x] 4.1 Write a failing test: adapter yields sanitized `descriptionHtml` (not `descriptionPlain`)
- [x] 4.2 Switch the ashby response struct/field to `descriptionHtml` + `sanitizeHTML`; make the test green

## 5. SPA rendering

- [x] 5.1 In `web/src/lib/components/JobView.svelte`, render the description via `{@html job.description}` inside a styled prose container (headings/lists/emphasis legible), replacing the escaped `{job.description}`
- [x] 5.2 Manually verify a greenhouse job (Dropbox Data Engineer) and a lever job (Spotify Partner Marketing Manager) render formatted, complete descriptions — DB layer verified (full sanitized HTML, no img/script); browser render confirmed by user

## 6. Backfill & verify

- [x] 6.1 `go build ./... && go vet ./... && go test ./...` all green
- [x] 6.2 Re-ingest (`go run ./cmd/ingest`) to backfill existing rows; spot-check the two reported jobs in the DB now hold sanitized HTML of expected length
