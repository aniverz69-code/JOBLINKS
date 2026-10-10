package sources

import (
	"context"
	"strings"
	"testing"
)

func TestDomclickProvider(t *testing.T) {
	if got := NewDomclick(nil).Provider(); got != "domclick" {
		t.Errorf("Provider() = %q, want %q", got, "domclick")
	}
}

func TestDomclickIsBoardless(t *testing.T) {
	if _, ok := NewDomclick(nil).(boardless); !ok {
		t.Error("domclick should implement the boardless marker")
	}
}

func TestDomclickFetchListsAndMapsDetailBySlug(t *testing.T) {
	// Detail routes are registered first: routedHTTP matches the first route whose
	// substring is in the URL, and the detail URL (/api/v1/vacancy/detail/<slug>/)
	// also contains the list match (/api/v1/vacancy/), so the more specific detail
	// match must come first.
	fake := (&routedHTTP{}).
		route("/detail/marketing-84/", `{"success":true,"result":{"vacancycontent":{
			"branded_description":"<style>.x{}</style><p>Branded marketing copy.</p>",
			"description":"<p>Plain description.</p>"
		}}}`).
		route("/detail/backend-85/", `{"success":true,"result":{"vacancycontent":{
			"branded_description":"",
			"description":"<p>Plain backend copy.</p><script>alert(1)</script>"
		}}}`).
		route("/api/v1/vacancy/", `{"success":true,"result":[
			{"id":84,"slug":"marketing-84","title":"Marketing Lead","area":{"name":"Москва"},"vacancycontent":{"work_format":["ON_SITE","HYBRID"]}},
			{"id":85,"slug":"backend-85","title":"Backend Engineer","area":{"name":"Удалённо"},"vacancycontent":{"work_format":["REMOTE"]}}
		]}`)

	jobs, err := NewDomclick(fake).Fetch(context.Background(), CompanyEntry{Company: "DomClick", Provider: "domclick"})
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

	j, ok := byID["marketing-84"]
	if !ok {
		t.Fatal("vacancy marketing-84 missing (ExternalID should be the slug)")
	}
	if j.Title != "Marketing Lead" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.Company != "DomClick" {
		t.Errorf("Company = %q, want DomClick", j.Company)
	}
	if want := "https://career.domclick.ru/vacancy/marketing-84"; j.URL != want {
		t.Errorf("URL = %q, want %q", j.URL, want)
	}
	if j.Location != "Москва" {
		t.Errorf("Location = %q, want area.name", j.Location)
	}
	if !strings.Contains(j.Description, "Branded marketing copy.") {
		t.Errorf("Description should use branded_description, got %q", j.Description)
	}
	if strings.Contains(j.Description, "<style>") {
		t.Errorf("Description not sanitized: %q", j.Description)
	}
	if j.Remote {
		t.Error("84 Remote = true, want false (ON_SITE/HYBRID)")
	}
	if j.PostedAt != nil {
		t.Errorf("PostedAt = %v, want nil", j.PostedAt)
	}

	b := byID["backend-85"]
	if !b.Remote {
		t.Error("85 Remote = false, want true (work_format REMOTE)")
	}
	if !strings.Contains(b.Description, "Plain backend copy.") {
		t.Errorf("Description should fall back to description when branded empty, got %q", b.Description)
	}
	if strings.Contains(b.Description, "<script>") {
		t.Errorf("Description not sanitized: %q", b.Description)
	}
}

func TestDomclickFetchSkipsFailedDetail(t *testing.T) {
	fake := (&routedHTTP{}).
		route("/detail/kept-1/", `{"success":true,"result":{"vacancycontent":{"branded_description":"<p>ok</p>","description":""}}}`).
		route("/api/v1/vacancy/", `{"success":true,"result":[
			{"id":1,"slug":"kept-1","title":"Kept","area":{"name":"Москва"},"vacancycontent":{"work_format":["ON_SITE"]}},
			{"id":2,"slug":"broken-2","title":"Broken","area":{"name":"Москва"},"vacancycontent":{"work_format":["ON_SITE"]}}
		]}`)

	jobs, err := NewDomclick(fake).Fetch(context.Background(), CompanyEntry{Company: "DomClick", Provider: "domclick"})
	if err != nil {
		t.Fatalf("Fetch should not abort on one failed detail: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ExternalID != "kept-1" {
		t.Fatalf("want only kept-1 to survive, got %d jobs", len(jobs))
	}
}

func TestDomclickEmptyListYieldsNoJobsNoError(t *testing.T) {
	fake := (&routedHTTP{}).route("/api/v1/vacancy/", `{"success":true,"result":[]}`)

	jobs, err := NewDomclick(fake).Fetch(context.Background(), CompanyEntry{Company: "DomClick", Provider: "domclick"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("got %d jobs, want 0", len(jobs))
	}
}
