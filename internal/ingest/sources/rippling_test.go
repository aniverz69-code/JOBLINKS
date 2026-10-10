package sources

import (
	"context"
	"strings"
	"testing"
)

func TestRipplingProvider(t *testing.T) {
	if got := NewRippling(nil).Provider(); got != "rippling" {
		t.Errorf("Provider() = %q, want %q", got, "rippling")
	}
}

func ripplingDetail(uuid, name, role string) string {
	return `{
		"uuid": "` + uuid + `",
		"name": "` + name + `",
		"url": "https://ats.rippling.com/acme/jobs/` + uuid + `",
		"createdOn": "2026-03-11T09:37:20.990000-07:00",
		"description": {
			"company": "<p>About Acme boilerplate.</p>",
			"role": "` + role + `"
		}
	}`
}

func TestRipplingFetchListsAndFetchesDetail(t *testing.T) {
	// Detail routes must precede the list route: the list URL ends in /jobs and the
	// detail URLs are /jobs/<uuid>, so the more specific matches come first.
	fake := (&routedHTTP{}).
		route("/jobs/U1", ripplingDetail("U1", "Backend Engineer", "<h3>Role</h3><p>Build the backend.</p>")).
		route("/jobs/U2", ripplingDetail("U2", "Remote SRE", "<p>Keep things up.</p>")).
		route("/board/acme/jobs", `[
			{"uuid": "U1", "name": "Backend Engineer", "url": "https://ats.rippling.com/acme/jobs/U1", "workLocation": {"label": "London, United Kingdom"}},
			{"uuid": "U2", "name": "Remote SRE", "url": "https://ats.rippling.com/acme/jobs/U2", "workLocation": {"label": "Remote"}}
		]`)

	jobs, err := NewRippling(fake).Fetch(context.Background(), CompanyEntry{
		Company: "Acme", Provider: "rippling", Board: "acme",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2", len(jobs))
	}

	byID := map[string]Job{}
	for _, j := range jobs {
		byID[j.ExternalID] = j
	}
	j, ok := byID["U1"]
	if !ok {
		t.Fatal("job U1 missing")
	}
	if j.Title != "Backend Engineer" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.URL != "https://ats.rippling.com/acme/jobs/U1" {
		t.Errorf("URL = %q", j.URL)
	}
	if j.Location != "London, United Kingdom" {
		t.Errorf("Location = %q, want the workLocation label", j.Location)
	}
	if !strings.Contains(j.Description, "Build the backend.") {
		t.Errorf("Description missing the role body, got %q", j.Description)
	}
	if strings.Contains(j.Description, "boilerplate") {
		t.Errorf("Description should exclude the company boilerplate, got %q", j.Description)
	}
	if j.Remote {
		t.Error("Remote = true for a London role, want false")
	}
	if j.PostedAt == nil || j.PostedAt.UTC().Year() != 2026 {
		t.Errorf("PostedAt = %v, want parsed createdOn (2026)", j.PostedAt)
	}
	if !byID["U2"].Remote {
		t.Error("U2 workLocation=Remote should infer Remote = true")
	}
}

func TestRipplingFetchSkipsFailedDetail(t *testing.T) {
	// U2 has no detail route -> its detail fetch errors and the posting is skipped,
	// but U1 still comes through.
	fake := (&routedHTTP{}).
		route("/jobs/U1", ripplingDetail("U1", "Engineer", "<p>Work.</p>")).
		route("/board/acme/jobs", `[
			{"uuid": "U1", "name": "Engineer", "url": "https://ats.rippling.com/acme/jobs/U1", "workLocation": {"label": "Berlin"}},
			{"uuid": "U2", "name": "Broken", "url": "https://ats.rippling.com/acme/jobs/U2", "workLocation": {"label": "Berlin"}}
		]`)

	jobs, err := NewRippling(fake).Fetch(context.Background(), CompanyEntry{
		Company: "Acme", Provider: "rippling", Board: "acme",
	})
	if err != nil {
		t.Fatalf("Fetch should not abort the board on one failed detail: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ExternalID != "U1" {
		t.Fatalf("want only U1 to survive, got %d jobs", len(jobs))
	}
}
