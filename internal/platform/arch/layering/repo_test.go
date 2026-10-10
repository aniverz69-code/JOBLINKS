package layering_test

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/strelov1/freehire/internal/platform/arch/layering"
)

const modulePath = "github.com/strelov1/freehire"

// buildTags must name every constraint the repo uses. `go list` without them reports the
// untagged graph, and 222 files sit behind //go:build integration — many of them
// in-package tests, whose imports constrain the package itself. A guard that cannot see
// them passes over exactly the case AGENTS.md warns about: green in every local command
// except the tagged vet, then a failure in CI, which runs the tagged suite.
//
// Naming them is sound rather than merely convenient: the repo has no negated constraint
// and no legacy `// +build` line, so the tagged build is a strict superset.
const buildTags = "integration,llmlive"

// loadGraph reads the real import graph of internal/ once per test binary. Test imports
// are included deliberately: a _test.go file can create a cross-layer dependency the
// production build never reveals.
var loadGraph = sync.OnceValues(func() (map[string][]string, error) {
	const sep = "\x1f"
	format := "{{.ImportPath}}" + sep +
		"{{join .Imports \",\"}}" + sep +
		"{{join .TestImports \",\"}}" + sep +
		"{{join .XTestImports \",\"}}"

	out, err := exec.CommandContext(context.Background(), "go", "list",
		"-tags", buildTags, "-f", format, modulePath+"/internal/...").Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, errors.New(err.Error() + "\n" + string(exit.Stderr))
		}
		return nil, err
	}

	graph := make(map[string][]string)
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Split(line, sep)
		if len(parts) != 4 {
			return nil, errors.New("unparseable go list line: " + line)
		}
		var imports []string
		for _, group := range parts[1:] {
			for _, imp := range strings.Split(group, ",") {
				if imp != "" {
					imports = append(imports, imp)
				}
			}
		}
		graph[parts[0]] = imports
	}
	if len(graph) == 0 {
		return nil, errors.New("go list returned no packages; the guard would pass vacuously")
	}
	return graph, nil
})

func repoGraph(t *testing.T) map[string][]string {
	t.Helper()
	graph, err := loadGraph()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	return graph
}

// relName is a package's name relative to its block: internal/dict/normalize is
// "normalize", internal/identity/auth/oauth is "auth/oauth". A package directly under
// internal/ has no block to strip and yields its whole path, which is a violation
// TestEveryPackageSitsWhereAssignmentSaysItShould reports by name.
func relName(importPath string) string {
	rest, ok := strings.CutPrefix(importPath, modulePath+"/internal/")
	if !ok {
		return ""
	}
	if head, tail, found := strings.Cut(rest, "/"); found {
		if _, isBlock := layering.Layers[head]; isBlock {
			return tail
		}
	}
	return rest
}

// atExpectedLocation reports whether a package sits at exactly the path Assignment gives
// it. Stricter than "is under some block", and deliberately: a package dropped in the
// WRONG block is still under a block, and the layering check catches that only if the
// mistake happens to cross a layer. Two blocks on adjacent layers would hide it entirely.
func atExpectedLocation(importPath string) bool {
	name := relName(importPath)
	block, ok := layering.Assignment[name]
	if !ok {
		return false
	}
	return importPath == modulePath+"/internal/"+block+"/"+name
}

func TestEveryPackageSitsWhereAssignmentSaysItShould(t *testing.T) {
	var misplaced []string
	for importPath := range repoGraph(t) {
		if !atExpectedLocation(importPath) {
			misplaced = append(misplaced, importPath)
		}
	}
	slices.Sort(misplaced)
	for _, importPath := range misplaced {
		name := relName(importPath)
		t.Errorf("%s is not where it belongs: Assignment puts %q in block %q",
			importPath, name, layering.Assignment[name])
	}
}

func TestEveryPackageInTheRepoIsAssignedToABlock(t *testing.T) {
	var missing []string
	for importPath := range repoGraph(t) {
		name := relName(importPath)
		if _, ok := layering.Assignment[name]; !ok {
			missing = append(missing, name)
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		t.Errorf("packages exist in the repo but are in no block: %v", missing)
	}
}

func TestBlockTableNamesNoPackageThatDoesNotExist(t *testing.T) {
	present := make(map[string]bool)
	for importPath := range repoGraph(t) {
		present[relName(importPath)] = true
	}
	var stale []string
	for name := range layering.Assignment {
		if !present[name] {
			stale = append(stale, name)
		}
	}
	slices.Sort(stale)
	if len(stale) > 0 {
		t.Errorf("block table names packages that do not exist: %v", stale)
	}
}

// The repo-wide assertion: every import in internal/, production and test alike, respects
// the layering. depguard enforces the same rule one import line at a time; this reports
// the whole graph in one pass, which is what makes a broad violation diagnosable.
func TestRepoRespectsTheLayering(t *testing.T) {
	violations := layering.Check(repoGraph(t), layering.Layers)
	if len(violations) == 0 {
		return
	}
	var b strings.Builder
	for _, v := range violations {
		b.WriteString("\n  " + v.String())
	}
	t.Errorf("%d layering violations:%s", len(violations), b.String())
}
