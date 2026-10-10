-- name: ObserveLearnedDomain :one
-- Record one confident job-mail sighting for a sender domain, returning its running
-- count. The classifier calls this whenever it confidently labels an email as
-- application mail, so a recurring unknown ATS domain accrues hits toward promotion.
INSERT INTO learned_ats_domains (domain, confident_hits)
VALUES ($1, 1)
ON CONFLICT (domain) DO UPDATE
  SET confident_hits = learned_ats_domains.confident_hits + 1,
      last_seen_at = now()
RETURNING confident_hits;

-- name: PromotedDomains :many
-- Domains whose confident-hit count has reached the promotion threshold; the sync
-- worker unions these into the Gmail search query.
SELECT domain
FROM learned_ats_domains
WHERE confident_hits >= sqlc.arg(threshold)::int
ORDER BY domain;
