package licensecheck

import (
	"reflect"
	"strings"
	"testing"
)

func mustInput(t *testing.T, body string) Input {
	t.Helper()
	in, err := LoadInput(strings.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	return in
}

func findProblem(t *testing.T, r Report, kind, comp string) Problem {
	t.Helper()
	for _, p := range r.Problems {
		if p.Kind == kind && p.Component == comp {
			return p
		}
	}
	t.Fatalf("problem %s for %s not found in %+v", kind, comp, r.Problems)
	return Problem{}
}

const rulesMITGPL = `
	"rules": [
		{"license": "MIT", "required": ["LICENSE"], "allowedPairs": [["MIT", "GPL"]]},
		{"license": "GPL", "required": ["LICENSE"], "allowedPairs": []}
	]`

// The pair GPL->MIT is not allowed even though MIT->GPL is.
func TestEdgeDirectionMatters(t *testing.T) {
	body := `{
		"components": [
			{"id": "g", "license": "GPL", "dependsOn": ["m"], "materials": ["LICENSE"]},
			{"id": "m", "license": "MIT", "dependsOn": [], "materials": ["LICENSE"]}
		],` + rulesMITGPL + `,
		"roots": ["g"]
	}`
	reports := Check(mustInput(t, body))
	if len(reports) != 1 {
		t.Fatalf("want 1 report, got %d", len(reports))
	}
	p := findProblem(t, reports[0], "incompatible-license", "g")
	if p.Edge == nil || p.Edge.Parent != "g" || p.Edge.Child != "m" {
		t.Fatalf("unexpected edge: %+v", p.Edge)
	}
	if !reflect.DeepEqual(p.Path, []string{"g"}) {
		t.Fatalf("unexpected path: %v", p.Path)
	}
}

// Same-license pairs are not implicitly compatible.
func TestSameLicenseNotImplicitlyCompatible(t *testing.T) {
	body := `{
		"components": [
			{"id": "a", "license": "MIT", "dependsOn": ["b"], "materials": ["LICENSE"]},
			{"id": "b", "license": "MIT", "dependsOn": [], "materials": ["LICENSE"]}
		],` + rulesMITGPL + `,
		"roots": ["a"]
	}`
	reports := Check(mustInput(t, body))
	findProblem(t, reports[0], "incompatible-license", "a")
}

// Cycles must terminate, visit each node/edge once, and shortest paths
// must be used.
func TestCycleVisitedOnce(t *testing.T) {
	body := `{
		"components": [
			{"id": "a", "license": "MIT", "dependsOn": ["b"], "materials": ["LICENSE"]},
			{"id": "b", "license": "GPL", "dependsOn": ["a"], "materials": ["LICENSE"]}
		],` + rulesMITGPL + `,
		"roots": ["a"]
	}`
	reports := Check(mustInput(t, body))
	r := reports[0]
	if !reflect.DeepEqual(r.Reachable, []string{"a", "b"}) {
		t.Fatalf("unexpected reachable: %v", r.Reachable)
	}
	// a->b allowed; b->a (GPL->MIT) incompatible, edge checked once only.
	bad := 0
	for _, p := range r.Problems {
		if p.Kind == "incompatible-license" {
			bad++
		}
	}
	if bad != 1 {
		t.Fatalf("want exactly 1 incompatible edge, got %d (%+v)", bad, r.Problems)
	}
	p := findProblem(t, r, "incompatible-license", "b")
	if !reflect.DeepEqual(p.Path, []string{"a", "b"}) {
		t.Fatalf("path to b should be shortest a,b: %v", p.Path)
	}
}

// A shared component is reported independently for each root, and
// problems that are unreachable from a root must not show up there.
func TestSharedDependencyAndUnreachable(t *testing.T) {
	body := `{
		"components": [
			{"id": "r1", "license": "MIT", "dependsOn": ["s"], "materials": ["LICENSE"]},
			{"id": "r2", "license": "MIT", "dependsOn": ["s"], "materials": ["LICENSE"]},
			{"id": "s", "license": "GPL", "dependsOn": [], "materials": []},
			{"id": "u", "license": "MIT", "dependsOn": [], "materials": []}
		],` + rulesMITGPL + `,
		"roots": ["r1", "r2"]
	}`
	reports := Check(mustInput(t, body))
	if len(reports) != 2 || reports[0].Root != "r1" || reports[1].Root != "r2" {
		t.Fatalf("unexpected reports: %+v", reports)
	}
	for _, r := range reports {
		if !reflect.DeepEqual(r.Reachable, []string{"r" + r.Root[1:], "s"}) {
			t.Fatalf("root %s unexpected reachable: %v", r.Root, r.Reachable)
		}
		p := findProblem(t, r, "missing-materials", "s")
		if !reflect.DeepEqual(p.Materials, []string{"LICENSE"}) {
			t.Fatalf("unexpected missing materials: %v", p.Materials)
		}
		if !reflect.DeepEqual(p.Path, []string{r.Root, "s"}) {
			t.Fatalf("unexpected path: %v", p.Path)
		}
		for _, q := range r.Problems {
			if q.Component == "u" {
				t.Fatalf("unreachable u must not appear: %+v", q)
			}
		}
	}
}

// Unknown licenses are flagged and edges touching them are reported as
// unjudgable rather than incompatible.
func TestUnknownLicenseAndEdge(t *testing.T) {
	body := `{
		"components": [
			{"id": "a", "license": "MIT", "dependsOn": ["b", "c"], "materials": ["LICENSE"]},
			{"id": "b", "license": "WTF", "dependsOn": [], "materials": []},
			{"id": "c", "license": "GPL", "dependsOn": [], "materials": ["LICENSE"]}
		],` + rulesMITGPL + `,
		"roots": ["a"]
	}`
	r := Check(mustInput(t, body))[0]
	findProblem(t, r, "unknown-license", "b")
	findProblem(t, r, "unknown-license-edge", "a")
	for _, p := range r.Problems {
		if p.Kind == "incompatible-license" {
			t.Fatalf("unknown edges must not be marked incompatible: %+v", p)
		}
	}
}

// When several shortest paths exist, the lexicographically smallest id
// sequence wins.
func TestShortestPathLexicographic(t *testing.T) {
	body := `{
		"components": [
			{"id": "root", "license": "MIT", "dependsOn": ["zzz", "aaa"], "materials": ["LICENSE"]},
			{"id": "zzz", "license": "GPL", "dependsOn": ["target"], "materials": ["LICENSE"]},
			{"id": "aaa", "license": "GPL", "dependsOn": ["target"], "materials": ["LICENSE"]},
			{"id": "target", "license": "GPL", "dependsOn": [], "materials": []}
		],` + rulesMITGPL + `,
		"roots": ["root"]
	}`
	r := Check(mustInput(t, body))[0]
	p := findProblem(t, r, "missing-materials", "target")
	if !reflect.DeepEqual(p.Path, []string{"root", "aaa", "target"}) {
		t.Fatalf("want lexicographically smallest shortest path, got %v", p.Path)
	}
}

func TestCleanReport(t *testing.T) {
	body := `{
		"components": [
			{"id": "a", "license": "MIT", "dependsOn": ["b"], "materials": ["LICENSE"]},
			{"id": "b", "license": "GPL", "dependsOn": [], "materials": ["LICENSE"]}
		],` + rulesMITGPL + `,
		"roots": ["a"]
	}`
	r := Check(mustInput(t, body))[0]
	if len(r.Problems) != 0 {
		t.Fatalf("expected no problems, got %+v", r.Problems)
	}
}
