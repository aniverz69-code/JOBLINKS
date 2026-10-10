package normalize

import (
	"regexp"
	"strings"
)

// Some ATS boards jam raw coordinates into the free-text location field
// (e.g. "Gaston, SC 29053 | 33.8316 | -81.1126"), which then surface in the UI
// and in the JobPosting addressLocality. This matches a trailing run of one or
// more pipe-delimited numeric segments — the coordinate tail — while leaving a
// legitimate "City | Country" split (no numeric tail) intact.
var coordTail = regexp.MustCompile(`\s*(\|\s*-?\d+(\.\d+)?\s*)+$`)

// CleanLocation strips a source's coordinate tail from a free-text location and
// trims surrounding whitespace. It never touches a location that has no such
// tail, so plain places and "City | Country" splits pass through unchanged.
func CleanLocation(s string) string {
	return strings.TrimSpace(coordTail.ReplaceAllString(s, ""))
}
