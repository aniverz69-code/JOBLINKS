## MODIFIED Requirements

### Requirement: Jobs needing enrichment are tracked in a durable outbox queue

The system SHALL maintain an `enrichment_outbox` table holding one entry per
`(job_id, target_version)` that needs enriching. The entry SHALL reference the job by
id and SHALL NOT duplicate the job's source fields. The system SHALL provide an
idempotent enqueue that adds entries for **open** jobs (`closed_at IS NULL`) whose
`enriched_at IS NULL` or whose `enrichment_version` is below the current schema version
(`enrich.Version`); a closed job SHALL NOT be enqueued, and re-enqueuing an
already-queued `(job_id, target_version)` SHALL NOT create a duplicate.

The ingest write path SHALL additionally enqueue a job into the outbox in the **same
transaction** as the job's upsert, gated on the same condition (`enriched_at IS NULL`
or `enrichment_version` below the current version), so that a newly ingested job is
queued for enrichment atomically with its write while an already-enriched job is not
re-queued.

#### Scenario: Pending jobs are enqueued

- **WHEN** the enqueue runs and an open job has `enriched_at = NULL`
- **THEN** an outbox entry for that job at the current `target_version` exists

#### Scenario: Stale-version jobs are enqueued

- **WHEN** an open job's `enrichment_version` is below the current `enrich.Version`
- **THEN** an outbox entry for that job at the current `target_version` exists

#### Scenario: Closed jobs are not enqueued

- **WHEN** the enqueue runs and a job has `closed_at IS NOT NULL` and `enriched_at = NULL`
- **THEN** no outbox entry is created for that job

#### Scenario: Enqueue is idempotent

- **WHEN** the enqueue runs twice without the job being enriched in between
- **THEN** the job has exactly one outbox entry for that `target_version`

#### Scenario: Ingest enqueues a new job transactionally

- **WHEN** the ingest write path upserts a job whose `enriched_at IS NULL`
- **THEN** an outbox entry for that job at the current `target_version` is created in
  the same transaction as the upsert

#### Scenario: Ingest does not re-queue an already-enriched job

- **WHEN** the ingest write path re-ingests a job already enriched to the current
  `enrich.Version`
- **THEN** no new outbox entry is created for that job

### Requirement: Queue entries are claimed safely under concurrency

The system SHALL claim a bounded batch of outbox entries that are not dead-lettered,
not currently leased, and whose job is **open** (`closed_at IS NULL`), using
`FOR UPDATE SKIP LOCKED`, stamping `claimed_at` on each claimed entry. The claim SHALL
order candidates by job freshness — `COALESCE(posted_at, created_at) DESC, id DESC` —
so the newest open postings are enriched first; a job without a source post date SHALL
rank by its ingest time (`created_at`) rather than always sorting last, so undated jobs
are not starved while dated jobs keep arriving. Concurrent claimers SHALL receive
disjoint entries. An entry whose `claimed_at` is older than the lease duration SHALL
become claimable again, so a crashed or stalled worker's entries are reclaimed without a
separate process. An outbox entry whose job has since been closed SHALL NOT be claimed.

#### Scenario: Concurrent workers get disjoint entries

- **WHEN** two enrichment runs claim a batch at the same time
- **THEN** no outbox entry is handed to both runs

#### Scenario: Fresher open jobs are claimed first

- **WHEN** the outbox holds entries for two open jobs with different `posted_at`
- **THEN** a claim returns the entry for the job with the later `posted_at` before the
  one with the earlier `posted_at`

#### Scenario: Undated jobs rank by ingest time, not last

- **WHEN** the outbox holds an entry for an old dated job and one for a recently
  ingested job with no `posted_at`
- **THEN** a claim returns the undated-but-recent job's entry before the old dated one

#### Scenario: Entries for closed jobs are not claimed

- **WHEN** an outbox entry references a job with `closed_at IS NOT NULL`
- **THEN** it is not returned by a claim

#### Scenario: A stalled claim is reclaimed after the lease

- **WHEN** an entry was claimed but its `claimed_at` is older than the lease duration
- **THEN** a subsequent claim is allowed to pick it up again

#### Scenario: Dead-lettered entries are not claimed

- **WHEN** an entry has been dead-lettered (`failed_at` set)
- **THEN** it is not returned by a claim

### Requirement: A batch command runs the enrichment process

The system SHALL provide a standalone command (`cmd/enrich`) that connects to the
database, enqueues pending jobs, then repeatedly claims a wave of outbox entries and
drains it, enriching and writing back each entry, until no claimable entry remains. The
command SHALL process each claim wave **concurrently** across a configurable number of
workers (`ENRICH_CONCURRENCY`, default 4), and SHALL size each claim wave to the
configured concurrency so that the time an entry stays leased before processing remains
well under the lease duration. The command SHALL report how many entries were enriched,
failed, and dead-lettered. A failure on one entry SHALL NOT abort the run.

#### Scenario: A run reports its outcome

- **WHEN** `cmd/enrich` processes a wave with some enrichable and some failing entries
- **THEN** it writes the enrichable ones, advances the failing ones' attempts, and
  exits reporting the enriched / failed / dead-lettered counts

#### Scenario: A wave is drained concurrently

- **WHEN** a claim wave of multiple entries is drained with concurrency greater than one
- **THEN** entries in the wave are processed in parallel and the reported counts equal
  the sum of each entry's outcome

#### Scenario: One failing entry does not abort the run

- **WHEN** enriching a single entry returns an error (e.g. an LLM call fails)
- **THEN** that entry is recorded as a failed attempt and the run proceeds to the
  remaining entries
