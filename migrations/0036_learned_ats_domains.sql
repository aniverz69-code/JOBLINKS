-- Self-learning ATS sender-domain cache. A domain earns a place in the Gmail sync
-- allowlist after its mail is confidently classified as job-application mail
-- enough times (see gmailsync.PromoteThreshold), so coverage grows from real
-- classifications instead of hand-curation-by-anecdote. The Worker unions the
-- promoted domains into BuildQuery; the classifier calls Observe on confident
-- job-mail. Global, not per-user: a domain that is an ATS for one seeker is for all.
-- Queries: internal/db/queries/learned_domains.sql (ObserveLearnedDomain upsert,
-- PromotedDomains read). The promotion threshold mirrors gmailsync.PromoteThreshold.
CREATE TABLE learned_ats_domains (
    domain         TEXT PRIMARY KEY,
    confident_hits INTEGER     NOT NULL DEFAULT 0,
    first_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
