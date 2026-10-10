package sources

import (
	"context"
	"strings"
	"testing"
)

func TestMtslinkProvider(t *testing.T) {
	if got := NewMtslink(nil).Provider(); got != "mtslink" {
		t.Errorf("Provider() = %q, want %q", got, "mtslink")
	}
}

func TestMtslinkIsBoardless(t *testing.T) {
	if _, ok := NewMtslink(nil).(boardless); !ok {
		t.Error("mtslink should implement the boardless marker")
	}
}

func TestMtslinkFetchFiltersOpenAndMapsDetail(t *testing.T) {
	// 111 is OPEN (kept), 999 is CLOSED (dropped before any detail fetch).
	fake := (&routedHTTP{}).
		route("/api/huntflow/vacancies", `[
			{"id":111,"position":"Golang Developer","state":"OPEN","workFormat":"Удаленный"},
			{"id":999,"position":"Closed Role","state":"CLOSED","workFormat":"Удаленный"}
		]`).
		route("/api/huntflow/vacancy/111", `{
			"body":"<p>Build the platform.</p><script>alert(1)</script>",
			"requirements":"<ul><li>Go</li></ul>",
			"conditions":"<ul><li>remote</li></ul>",
			"workFormat":"Удаленный",
			"created":{"date":"2026-05-08 13:21:25.000000","timezone_type":1,"timezone":"+03:00"}
		}`)

	jobs, err := NewMtslink(fake).Fetch(context.Background(), CompanyEntry{Company: "MTS Link", Provider: "mtslink"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1 (only the OPEN vacancy)", len(jobs))
	}

	j := jobs[0]
	if j.ExternalID != "111" {
		t.Errorf("ExternalID = %q, want 111", j.ExternalID)
	}
	if j.Title != "Golang Developer" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.Company != "MTS Link" {
		t.Errorf("Company = %q, want MTS Link", j.Company)
	}
	if want := "https://mts-link.ru/vacancies/111/"; j.URL != want {
		t.Errorf("URL = %q, want %q", j.URL, want)
	}
	if j.Location != "" {
		t.Errorf("Location = %q, want empty (no city field)", j.Location)
	}
	for _, want := range []string{"Build the platform.", "Go", "remote"} {
		if !strings.Contains(j.Description, want) {
			t.Errorf("Description missing %q, got %q", want, j.Description)
		}
	}
	if strings.Contains(j.Description, "<script>") {
		t.Errorf("Description not sanitized: %q", j.Description)
	}
	if !j.Remote {
		t.Error("Remote = false, want true (Удаленный)")
	}
	if j.PostedAt == nil || j.PostedAt.Year() != 2026 || j.PostedAt.Month() != 5 || j.PostedAt.Day() != 8 {
		t.Errorf("PostedAt = %v, want parsed 2026-05-08 created.date", j.PostedAt)
	}
}

func TestMtslinkFetchSkipsFailedDetail(t *testing.T) {
	fake := (&routedHTTP{}).
		route("/api/huntflow/vacancies", `[
			{"id":111,"position":"Kept","state":"OPEN","workFormat":"Офис"},
			{"id":222,"position":"Broken","state":"OPEN","workFormat":"Офис"}
		]`).
		route("/api/huntflow/vacancy/111", `{"body":"<p>ok</p>","requirements":"","conditions":"","workFormat":"Офис","created":{"date":""}}`)

	jobs, err := NewMtslink(fake).Fetch(context.Background(), CompanyEntry{Company: "MTS Link", Provider: "mtslink"})
	if err != nil {
		t.Fatalf("Fetch should not abort on one failed detail: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ExternalID != "111" {
		t.Fatalf("want only 111 to survive, got %d jobs", len(jobs))
	}
	if jobs[0].Remote {
		t.Error("Remote = true, want false (Офис)")
	}
	if jobs[0].PostedAt != nil {
		t.Errorf("PostedAt = %v, want nil for empty created.date", jobs[0].PostedAt)
	}
}

func TestMtslinkEmptyListYieldsNoJobsNoError(t *testing.T) {
	fake := (&routedHTTP{}).route("/api/huntflow/vacancies", `[]`)

	jobs, err := NewMtslink(fake).Fetch(context.Background(), CompanyEntry{Company: "MTS Link", Provider: "mtslink"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("got %d jobs, want 0", len(jobs))
	}
}
