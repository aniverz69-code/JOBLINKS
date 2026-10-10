package gmailsync

import (
	"context"
	"testing"
)

func TestLearnableDomain(t *testing.T) {
	cases := []struct {
		name   string
		from   string
		want   string
		wantOK bool
	}{
		{"niche ATS relay is learnable", "TeamEx <notifications@updates.teamex.io>", "updates.teamex.io", true},
		{"personal company recruiter is learnable", "Brad <brad@innovarerec.com>", "innovarerec.com", true},
		{"free-mail is not learnable", "Friend <friend@gmail.com>", "", false},
		{"free-mail ru is not learnable", "x@yandex.ru", "", false},
		{"already-known ATS is not learnable", "no-reply@ashbyhq.com", "", false},
		{"known ATS subdomain is not learnable", "x@candidates.workablemail.com", "", false},
		{"unparseable is not learnable", "not-an-address", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := LearnableDomain(c.from)
			if got != c.want || ok != c.wantOK {
				t.Fatalf("LearnableDomain(%q) = (%q, %v), want (%q, %v)", c.from, got, ok, c.want, c.wantOK)
			}
		})
	}
}

// fakeLearnStore counts Observe calls per domain and promotes at the threshold.
type fakeLearnStore struct{ counts map[string]int }

func (f *fakeLearnStore) Observe(_ context.Context, domain string) (int, error) {
	f.counts[domain]++
	return f.counts[domain], nil
}
func (f *fakeLearnStore) Promoted(context.Context) ([]string, error) {
	var out []string
	for d, n := range f.counts {
		if n >= PromoteThreshold {
			out = append(out, d)
		}
	}
	return out, nil
}

func TestRecordJobMailLearnsThenPromotes(t *testing.T) {
	store := &fakeLearnStore{counts: map[string]int{}}
	ctx := context.Background()

	// Free-mail and known-ATS senders are ignored — no domain is recorded.
	_ = RecordJobMail(ctx, store, "friend@gmail.com")
	_ = RecordJobMail(ctx, store, "no-reply@ashbyhq.com")
	if len(store.counts) != 0 {
		t.Fatalf("non-learnable senders were recorded: %v", store.counts)
	}

	// A niche domain promotes only after PromoteThreshold confident sightings.
	for i := 0; i < PromoteThreshold; i++ {
		if err := RecordJobMail(ctx, store, "hi@updates.teamex.io"); err != nil {
			t.Fatalf("RecordJobMail: %v", err)
		}
	}
	promoted, _ := store.Promoted(ctx)
	if len(promoted) != 1 || promoted[0] != "updates.teamex.io" {
		t.Errorf("promoted = %v, want [updates.teamex.io]", promoted)
	}
}
