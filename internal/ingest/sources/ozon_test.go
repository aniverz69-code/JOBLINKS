package sources

import (
	"context"
	"strings"
	"testing"
)

// ozonListPage builds one page of the Ozon v2/vacancy list response with the given
// meta and inline item fragments.
func ozonListPage(page, totalPages int, items ...string) string {
	return `{"items":[` + strings.Join(items, ",") +
		`],"meta":{"page":` + itoa(page) + `,"perPage":50,"totalItems":3,"totalPages":` + itoa(totalPages) + `}}`
}

// itoa is a tiny local int->string so the test fixtures stay inline without importing strconv
// at call sites; the adapter under test uses strconv directly.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func ozonDetail(hhID int, name, city, slug, publishedAt, descr string, workFormat ...string) string {
	wf := make([]string, len(workFormat))
	for i, w := range workFormat {
		wf[i] = `"` + w + `"`
	}
	return `{"name":"` + name + `","city":"` + city + `","hhId":` + itoa(hhID) +
		`,"descr":"` + descr + `","slug":"` + slug + `","publishedAt":"` + publishedAt +
		`","workFormat":[` + strings.Join(wf, ",") + `]}`
}

func TestOzonProvider(t *testing.T) {
	if got := NewOzon(nil).Provider(); got != "ozon" {
		t.Errorf("Provider() = %q, want %q", got, "ozon")
	}
}

func TestOzonFetchPaginatesFiltersAndMapsDetail(t *testing.T) {
	// Page 1 carries an external_vacancy (kept) and an internal one (dropped by the filter);
	// page 2 carries a second external_vacancy. totalPages=2 drives the pagination.
	fake := (&routedHTTP{}).
		route("page=1", ozonListPage(1, 2,
			`{"hhId":111,"title":"Data Engineer (list)","workFormat":[" Удалённо"],"city":"Москва","vacancyType":"external_vacancy"}`,
			`{"hhId":999,"title":"Internal Only","workFormat":["Гибрид"],"city":"Москва","vacancyType":"internal_vacancy"}`,
		)).
		route("page=2", ozonListPage(2, 2,
			`{"hhId":222,"title":"Backend (list)","workFormat":["Гибрид"],"city":"Москва","vacancyType":"external_vacancy"}`,
		)).
		route("/vacancy/111", ozonDetail(111, "Data Engineer", "Москва", "data-engineer-111",
			"2026-05-26 12:50:08", "<p>Build pipelines.</p><script>alert(1)</script>", " Удалённо", "Гибрид")).
		route("/vacancy/222", ozonDetail(222, "Backend Engineer", "Санкт-Петербург", "backend-222",
			"2026-04-01 09:00:00", "<p>Write services.</p>", "Гибрид"))

	jobs, err := NewOzon(fake).Fetch(context.Background(), CompanyEntry{Company: "Ozon", Provider: "ozon"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2 (one external per page, internal filtered out)", len(jobs))
	}

	byID := map[string]Job{}
	for _, j := range jobs {
		byID[j.ExternalID] = j
	}

	j, ok := byID["111"]
	if !ok {
		t.Fatal("vacancy 111 missing")
	}
	if j.Title != "Data Engineer" {
		t.Errorf("Title = %q, want detail name", j.Title)
	}
	if j.Company != "Ozon" {
		t.Errorf("Company = %q, want Ozon", j.Company)
	}
	if want := "https://career.ozon.ru/vacancy/data-engineer-111/"; j.URL != want {
		t.Errorf("URL = %q, want %q", j.URL, want)
	}
	if j.Location != "Москва" {
		t.Errorf("Location = %q, want detail city", j.Location)
	}
	if !strings.Contains(j.Description, "Build pipelines.") || strings.Contains(j.Description, "<script>") {
		t.Errorf("Description not sanitized/assembled: %q", j.Description)
	}
	if !j.Remote {
		t.Error("Remote = false, want true (nbsp-normalized 'Удалённо' matches isRemote)")
	}
	if j.PostedAt == nil || j.PostedAt.Year() != 2026 || j.PostedAt.Month() != 5 {
		t.Errorf("PostedAt = %v, want parsed 2026-05 publishedAt", j.PostedAt)
	}

	if byID["222"].Remote {
		t.Error("222 Remote = true, want false (Гибрид only)")
	}
}

func TestOzonFetchSkipsFailedDetail(t *testing.T) {
	// 222 has no detail route -> its detail fetch errors and the posting is skipped,
	// but 111 still comes through.
	fake := (&routedHTTP{}).
		route("page=1", ozonListPage(1, 1,
			`{"hhId":111,"title":"Kept","workFormat":["Гибрид"],"city":"Москва","vacancyType":"external_vacancy"}`,
			`{"hhId":222,"title":"Broken","workFormat":["Гибрид"],"city":"Москва","vacancyType":"external_vacancy"}`,
		)).
		route("/vacancy/111", ozonDetail(111, "Kept", "Москва", "kept-111", "2026-04-01 09:00:00", "<p>ok</p>", "Гибрид"))

	jobs, err := NewOzon(fake).Fetch(context.Background(), CompanyEntry{Company: "Ozon", Provider: "ozon"})
	if err != nil {
		t.Fatalf("Fetch should not abort on one failed detail: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ExternalID != "111" {
		t.Fatalf("want only 111 to survive, got %d jobs", len(jobs))
	}
}

func TestOzonEmptyListYieldsNoJobsNoError(t *testing.T) {
	fake := (&routedHTTP{}).route("page=1", ozonListPage(1, 1))

	jobs, err := NewOzon(fake).Fetch(context.Background(), CompanyEntry{Company: "Ozon", Provider: "ozon"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("got %d jobs, want 0", len(jobs))
	}
}

func TestOzonIsBoardless(t *testing.T) {
	if _, ok := NewOzon(nil).(boardless); !ok {
		t.Error("ozon should implement the boardless marker")
	}
}
