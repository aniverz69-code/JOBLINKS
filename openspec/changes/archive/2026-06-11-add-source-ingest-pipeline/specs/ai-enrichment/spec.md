## MODIFIED Requirements

### Requirement: Jobs needing enrichment are tracked in a durable outbox queue

The system SHALL maintain an `enrichment_outbox` table holding one entry per
`(job_id, target_version)` that needs enriching. The entry SHALL reference the job by
id and SHALL NOT duplicate the job's source fields. The system SHALL provide an
idempotent enqueue that adds entries for jobs whose `enriched_at IS NULL` or whose
`enrichment_version` is below the current schema version (`enrich.Version`);
re-enqueuing an already-queued `(job_id, target_version)` SHALL NOT create a duplicate.

The ingest write path SHALL additionally enqueue a job into the outbox in the **same
transaction** as the job's upsert, gated on the same condition (`enriched_at IS NULL`
or `enrichment_version` below the current version), so that a newly ingested job is
queued for enrichment atomically with its write while an already-enriched job is not
re-queued.

#### Scenario: Pending jobs are enqueued

- **WHEN** the enqueue runs and a job has `enriched_at = NULL`
- **THEN** an outbox entry for that job at the current `target_version` exists

#### Scenario: Stale-version jobs are enqueued

- **WHEN** a job's `enrichment_version` is below the current `enrich.Version`
- **THEN** an outbox entry for that job at the current `target_version` exists

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
