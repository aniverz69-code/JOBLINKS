//go:build integration

// Integration tests for the telegram_posts queue semantics — idempotent insert,
// claim/lease, completion, and dead-lettering — which are SQL behavior and can only
// be verified against a real Postgres. Run with: go test -tags=integration ./internal/platform/db/
// Requires Docker (testcontainers spins up a throwaway Postgres with the migrations).
package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func truncateTelegramPosts(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		"TRUNCATE telegram_posts"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func insertPost(t *testing.T, q *Queries, channel string, msgID int64, extracted bool) {
	t.Helper()
	var extractedAt pgtype.Timestamptz
	if extracted {
		extractedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}
	_, err := q.InsertTelegramPost(context.Background(), InsertTelegramPostParams{
		Channel: channel,
		MsgID:   msgID,
		Text:    "We are hiring a Go engineer",
		// links is JSONB NOT NULL; the real caller (postStore.Insert) always passes
		// a valid array, defaulting to "[]" for a linkless post — mirror that here so
		// a nil []byte is not sent as SQL NULL.
		Links:       []byte("[]"),
		PostedAt:    pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		ExtractedAt: extractedAt,
	})
	if err != nil {
		t.Fatalf("insert post %s/%d: %v", channel, msgID, err)
	}
}

func TestTelegramPostsQueue(t *testing.T) {
	pool := startPostgres(t)
	q := New(pool)
	ctx := context.Background()

	t.Run("insert is idempotent and never resets a stored post", func(t *testing.T) {
		truncateTelegramPosts(t, pool)
		insertPost(t, q, "hrlunapark", 392, false)

		// Mark it done, then re-insert (a re-crawl seeing the same post).
		if err := q.MarkTelegramPostExtracted(ctx, MarkTelegramPostExtractedParams{
			Channel: "hrlunapark", MsgID: 392,
		}); err != nil {
			t.Fatal(err)
		}
		insertPost(t, q, "hrlunapark", 392, false)

		var n int
		var done bool
		if err := pool.QueryRow(ctx,
			"SELECT count(*), bool_and(extracted_at IS NOT NULL) FROM telegram_posts").Scan(&n, &done); err != nil {
			t.Fatal(err)
		}
		if n != 1 || !done {
			t.Errorf("rows=%d done=%v, want 1 row still marked extracted", n, done)
		}
	})

	t.Run("prefiltered post is stored done and is not claimable", func(t *testing.T) {
		truncateTelegramPosts(t, pool)
		insertPost(t, q, "boards", 1, true) // prefilter: not a vacancy

		claimed, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 3600, BatchSize: 10})
		if err != nil || len(claimed) != 0 {
			t.Errorf("claim: rows=%d err=%v, want 0 (prefiltered post is done)", len(claimed), err)
		}
	})

	t.Run("claim leases posts so concurrent claims are disjoint, oldest first", func(t *testing.T) {
		truncateTelegramPosts(t, pool)
		insertPost(t, q, "a", 1, false)
		insertPost(t, q, "b", 2, false)

		first, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 3600, BatchSize: 1})
		if err != nil || len(first) != 1 {
			t.Fatalf("first claim: rows=%d err=%v, want 1", len(first), err)
		}
		second, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 3600, BatchSize: 10})
		if err != nil || len(second) != 1 {
			t.Fatalf("second claim: rows=%d err=%v, want 1", len(second), err)
		}
		if first[0].Channel == second[0].Channel && first[0].MsgID == second[0].MsgID {
			t.Errorf("both claims returned %s/%d — not disjoint", first[0].Channel, first[0].MsgID)
		}
		third, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 3600, BatchSize: 10})
		if err != nil || len(third) != 0 {
			t.Errorf("third claim: rows=%d, want 0 (all leased)", len(third))
		}
	})

	t.Run("a stale lease is reclaimable", func(t *testing.T) {
		truncateTelegramPosts(t, pool)
		insertPost(t, q, "stale", 1, false)

		if c, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 3600, BatchSize: 10}); err != nil || len(c) != 1 {
			t.Fatalf("claim: rows=%d err=%v, want 1", len(c), err)
		}
		if c, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 3600, BatchSize: 10}); err != nil || len(c) != 0 {
			t.Fatalf("re-claim within lease: rows=%d, want 0", len(c))
		}
		if c, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 0, BatchSize: 10}); err != nil || len(c) != 1 {
			t.Errorf("re-claim with expired lease: rows=%d err=%v, want 1", len(c), err)
		}
	})

	t.Run("extracted post is never claimed again", func(t *testing.T) {
		truncateTelegramPosts(t, pool)
		insertPost(t, q, "done", 1, false)

		if c, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 3600, BatchSize: 10}); err != nil || len(c) != 1 {
			t.Fatalf("claim: rows=%d err=%v, want 1", len(c), err)
		}
		if err := q.MarkTelegramPostExtracted(ctx, MarkTelegramPostExtractedParams{Channel: "done", MsgID: 1}); err != nil {
			t.Fatal(err)
		}
		if c, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 0, BatchSize: 10}); err != nil || len(c) != 0 {
			t.Errorf("claim after extract: rows=%d, want 0", len(c))
		}
	})

	t.Run("attempts reaching max dead-letters the post", func(t *testing.T) {
		truncateTelegramPosts(t, pool)
		insertPost(t, q, "dead", 1, false)

		if c, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 3600, BatchSize: 10}); err != nil || len(c) != 1 {
			t.Fatalf("claim: rows=%d err=%v, want 1", len(c), err)
		}

		first, err := q.RecordTelegramPostFailure(ctx, RecordTelegramPostFailureParams{
			LastError: "boom", MaxAttempts: 2, Channel: "dead", MsgID: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if first.Attempts != 1 || first.FailedAt.Valid {
			t.Errorf("after 1st failure: attempts=%d failed=%v, want 1/not-dead", first.Attempts, first.FailedAt.Valid)
		}
		second, err := q.RecordTelegramPostFailure(ctx, RecordTelegramPostFailureParams{
			LastError: "boom", MaxAttempts: 2, Channel: "dead", MsgID: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if second.Attempts != 2 || !second.FailedAt.Valid {
			t.Errorf("after 2nd failure: attempts=%d failed=%v, want 2/dead-lettered", second.Attempts, second.FailedAt.Valid)
		}
		if c, err := q.ClaimTelegramPosts(ctx, ClaimTelegramPostsParams{LeaseSeconds: 0, BatchSize: 10}); err != nil || len(c) != 0 {
			t.Errorf("claim after dead-letter: rows=%d, want 0", len(c))
		}
	})
}
