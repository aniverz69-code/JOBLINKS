package normalize

import "testing"

func TestCleanLocation(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "strips a trailing coordinate pair a source jammed into the location",
			in:   "Gaston, SC 29053 | 33.831598379 | -81.112575386",
			want: "Gaston, SC 29053",
		},
		{
			name: "keeps a legitimate city | country split (no numeric tail)",
			in:   "Berlin | Germany",
			want: "Berlin | Germany",
		},
		{
			name: "strips a single trailing numeric segment",
			in:   "New York | 40.7128",
			want: "New York",
		},
		{
			name: "leaves a plain location untouched",
			in:   "Amsterdam",
			want: "Amsterdam",
		},
		{
			name: "trims surrounding whitespace",
			in:   "  Remote  ",
			want: "Remote",
		},
		{
			name: "empty stays empty",
			in:   "",
			want: "",
		},
		{
			name: "does not strip a number that is part of the place text",
			in:   "District 9",
			want: "District 9",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CleanLocation(tc.in); got != tc.want {
				t.Errorf("CleanLocation(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
