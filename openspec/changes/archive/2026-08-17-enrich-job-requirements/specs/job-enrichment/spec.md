## ADDED Requirements

### Requirement: Enrichment captures the posting's own stated requirements

The enrichment contract SHALL include a `requirements` field: an array of
objects, each with a `text` (the requirement as stated) and a `priority` of
either `required` or `preferred`. This list SHALL be job-only — derived
solely from the posting, with no comparison against any candidate — and is
distinct from any candidate-comparison requirement list produced elsewhere in
the system. The list SHALL be bounded to a maximum number of entries and each
`text` bounded to a maximum length, consistent with how other free-text
enrichment fields (e.g. `cities`) are bounded. A `priority` value outside
`required`/`preferred` SHALL be coerced into that vocabulary rather than
rejected, and an entry whose `text` is empty after bounding SHALL be dropped.

#### Scenario: A posting's requirements round-trip through the typed contract

- **WHEN** an `Enrichment` value with `requirements=[{text: "5+ years Go", priority: required}, {text: "Kubernetes", priority: preferred}]` is marshalled to JSON, stored, read back, and unmarshalled
- **THEN** the resulting value equals the original

#### Scenario: An oversized requirements list is bounded

- **WHEN** an enrichment payload's `requirements` list exceeds the maximum entry count, or an entry's `text` exceeds the maximum length
- **THEN** sanitizing the payload truncates the list to the maximum entry count and clips each `text` to the maximum length

#### Scenario: An unrecognized priority is coerced, not rejected

- **WHEN** a `requirements` entry's `priority` is not exactly `required` (e.g. a different casing, surrounding whitespace, or an unrelated value)
- **THEN** sanitizing the payload coerces that entry's `priority` to `required` when it case/whitespace-insensitively matches `required`, and to `preferred` otherwise

#### Scenario: No stated requirements yields an empty list, not a guess

- **WHEN** a job description states no explicit requirements
- **THEN** the returned `Enrichment` has an empty (or absent) `requirements` list rather than a fabricated one

### Requirement: The jobs read API exposes the posting's stated requirements

The `requirements` field SHALL be served as part of a job's `enrichment`
payload in the jobs read endpoints (`GET /api/v1/jobs`, `GET /api/v1/jobs/:id`,
and jobs nested under a company), the same way every other served enrichment
field is — with no suppression.

#### Scenario: Job detail includes the posting's requirements

- **WHEN** a client requests `GET /api/v1/jobs/:id` for a job whose enrichment includes a non-empty `requirements` list
- **THEN** the returned object's `enrichment.requirements` matches the stored list
