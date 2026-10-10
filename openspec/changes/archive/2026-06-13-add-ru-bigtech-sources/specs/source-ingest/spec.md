## MODIFIED Requirements

### Requirement: Boards to crawl are configured in a file

The system SHALL read the set of boards to crawl from configuration at ingest startup,
each entry naming a `company`, a `provider`, and — for a provider that has a board/tenant
concept — a `board`. A configured entry whose `provider` has no registered adapter SHALL
cause the ingest command to fail fast at startup rather than silently skip the board. An
entry whose `board` is empty SHALL fail fast at startup **unless** its provider is a
single-company provider that declares it needs no board (a `boardless` provider), in which
case the empty `board` SHALL be accepted.

#### Scenario: Configured boards are loaded

- **WHEN** the configuration lists a board with `company`, `provider`, and `board`
- **THEN** the ingest run includes that board, dispatched to the named provider

#### Scenario: Unknown provider fails fast

- **WHEN** the configuration names a `provider` with no registered adapter
- **THEN** the ingest command exits with an error naming the unknown provider and
  ingests nothing

#### Scenario: Empty board fails fast for a board-based provider

- **WHEN** the configuration has an entry with an empty `board` whose provider has a
  board concept (e.g. `greenhouse`)
- **THEN** the ingest command exits with an error naming the company with the empty board

#### Scenario: Empty board is accepted for a boardless provider

- **WHEN** the configuration has an entry with an empty `board` whose provider is a
  single-company `boardless` provider (e.g. `ozon`)
- **THEN** validation accepts the entry and the ingest run includes that board

## ADDED Requirements

### Requirement: A boardless provider may omit its board id

A single-company source adapter SHALL be able to declare itself `boardless` when its API
serves exactly one company and has no per-tenant board id, so its configured entries MAY
omit `board`. A provider that has a board or tenant concept SHALL NOT be boardless and
SHALL continue to require a non-empty `board`; this includes `yandex`, whose `board`
selects host and language.

#### Scenario: Boardless adapter is dispatched without a board

- **WHEN** a `boardless` provider's entry is crawled with an empty `board`
- **THEN** the adapter fetches that company's postings without requiring a board id and
  yields the normalized job shape

#### Scenario: Yandex requires its board

- **WHEN** a `yandex` entry is configured with an empty `board`
- **THEN** validation fails fast, because `yandex` selects host and language (`ru`/`com`)
  by its `board` and is not boardless

### Requirement: An adapter may send a per-request custom header

The shared HTTP client SHALL allow an adapter to attach custom request headers to an
individual request, in addition to the client's standard `User-Agent` and `Accept`,
without changing the behavior of requests that send no custom header. This SHALL support
sources that gate an otherwise-public JSON API behind a non-secret header (for example
MTS's public `x-api-key`), and the standard `User-Agent`/`Accept` and the retry/backoff
behavior SHALL be unchanged.

#### Scenario: Custom header is sent when provided

- **WHEN** an adapter issues a request with a custom header (e.g. `x-api-key`)
- **THEN** the outgoing request carries that header alongside the standard headers

#### Scenario: Existing adapters are unaffected

- **WHEN** an existing adapter issues a request through the unchanged `GetJSON`/`PostJSON`
  entry points
- **THEN** the request is sent with the standard headers and no custom header, exactly as
  before

### Requirement: An adapter may derive the description from source HTML

An adapter SHALL be permitted to obtain a posting body by extracting it from the source's
server-rendered HTML when the platform exposes no machine-readable description in its API,
and SHALL yield it as sanitized HTML, consistent with every other adapter's description
contract. Such extraction SHALL be isolated to that adapter, and an extraction failure for
one posting SHALL drop only that posting rather than abort the board.

#### Scenario: VK description is extracted from the vacancy page

- **WHEN** a `vk` board is crawled and VK's API carries no description field
- **THEN** the adapter fetches the vacancy page, extracts the description sections from its
  HTML, and yields a sanitized HTML description in the normalized job shape

#### Scenario: A failed extraction drops only that posting

- **WHEN** one VK posting's page cannot be fetched or its body cannot be extracted
- **THEN** that posting is skipped and every other posting is still yielded, without
  aborting the board

### Requirement: Russian bigtech single-company providers are registered

The system SHALL register adapters for the single-company Russian career APIs `yandex`,
`ozon`, `rwb`, `sber`, `alfabank`, `lamoda`, `kuper`, `aviasales`, `dodo`, `domclick`,
`mtslink`, `tbank`, `mts`, and `vk`, so each can be listed in the boards
configuration. Each adapter SHALL yield the normalized job shape (at least title, url,
location, remote flag, description, and the platform's native posting id) with the
`description` as sanitized HTML (or sanitized text for an API that publishes plain text)
assembled from the platform's authoritative field(s). An adapter whose list endpoint omits
the description SHALL fetch each posting's detail with bounded concurrency rather than yield
an empty body, and a single failed detail SHALL drop only that posting rather than abort
the board. All providers except `yandex` SHALL be `boardless`; `yandex` SHALL select host
and language (`ru`/`com`) from its `board`.

#### Scenario: Cursor-paginated board is fully crawled

- **WHEN** a `yandex` board is crawled and its list endpoint paginates by cursor
- **THEN** the adapter follows the cursor until exhausted, skips postings that redirect out
  or are hiring events, fetches each remaining posting's detail for the description, and
  yields each as the normalized job shape with `external_id` set to the posting's native id

#### Scenario: Page-paginated board is fully crawled

- **WHEN** an `ozon` board is crawled and its list endpoint paginates by page number
- **THEN** the adapter walks every page to the reported total, keeps only externally-listed
  vacancies, fetches each posting's detail for the description, and yields each as the
  normalized job shape

#### Scenario: Offset-paginated board with inline body needs no detail call

- **WHEN** a `sber` or `alfabank` board is crawled and the list endpoint carries the full
  body inline
- **THEN** the adapter walks the offset window to the reported total and yields each posting
  directly with a sanitized description, issuing no per-posting detail request

#### Scenario: Single-request board is crawled in one call

- **WHEN** an `aviasales`, `dodo`, `domclick`, or `mtslink` board is crawled and the list
  endpoint returns all postings in one response
- **THEN** the adapter fetches the list once and, where the body is not inline, fetches each
  posting's detail with bounded concurrency, yielding each as the normalized job shape

#### Scenario: POST-transport board is crawled

- **WHEN** a `tbank` board is crawled and its list and detail endpoints are POST requests
- **THEN** the adapter posts the paginated list request until the source reports completion,
  posts each posting's detail request to assemble the description, and yields each as the
  normalized job shape

#### Scenario: Header-gated board is crawled

- **WHEN** an `mts` board is crawled and its API requires a non-secret `x-api-key` header
- **THEN** the adapter obtains the public key, sends it on each request, walks the offset
  window to the reported total, fetches each posting's detail, and yields each as the
  normalized job shape

#### Scenario: A board with no open postings yields no jobs without error

- **WHEN** any of these providers' endpoints returns an empty posting list for a configured
  board
- **THEN** the adapter yields zero jobs and returns no error, so the board is simply skipped
