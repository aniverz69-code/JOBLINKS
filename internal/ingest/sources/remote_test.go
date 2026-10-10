package sources

import "testing"

// TestIsRemote characterizes the location-based remote heuristic. It pins that the
// heuristic already flags non-breaking-space-bearing remote text on its own, so the
// former normalizeNBSP wrapper at the call sites was a no-op: isRemote substring-matches
// "remote"/"удал", neither of which contains a space, so an NBSP never affects the match.
func TestIsRemote(t *testing.T) {
	const nbsp = " "
	cases := map[string]bool{
		"Remote":              true,
		"Remote work":         true,
		"Удалённо":            true,
		"Удалённая работа":    true,
		nbsp + "Удалённо":     true, // leading NBSP (as Ozon emits) still matches
		"Remote" + nbsp + "x": true, // NBSP between tokens still matches
		"Москва":              false,
		"On-site":             false,
		"":                    false,
	}
	for in, want := range cases {
		if got := isRemote(in); got != want {
			t.Errorf("isRemote(%q) = %v, want %v", in, got, want)
		}
	}
}
