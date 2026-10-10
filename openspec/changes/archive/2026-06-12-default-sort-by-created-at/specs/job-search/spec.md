# job-search Specification (delta)

## ADDED Requirements

### Requirement: Default ordering is newest-added first

A search request with no query text and no valid `sort` parameter SHALL return
jobs ordered by the time they entered the catalogue (`created_at`), newest
first. A request with query text and no `sort` SHALL keep relevance order. An
explicit valid `sort` parameter SHALL always take precedence. `created_at`
SHALL be a sortable attribute of the index and an accepted `sort` value. The
DB-backed jobs list SHALL use the same newest-added-first ordering.

#### Scenario: Browsing without a query shows newest additions first

- **WHEN** the search endpoint is called with empty `q` and no `sort`
- **THEN** results are ordered `created_at` descending

#### Scenario: A text query keeps relevance order

- **WHEN** the search endpoint is called with `q=golang` and no `sort`
- **THEN** results are in relevance order (no sort directive)

#### Scenario: Explicit sort wins

- **WHEN** the search endpoint is called with `sort=posted_at&order=asc`
- **THEN** results are ordered by `posted_at` ascending regardless of `q`
