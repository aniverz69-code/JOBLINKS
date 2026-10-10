package sources

import (
	"context"
	"strings"
	"testing"
	"time"
)

// vkListPage builds one career/api/v2/vacancies page with the given next URL (empty → null)
// and inline result fragments.
func vkListPage(next string, results ...string) string {
	n := "null"
	if next != "" {
		n = `"` + next + `"`
	}
	return `{"count":2,"next":` + n + `,"previous":null,"results":[` + strings.Join(results, ",") + `]}`
}

func vkResultJSON(id int, title, town string, remote bool) string {
	r := "false"
	if remote {
		r = "true"
	}
	return `{"id":` + itoa(id) + `,"title":"` + title + `","town":{"id":1,"name":"` + town +
		`"},"remote":` + r + `}`
}

// vkDetailHTML wraps a description body in VK's schema.org JobPosting microdata, as the live
// vacancy page renders it (a single div.article[itemprop=description]).
func vkDetailHTML(body string) string {
	return `<html><body><div itemscope itemtype="http://schema.org/JobPosting">` +
		`<h1 itemprop="title">Ignored</h1>` +
		`<div class="article" itemprop="description">` + body + `</div>` +
		`</div></body></html>`
}

func TestVKProvider(t *testing.T) {
	if got := NewVK(nil).Provider(); got != "vk" {
		t.Errorf("Provider() = %q, want %q", got, "vk")
	}
}

func TestVKIsBoardless(t *testing.T) {
	if _, ok := NewVK(nil).(boardless); !ok {
		t.Error("vk should implement the boardless marker")
	}
}

func TestVKFetchPaginatesScrapesAndMaps(t *testing.T) {
	fake := (&routedHTTP{}).
		route("offset=0", vkListPage(
			"https://team.vk.company/career/api/v2/vacancies/?limit=50&offset=50",
			vkResultJSON(100, "Backend Engineer", "Москва", false),
		)).
		route("offset=50", vkListPage("",
			vkResultJSON(200, "Remote SRE", "Санкт-Петербург", true),
		)).
		route("/vacancy/100/", vkDetailHTML(`<p>Build services.</p><h3>Tasks</h3><ul><li>Code</li></ul><script>alert(1)</script>`)).
		route("/vacancy/200/", vkDetailHTML(`<p>Keep it running.</p>`))

	jobs, err := newVK(fake, 0).Fetch(context.Background(), CompanyEntry{Company: "VK", Provider: "vk"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2 (one per page via next)", len(jobs))
	}

	byID := map[string]Job{}
	for _, j := range jobs {
		byID[j.ExternalID] = j
	}

	j1, ok := byID["100"]
	if !ok {
		t.Fatal("vacancy 100 missing")
	}
	if j1.Title != "Backend Engineer" {
		t.Errorf("Title = %q", j1.Title)
	}
	if j1.Company != "VK" {
		t.Errorf("Company = %q, want VK", j1.Company)
	}
	if want := "https://team.vk.company/vacancy/100/"; j1.URL != want {
		t.Errorf("URL = %q, want %q", j1.URL, want)
	}
	if j1.Location != "Москва" {
		t.Errorf("Location = %q, want town.name", j1.Location)
	}
	if strings.Contains(j1.Description, "<script>") {
		t.Errorf("Description not sanitized: %q", j1.Description)
	}
	if !strings.Contains(j1.Description, "Build services.") || !strings.Contains(j1.Description, "Tasks") {
		t.Errorf("Description not scraped/assembled: %q", j1.Description)
	}
	if j1.Remote {
		t.Error("100 Remote = true, want false")
	}
	if j1.PostedAt != nil {
		t.Errorf("PostedAt = %v, want nil", j1.PostedAt)
	}

	if !byID["200"].Remote {
		t.Error("200 Remote = false, want true (remote bool)")
	}
}

func TestVKFailedDetailDropsOnlyThatPosting(t *testing.T) {
	// No route for /vacancy/200/ → GetHTML errors → that posting drops.
	fake := (&routedHTTP{}).
		route("offset=0", vkListPage("",
			vkResultJSON(100, "Kept", "Москва", false),
			vkResultJSON(200, "Dropped", "Москва", false),
		)).
		route("/vacancy/100/", vkDetailHTML(`<p>ok</p>`))

	jobs, err := newVK(fake, 0).Fetch(context.Background(), CompanyEntry{Company: "VK"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ExternalID != "100" {
		t.Fatalf("got %v, want only the kept posting", jobs)
	}
}

func TestVKKeepsPostingWhenDetailHasNoDescription(t *testing.T) {
	// VK's edge serves a challenge page (no JobPosting microdata) under rate limiting.
	// The posting must stay live from its list fields with an empty description rather
	// than being dropped — otherwise a rate-limit blip would slowly close the catalogue.
	fake := (&routedHTTP{}).
		route("offset=0", vkListPage("", vkResultJSON(100, "Engineer", "Москва", false))).
		route("/vacancy/100/", `<html><body>rate-limit challenge, no microdata</body></html>`)

	jobs, err := newVK(fake, 0).Fetch(context.Background(), CompanyEntry{Company: "VK"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ExternalID != "100" {
		t.Fatalf("want the posting kept, got %v", jobs)
	}
	if jobs[0].Description != "" {
		t.Errorf("Description = %q, want empty", jobs[0].Description)
	}
}

func TestVKPacingRespectsContextCancellation(t *testing.T) {
	// Detail fetches are paced to stay under VK's rate limit; the pace must be
	// context-aware, so a cancelled crawl returns promptly instead of sleeping.
	fake := (&routedHTTP{}).
		route("offset=0", vkListPage("",
			vkResultJSON(100, "A", "Москва", false),
			vkResultJSON(200, "B", "Москва", false),
		)).
		route("/vacancy/100/", vkDetailHTML(`<p>x</p>`)).
		route("/vacancy/200/", vkDetailHTML(`<p>y</p>`))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := newVK(fake, time.Hour).Fetch(ctx, CompanyEntry{Company: "VK"}); err == nil {
		t.Fatal("want a context error when cancelled during pacing, got nil")
	}
}

func TestVKEmptyListYieldsNoJobsNoError(t *testing.T) {
	fake := (&routedHTTP{}).route("offset=0", vkListPage(""))

	jobs, err := newVK(fake, 0).Fetch(context.Background(), CompanyEntry{Company: "VK"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("got %d jobs, want 0", len(jobs))
	}
}
