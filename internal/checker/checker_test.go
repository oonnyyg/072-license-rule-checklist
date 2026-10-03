package checker

import (
	"reflect"
	"testing"
)

func sampleInput() (*Manifest, *Rules) {
	manifest := &Manifest{
		Roots: []string{"a", "c", "u"},
		Components: []Component{
			{ID: "a", License: "A", Dependencies: []string{"b"}, Materials: []string{"notice"}},
			{ID: "b", License: "B", Dependencies: []string{"a"}, Materials: nil},
			{ID: "c", License: "A", Dependencies: []string{"b"}, Materials: []string{"notice"}},
			{ID: "u", License: "X", Dependencies: []string{"b"}, Materials: []string{"notice"}},
			{ID: "isolated", License: "A", Materials: nil},
		},
	}
	rules := &Rules{
		Licenses: []LicenseRule{
			{Code: "A", RequiredMaterials: []string{"notice"}},
			{Code: "B", RequiredMaterials: []string{"notice"}},
		},
		AllowedPairs: []LicensePair{{Parent: "A", Child: "B"}},
	}
	return manifest, rules
}

func TestRunRootReports(t *testing.T) {
	manifest, rules := sampleInput()
	report := Run(manifest, rules)

	if len(report.Roots) != 3 {
		t.Fatalf("want 3 root reports, got %d", len(report.Roots))
	}

	rootA := report.Roots[0]
	if got := rootA.Reachable; !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("root a reachable = %v", got)
	}
	if issue := findIssue(rootA.Issues, "b", ""); issue == nil || issue.Kind != IssueMissingMaterials ||
		!reflect.DeepEqual(issue.Materials, []string{"notice"}) ||
		!reflect.DeepEqual(issue.Path, []string{"a", "b"}) {
		t.Fatalf("root a: unexpected missing-material issue: %+v", issue)
	}
	if issue := findIssue(rootA.Issues, "b", "a"); issue == nil || issue.Kind != IssueIncompatibleEdge ||
		!reflect.DeepEqual(issue.Path, []string{"a", "b"}) {
		t.Fatalf("root a: reverse edge b->a must be incompatible: %+v", issue)
	}
	if findIssue(rootA.Issues, "a", "b") != nil {
		t.Fatalf("root a: forward edge A->B must be allowed")
	}

	rootC := report.Roots[1]
	if got := rootC.Reachable; !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("root c reachable = %v", got)
	}
	if issue := findIssue(rootC.Issues, "b", "a"); issue == nil ||
		!reflect.DeepEqual(issue.Path, []string{"c", "b"}) {
		t.Fatalf("root c: shared component b must carry path from c: %+v", issue)
	}

	rootU := report.Roots[2]
	if issue := findIssue(rootU.Issues, "u", ""); issue == nil || issue.Kind != IssueUnknownLicense {
		t.Fatalf("root u: u must be flagged unknown license: %+v", issue)
	}
	if issue := findIssue(rootU.Issues, "u", "b"); issue == nil || issue.Kind != IssueUndecidableEdge {
		t.Fatalf("root u: edge from unknown license must be undecidable: %+v", issue)
	}
	if issue := findIssue(rootU.Issues, "b", "a"); issue == nil ||
		!reflect.DeepEqual(issue.Path, []string{"u", "b"}) {
		t.Fatalf("root u: known pair b->a still evaluated with path from u: %+v", issue)
	}

	for _, rr := range report.Roots {
		if findIssue(rr.Issues, "isolated", "") != nil {
			t.Fatalf("unreachable component must not affect root %s", rr.Root)
		}
	}
}

func TestCycleCheckedOnce(t *testing.T) {
	manifest := &Manifest{
		Roots: []string{"a"},
		Components: []Component{
			{ID: "a", License: "A", Dependencies: []string{"b"}},
			{ID: "b", License: "B", Dependencies: []string{"a"}},
		},
	}
	rules := &Rules{Licenses: []LicenseRule{{Code: "A"}, {Code: "B"}}}
	report := Run(manifest, rules)
	edges := 0
	for _, is := range report.Roots[0].Issues {
		if is.Dependency != "" {
			edges++
		}
	}
	if edges != 2 {
		t.Fatalf("each directed edge checked once: want 2 edge issues, got %d", edges)
	}
}

func TestSameLicenseNotAllowedByDefault(t *testing.T) {
	manifest := &Manifest{Roots: []string{"a"}, Components: []Component{
		{ID: "a", License: "A", Dependencies: []string{"b"}},
		{ID: "b", License: "A"},
	}}
	rules := &Rules{Licenses: []LicenseRule{{Code: "A"}}}
	report := Run(manifest, rules)
	if issue := findIssue(report.Roots[0].Issues, "a", "b"); issue == nil ||
		issue.Kind != IssueIncompatibleEdge {
		t.Fatalf("same license pair must not be compatible by default: %+v", issue)
	}
}

func TestLexicographicallySmallestShortestPath(t *testing.T) {
	manifest := &Manifest{Roots: []string{"r"}, Components: []Component{
		{ID: "r", License: "A", Dependencies: []string{"m2", "m1"}},
		{ID: "m1", License: "A", Dependencies: []string{"z"}},
		{ID: "m2", License: "A", Dependencies: []string{"z"}},
		{ID: "z", License: "A"},
	}}
	paths := bfs("r", indexComponents(manifest))
	if got := paths["z"]; !reflect.DeepEqual(got, []string{"r", "m1", "z"}) {
		t.Fatalf("want lexicographically smallest shortest path, got %v", got)
	}
}

func TestValidationErrors(t *testing.T) {
	base, rules := sampleInput()

	cases := []struct {
		name   string
		mutate func(*Manifest, *Rules)
	}{
		{"empty component id", func(m *Manifest, _ *Rules) {
			m.Components[0].ID = ""
		}},
		{"duplicate component id", func(m *Manifest, _ *Rules) {
			m.Components[0].ID = "b"
		}},
		{"duplicate license rule", func(_ *Manifest, r *Rules) {
			r.Licenses = append(r.Licenses, LicenseRule{Code: "A"})
		}},
		{"unknown dependency", func(m *Manifest, _ *Rules) {
			m.Components[0].Dependencies = []string{"ghost"}
		}},
		{"duplicate dependency", func(m *Manifest, _ *Rules) {
			m.Components[0].Dependencies = []string{"b", "b"}
		}},
		{"unknown root", func(m *Manifest, _ *Rules) {
			m.Roots = []string{"ghost"}
		}},
		{"duplicate root", func(m *Manifest, _ *Rules) {
			m.Roots = []string{"a", "a"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, r := &Manifest{Components: append([]Component(nil), base.Components...), Roots: append([]string(nil), base.Roots...)},
				&Rules{Licenses: append([]LicenseRule(nil), rules.Licenses...), AllowedPairs: append([]LicensePair(nil), rules.AllowedPairs...)}
			tc.mutate(m, r)
			if err := Validate(m, r); err == nil {
				t.Fatalf("expected input error")
			}
		})
	}
}

func indexComponents(m *Manifest) map[string]*Component {
	out := make(map[string]*Component, len(m.Components))
	for i := range m.Components {
		out[m.Components[i].ID] = &m.Components[i]
	}
	return out
}

func findIssue(issues []Issue, component, dependency string) *Issue {
	for i := range issues {
		if issues[i].Component == component && issues[i].Dependency == dependency {
			return &issues[i]
		}
	}
	return nil
}
