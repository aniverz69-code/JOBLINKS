## ADDED Requirements

### Requirement: The LLM provider extracts the posting's stated requirements

The `Provider` abstraction SHALL additionally extract the job's stated
requirements as a `requirements` list, each entry carrying the requirement's
text and a priority of `required` or `preferred`, based solely on the job
posting — the provider SHALL NOT compare requirements against any candidate,
since no candidate is available at enrichment time. Unlike the served scalar
and multi-value enum fields, the request schema SHALL NOT constrain each
entry's `priority` to its two allowed values (the schema mechanism used
elsewhere in this capability constrains only top-level fields and array
elements that are themselves scalars, not a nested property of an array-of-
objects field); `priority` correctness SHALL instead rely on the prompt
instruction and on sanitizing the response into the controlled vocabulary
after receipt, the same fallback the discovery facets already rely on when
schema-level constraint is not available.

#### Scenario: Requirements are extracted from the posting alone

- **WHEN** the provider is given a job whose description states "Must have 5+ years of Go experience. Kubernetes experience is a plus."
- **THEN** it returns an `Enrichment` whose `requirements` includes an entry with `text` describing the Go experience requirement and `priority=required`, and an entry with `text` describing the Kubernetes experience and `priority=preferred`

#### Scenario: A posting with no explicit requirements yields an empty list

- **WHEN** the provider is given a job whose description states no explicit requirements
- **THEN** the returned `Enrichment` has an empty `requirements` list rather than an invented one

#### Scenario: A malformed priority is not rejected outright

- **WHEN** the model returns a `requirements` entry whose `priority` is not schema-constrained and does not exactly match `required` or `preferred`
- **THEN** the enrichment write-back path sanitizes the entry's `priority` into the controlled vocabulary rather than dead-lettering the job over it
