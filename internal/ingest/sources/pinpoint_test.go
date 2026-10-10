package sources

import (
	"context"
	"strings"
	"testing"
)

func TestPinpointProvider(t *testing.T) {
	if got := NewPinpoint(nil).Provider(); got != "pinpoint" {
		t.Errorf("Provider() = %q, want %q", got, "pinpoint")
	}
}

func TestPinpointFetch(t *testing.T) {
	fake := &fakeHTTP{body: `{"data": [
		{
			"id": "217178",
			"title": "Pipeline Engineer",
			"url": "https://acme.pinpointhq.com/en/postings/abc",
			"description": "<div>Build pipelines.</div>",
			"key_responsibilities": "<ul><li>Own ingest</li></ul>",
			"skills_knowledge_expertise": "<ul><li>Go</li></ul>",
			"benefits": "<div>Health cover.</div>",
			"employment_type": "contract",
			"workplace_type": "onsite",
			"location": {"city": "Annapolis", "province": "Maryland", "name": "HQ"}
		},
		{
			"id": "217179",
			"title": "Remote Engineer",
			"url": "https://acme.pinpointhq.com/en/postings/def",
			"description": "<div>Work from anywhere.</div>",
			"workplace_type": "remote",
			"location": {"city": "Anywhere", "province": ""}
		}
	]}`}

	jobs, err := NewPinpoint(fake).Fetch(context.Background(), CompanyEntry{
		Company: "Acme", Provider: "pinpoint", Board: "acme",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !strings.Contains(fake.gotURL, "acme.pinpointhq.com") || !strings.Contains(fake.gotURL, "postings.json") {
		t.Errorf("requested URL %q should target the board postings.json", fake.gotURL)
	}
	if len(jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2", len(jobs))
	}

	j := jobs[0]
	if j.ExternalID != "217178" {
		t.Errorf("ExternalID = %q", j.ExternalID)
	}
	if j.Title != "Pipeline Engineer" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.URL != "https://acme.pinpointhq.com/en/postings/abc" {
		t.Errorf("URL = %q", j.URL)
	}
	if j.Location != "Annapolis, Maryland" {
		t.Errorf("Location = %q, want city/province joined", j.Location)
	}
	// Description assembles the body sections, sanitized.
	for _, want := range []string{"Build pipelines.", "Own ingest", "Go", "Health cover."} {
		if !strings.Contains(j.Description, want) {
			t.Errorf("Description missing %q, got %q", want, j.Description)
		}
	}
	if j.Remote {
		t.Error("Remote = true for workplace_type=onsite, want false")
	}
	if !jobs[1].Remote {
		t.Error("second job workplace_type=remote should set Remote = true")
	}
}
