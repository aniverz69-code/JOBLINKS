package jobview

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/platform/db"
)

// The authorship audit columns (created_by/updated_by) are internal — FromRow must not
// carry them onto the public wire shape, the same way user_jobs omits user_id.
func TestFromRow_OmitsAuthorshipAudit(t *testing.T) {
	view, err := FromRow(db.Job{
		ID:         1,
		Title:      "Dev",
		PublicSlug: "dev-1",
		CreatedBy:  pgtype.Int8{Int64: 5, Valid: true},
		UpdatedBy:  pgtype.Int8{Int64: 6, Valid: true},
	})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}
	b, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if s := string(b); strings.Contains(s, "created_by") || strings.Contains(s, "updated_by") {
		t.Errorf("wire shape leaks authorship audit: %s", s)
	}
	// created_by is set → the posting is flagged manually added (the provenance signal).
	if !view.ManuallyAdded {
		t.Error("ManuallyAdded = false, want true when created_by is set")
	}
}

// An automated-source job (created_by NULL) is not flagged manually added.
func TestFromRow_ManuallyAddedFalseForAutomated(t *testing.T) {
	view, err := FromRow(db.Job{ID: 2, Title: "Dev", PublicSlug: "dev-2", Source: "greenhouse"})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}
	if view.ManuallyAdded {
		t.Error("ManuallyAdded = true, want false for an automated source (no created_by)")
	}
}
