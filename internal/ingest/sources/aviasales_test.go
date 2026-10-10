package sources

import (
	"context"
	"strings"
	"testing"
)

func TestAviasalesProvider(t *testing.T) {
	if got := NewAviasales(nil).Provider(); got != "aviasales" {
		t.Errorf("Provider() = %q, want %q", got, "aviasales")
	}
}

func TestAviasalesIsBoardless(t *testing.T) {
	if _, ok := NewAviasales(nil).(boardless); !ok {
		t.Error("aviasales should implement the boardless marker")
	}
}

func TestAviasalesFetchListsAndMapsDetail(t *testing.T) {
	// One remote vacancy (kept, flagged remote) and one office vacancy; both come from
	// the single list, each body from its own loader detail.
	fake := (&routedHTTP{}).
		route("/api/vacancies", `[
			{"id":111,"position":"Support Specialist","workPlace":"Удалённо","team":{"name":"B2B"},"tags":["Support"]},
			{"id":222,"position":"Office Manager","workPlace":"Москва","team":{"name":"Ops"},"tags":[]}
		]`).
		route("/about/vacancies/111?", `{"vacancy":{
			"description":"<p>About Aviasales.</p><script>alert(1)</script>",
			"todo":"<ul><li>help clients</li></ul>",
			"requirements":"<ul><li>2 years support</li></ul>",
			"conditions":"<ul><li>flexible schedule</li></ul>",
			"workPlace":"Удалённо"
		}}`).
		route("/about/vacancies/222?", `{"vacancy":{
			"description":"<p>Manage the office.</p>",
			"todo":"<ul><li>keep things tidy</li></ul>",
			"requirements":"",
			"conditions":"",
			"workPlace":"Москва"
		}}`)

	jobs, err := NewAviasales(fake).Fetch(context.Background(), CompanyEntry{Company: "Aviasales", Provider: "aviasales"})
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

	j, ok := byID["111"]
	if !ok {
		t.Fatal("vacancy 111 missing")
	}
	if j.Title != "Support Specialist" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.Company != "Aviasales" {
		t.Errorf("Company = %q, want Aviasales", j.Company)
	}
	if want := "https://www.aviasales.ru/about/vacancies/111"; j.URL != want {
		t.Errorf("URL = %q, want %q", j.URL, want)
	}
	if j.Location != "Удалённо" {
		t.Errorf("Location = %q, want workPlace", j.Location)
	}
	for _, want := range []string{"About Aviasales.", "help clients", "2 years support", "flexible schedule"} {
		if !strings.Contains(j.Description, want) {
			t.Errorf("Description missing %q, got %q", want, j.Description)
		}
	}
	if strings.Contains(j.Description, "<script>") {
		t.Errorf("Description not sanitized: %q", j.Description)
	}
	if !j.Remote {
		t.Error("Remote = false, want true (Удалённо)")
	}
	if j.PostedAt != nil {
		t.Errorf("PostedAt = %v, want nil", j.PostedAt)
	}

	if byID["222"].Remote {
		t.Error("222 Remote = true, want false (Москва)")
	}
}

func TestAviasalesFetchSkipsFailedDetail(t *testing.T) {
	// 222 has no detail route -> its detail fetch errors and the posting is skipped,
	// but 111 still comes through.
	fake := (&routedHTTP{}).
		route("/api/vacancies", `[
			{"id":111,"position":"Kept","workPlace":"Москва","team":{"name":"X"},"tags":[]},
			{"id":222,"position":"Broken","workPlace":"Москва","team":{"name":"Y"},"tags":[]}
		]`).
		route("/about/vacancies/111?", `{"vacancy":{"description":"<p>ok</p>","workPlace":"Москва"}}`)

	jobs, err := NewAviasales(fake).Fetch(context.Background(), CompanyEntry{Company: "Aviasales", Provider: "aviasales"})
	if err != nil {
		t.Fatalf("Fetch should not abort on one failed detail: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ExternalID != "111" {
		t.Fatalf("want only 111 to survive, got %d jobs", len(jobs))
	}
}

func TestAviasalesEmptyListYieldsNoJobsNoError(t *testing.T) {
	fake := (&routedHTTP{}).route("/api/vacancies", `[]`)

	jobs, err := NewAviasales(fake).Fetch(context.Background(), CompanyEntry{Company: "Aviasales", Provider: "aviasales"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("got %d jobs, want 0", len(jobs))
	}
}
