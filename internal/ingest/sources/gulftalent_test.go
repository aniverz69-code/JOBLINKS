package sources

import (
	"context"
	"strings"
	"testing"
	"time"
)

// gtSitemapIndexXML renders the sitemap index: a <sitemapindex> of shard <loc>s.
func gtSitemapIndexXML(shardLocs ...string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><sitemapindex>`)
	for _, l := range shardLocs {
		b.WriteString(`<sitemap><loc>` + l + `</loc></sitemap>`)
	}
	b.WriteString(`</sitemapindex>`)
	return b.String()
}

// gtShardXML renders a shard: a <urlset> of page <loc>s.
func gtShardXML(urls ...string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><urlset>`)
	for _, u := range urls {
		b.WriteString(`<url><loc>` + u + `</loc></url>`)
	}
	b.WriteString(`</urlset>`)
	return b.String()
}

// gtDetailHTML renders a GulfTalent job page: one schema.org JobPosting. datePosted is RFC3339,
// jobLocation is a single Place with a reliable address, and company of "" omits the org.
func gtDetailHTML(title, datePosted, company, locality, country string) string {
	org := ""
	if company != "" {
		org = `"hiringOrganization":{"@type":"Organization","name":"` + company + `"},`
	}
	return `<html><head><script type="application/ld+json">
{"@context":"https://schema.org","@type":"JobPosting",
"title":"` + title + `",
"description":"<p>Do the thing.<\/p><script>alert(1)<\/script>",
"identifier":{"@type":"PropertyValue","name":"` + company + `","value":"604168"},
"datePosted":"` + datePosted + `",
"employmentType":["FULL_TIME"],
` + org + `
"jobLocation":{"@type":"Place","address":{"@type":"PostalAddress","addressLocality":"` + locality + `","addressCountry":"` + country + `"}}}
</script></head><body>page</body></html>`
}

func TestGulfTalentProvider(t *testing.T) {
	if got := NewGulfTalent(nil).Provider(); got != "gulftalent" {
		t.Errorf("Provider() = %q, want %q", got, "gulftalent")
	}
}

// gulftalent is blocked by both its TLS fingerprint AND the direct datacenter IP, so it must
// route through the proxied fingerprint transport — not the standard proxied client (which
// keeps Go's default fingerprint and still 403s) and not the direct fingerprint (blocked by IP).
func TestGulfTalentIsFingerprintProxied(t *testing.T) {
	if _, ok := proxiedFingerprintProviders["gulftalent"]; !ok {
		t.Error("gulftalent must be in proxiedFingerprintProviders (needs Chrome fingerprint AND the proxy IP)")
	}
	if _, ok := proxiedProviders["gulftalent"]; ok {
		t.Error("gulftalent must NOT be in proxiedProviders — the standard proxied client keeps Go's fingerprint, which its edge 403s")
	}
	// bayt shares the fingerprint transport but is behind a Cloudflare JS challenge no
	// fingerprint/IP passes, so it must not be routed here (proxying it would not help).
	if _, ok := proxiedFingerprintProviders["bayt"]; ok {
		t.Error("bayt must NOT be in proxiedFingerprintProviders — its Cloudflare JS challenge is unaffected by the proxy")
	}
}

func TestGulfTalentJobID(t *testing.T) {
	cases := map[string]string{
		"https://www.gulftalent.com/uae/jobs/maintenance-specialist-604168": "604168",
		"https://www.gulftalent.com/saudi-arabia/jobs/dev-42":               "42",
		"https://www.gulftalent.com/uae/jobs/tracked-99?ref=feed":           "99", // query stripped
		"https://www.gulftalent.com/jobs/category/accounting":               "",
		"https://www.gulftalent.com/companies/some-group-careers":           "",
	}
	for loc, want := range cases {
		if got := gulftalentJobID(loc); got != want {
			t.Errorf("gulftalentJobID(%q) = %q, want %q", loc, got, want)
		}
	}
}

func TestGulfTalentRegisteredBoardlessAggregatorFacet(t *testing.T) {
	src, ok := All(nil)["gulftalent"]
	if !ok {
		t.Fatal("gulftalent not registered in sources.All")
	}
	if _, isBoardless := src.(boardless); !isBoardless {
		t.Error("gulftalent should be boardless (one global sitemap crawl, no board id)")
	}
	if _, isAggregator := src.(aggregator); !isAggregator {
		t.Error("gulftalent should be an aggregator (many companies per crawl)")
	}
	found := false
	for _, p := range FilterableProviders() {
		if p == "gulftalent" {
			found = true
		}
	}
	if !found {
		t.Error("gulftalent should appear in the source facet")
	}
}

func TestGulfTalentFetchSitemapThenDetailAndMaps(t *testing.T) {
	jobURL := "https://www.gulftalent.com/uae/jobs/maintenance-specialist-604168"
	fake := (&routedHTTP{}).
		route("/sitemap.xml", gtSitemapIndexXML("https://www.gulftalent.com/sitemaps/sitemap_jx000.xml")).
		route("sitemap_jx000.xml", gtShardXML(jobURL)).
		route("maintenance-specialist-604168", gtDetailHTML(
			"Maintenance Specialist", "2026-07-03T00:00:00+00:00", "RTC-1 Employment Services", "Dubai", "UAE"))

	jobs, err := NewGulfTalent(fake).Fetch(context.Background(), CompanyEntry{
		Company: "GulfTalent", Provider: "gulftalent",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("got %d jobs, want 1", len(jobs))
	}
	j := jobs[0]
	if j.ExternalID != "604168" {
		t.Errorf("ExternalID = %q, want 604168", j.ExternalID)
	}
	if j.URL != jobURL {
		t.Errorf("URL = %q, want %q", j.URL, jobURL)
	}
	if j.Title != "Maintenance Specialist" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.Company != "RTC-1 Employment Services" {
		t.Errorf("Company = %q, want the posting's hiringOrganization", j.Company)
	}
	if j.Location != "Dubai, UAE" {
		t.Errorf("Location = %q, want %q", j.Location, "Dubai, UAE")
	}
	if strings.Contains(j.Description, "<script>") || !strings.Contains(j.Description, "<p>Do the thing.</p>") {
		t.Errorf("Description not sanitized/assembled: %q", j.Description)
	}
	if j.PostedAt == nil || !j.PostedAt.Equal(time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("PostedAt = %v, want 2026-07-03", j.PostedAt)
	}
}

func TestGulfTalentFollowsOnlyJobPostingShards(t *testing.T) {
	// The index lists a job-posting shard (jx) and a category-listing shard (jl). Only jx pages
	// are real postings; a jl page must never be fetched or mapped even though its URL looks
	// job-shaped.
	jxJob := "https://www.gulftalent.com/uae/jobs/real-role-1000"
	jlJob := "https://www.gulftalent.com/uae/jobs/should-not-appear-2000"
	fake := (&routedHTTP{}).
		route("/sitemap.xml", gtSitemapIndexXML(
			"https://www.gulftalent.com/sitemaps/sitemap_jx000.xml",
			"https://www.gulftalent.com/sitemaps/sitemap_jl000.xml",
		)).
		route("sitemap_jx000.xml", gtShardXML(jxJob)).
		route("sitemap_jl000.xml", gtShardXML(jlJob)).
		route("real-role-1000", gtDetailHTML("Real", "2026-07-01T00:00:00+00:00", "Acme", "Dubai", "UAE")).
		route("should-not-appear-2000", gtDetailHTML("Nope", "2026-07-01T00:00:00+00:00", "Ghost", "Dubai", "UAE"))

	jobs, err := NewGulfTalent(fake).Fetch(context.Background(), CompanyEntry{Company: "GulfTalent"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ExternalID != "1000" {
		t.Fatalf("only the jx-shard posting should be crawled, got %+v", jobs)
	}
}

func TestGulfTalentDropsPostingWithNoCompany(t *testing.T) {
	jobURL := "https://www.gulftalent.com/uae/jobs/ghost-777"
	fake := (&routedHTTP{}).
		route("/sitemap.xml", gtSitemapIndexXML("https://www.gulftalent.com/sitemaps/sitemap_jx000.xml")).
		route("sitemap_jx000.xml", gtShardXML(jobURL)).
		route("ghost-777", gtDetailHTML("Ghost", "2026-07-01T00:00:00+00:00", "", "Dubai", "UAE"))

	jobs, err := NewGulfTalent(fake).Fetch(context.Background(), CompanyEntry{Company: "GulfTalent"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("company-less posting should be dropped, got %d", len(jobs))
	}
}

func TestGulfTalentDropsDetailWithNoJobPosting(t *testing.T) {
	// A detail page without a JobPosting block is dropped, not errored — one bad posting must
	// not abort the crawl (only an unparseable index does).
	jobURL := "https://www.gulftalent.com/uae/jobs/broken-555"
	fake := (&routedHTTP{}).
		route("/sitemap.xml", gtSitemapIndexXML("https://www.gulftalent.com/sitemaps/sitemap_jx000.xml")).
		route("sitemap_jx000.xml", gtShardXML(jobURL)).
		route("broken-555", `<html><body>no ld+json</body></html>`)

	jobs, err := NewGulfTalent(fake).Fetch(context.Background(), CompanyEntry{Company: "GulfTalent"})
	if err != nil {
		t.Fatalf("a JobPosting-less detail must not error the crawl: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("JobPosting-less detail should be dropped, got %d", len(jobs))
	}
}

func TestGulfTalentUnparseableIndexErrors(t *testing.T) {
	fake := (&routedHTTP{}).route("/sitemap.xml", "not xml at all <<<")
	_, err := NewGulfTalent(fake).Fetch(context.Background(), CompanyEntry{Company: "GulfTalent"})
	if err == nil {
		t.Fatal("an unparseable sitemap index must error, not silently yield no jobs")
	}
}
