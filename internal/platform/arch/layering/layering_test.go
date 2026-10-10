package layering_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/strelov1/freehire/internal/platform/arch/layering"
)

// testLayers is a miniature stand-in for the real table: three blocks over three layers.
// Using fixture blocks rather than the real ones keeps these tests about the rule, so a
// later re-layering of the repo does not rewrite them.
var testLayers = map[string]int{
	"bottom": 1,
	"left":   2,
	"right":  2,
	"top":    3,
}

const mod = "example.com/m/internal/"

func kinds(vs []layering.Violation) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, string(v.Kind))
	}
	slices.Sort(out)
	return out
}

func TestDownwardImportIsAllowed(t *testing.T) {
	graph := map[string][]string{
		mod + "top/handler": {mod + "bottom/db"},
	}
	if got := layering.Check(graph, testLayers); len(got) != 0 {
		t.Fatalf("downward import must be allowed, got %v", got)
	}
}

func TestIntraBlockImportIsAllowed(t *testing.T) {
	graph := map[string][]string{
		mod + "left/pipeline": {mod + "left/sources"},
	}
	if got := layering.Check(graph, testLayers); len(got) != 0 {
		t.Fatalf("intra-block import must be allowed, got %v", got)
	}
}

func TestUpwardImportIsReported(t *testing.T) {
	graph := map[string][]string{
		mod + "bottom/db": {mod + "top/handler"},
	}
	got := layering.Check(graph, testLayers)
	if len(got) != 1 || got[0].Kind != layering.KindUpward {
		t.Fatalf("want one upward violation, got %v", got)
	}
	v := got[0]
	// The report has to carry both ends and both layers, or a failure is not diagnosable.
	if v.From != mod+"bottom/db" || v.To != mod+"top/handler" {
		t.Errorf("violation must name both packages, got %q -> %q", v.From, v.To)
	}
	if v.FromLayer != 1 || v.ToLayer != 3 {
		t.Errorf("violation must carry both layers, got %d -> %d", v.FromLayer, v.ToLayer)
	}
}

func TestSameLayerCrossBlockImportIsReported(t *testing.T) {
	// left and right share layer 2. Neither may see the other, or the order is partial
	// rather than total and "imports only strictly below" stops meaning anything.
	graph := map[string][]string{
		mod + "left/pipeline": {mod + "right/notify"},
	}
	got := layering.Check(graph, testLayers)
	if len(got) != 1 || got[0].Kind != layering.KindSameLayer {
		t.Fatalf("want one same-layer violation, got %v", got)
	}
}

func TestPackageDirectlyUnderInternalIsReported(t *testing.T) {
	graph := map[string][]string{
		mod + "orphan": {mod + "bottom/db"},
	}
	got := layering.Check(graph, testLayers)
	if len(got) != 1 || got[0].Kind != layering.KindUnassigned {
		t.Fatalf("want one unassigned violation, got %v", got)
	}
	if got[0].From != mod+"orphan" {
		t.Errorf("violation must name the unassigned package, got %q", got[0].From)
	}
}

func TestUnknownBlockIsReported(t *testing.T) {
	graph := map[string][]string{
		mod + "nosuchblock/thing": {mod + "bottom/db"},
	}
	got := layering.Check(graph, testLayers)
	if len(got) != 1 || got[0].Kind != layering.KindUnknownBlock {
		t.Fatalf("want one unknown-block violation, got %v", got)
	}
	if !strings.Contains(got[0].Block, "nosuchblock") {
		t.Errorf("violation must name the block, got %q", got[0].Block)
	}
}

// An import TO an unassigned or unknown-block package is a violation of that package's
// own membership, already reported when it is visited as a source. Reporting it again at
// every import site would bury the one actionable line under dozens of duplicates.
func TestBadTargetIsNotReportedTwice(t *testing.T) {
	graph := map[string][]string{
		mod + "top/handler": {mod + "orphan"},
		mod + "orphan":      {},
	}
	if got := kinds(layering.Check(graph, testLayers)); !slices.Equal(got, []string{"unassigned"}) {
		t.Fatalf("want exactly one unassigned violation, got %v", got)
	}
}

func TestNonInternalImportsAreIgnored(t *testing.T) {
	graph := map[string][]string{
		mod + "bottom/db": {"context", "github.com/jackc/pgx/v5", "example.com/m/cmd/server"},
	}
	if got := layering.Check(graph, testLayers); len(got) != 0 {
		t.Fatalf("only internal/ imports are governed, got %v", got)
	}
}

func TestEmptyGraphYieldsNoViolations(t *testing.T) {
	if got := layering.Check(map[string][]string{}, testLayers); len(got) != 0 {
		t.Fatalf("empty graph must be clean, got %v", got)
	}
}

// Not hypothetical: go list puts a package's own path in XTestImports for every
// `package foo_test` file, so every externally-tested package self-imports in this graph.
func TestSelfImportIsNotAViolation(t *testing.T) {
	graph := map[string][]string{
		mod + "bottom/db": {mod + "bottom/db"},
	}
	if got := layering.Check(graph, testLayers); len(got) != 0 {
		t.Fatalf("a package importing itself must be clean, got %v", got)
	}
}

// The same import arrives twice when a package appears in both .Imports and .TestImports.
func TestDuplicateImportIsReportedOnce(t *testing.T) {
	graph := map[string][]string{
		mod + "bottom/db": {mod + "top/handler", mod + "top/handler", mod + "top/handler"},
	}
	if got := layering.Check(graph, testLayers); len(got) != 1 {
		t.Fatalf("want one violation for a repeated import, got %d: %v", len(got), got)
	}
}

func TestViolationMessagesNameTheProblem(t *testing.T) {
	for _, tc := range []struct {
		name string
		v    layering.Violation
		want string
	}{
		{
			name: "unassigned",
			v:    layering.Violation{Kind: layering.KindUnassigned, From: "x/internal/orphan"},
			want: "directly under internal/",
		},
		{
			name: "unknown block",
			v:    layering.Violation{Kind: layering.KindUnknownBlock, From: "x/internal/q/r", Block: "q"},
			want: `unknown block "q"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.v.String(); !strings.Contains(got, tc.want) || !strings.Contains(got, tc.v.From) {
				t.Errorf("message %q must contain %q and the package name", got, tc.want)
			}
		})
	}
}

func TestViolationsAreDeterministicallyOrdered(t *testing.T) {
	graph := map[string][]string{
		mod + "bottom/z": {mod + "top/handler"},
		mod + "bottom/a": {mod + "top/handler"},
	}
	// Map iteration is random; a checker that reports in that order produces a diff-noisy
	// failure that is hard to compare between runs.
	first := layering.Check(graph, testLayers)
	for range 20 {
		if got := layering.Check(graph, testLayers); !slices.Equal(kinds(got), kinds(first)) ||
			got[0].From != first[0].From {
			t.Fatalf("output is not stable: %v vs %v", first, got)
		}
	}
	if first[0].From != mod+"bottom/a" {
		t.Errorf("want violations sorted by importing package, got %q first", first[0].From)
	}
}
