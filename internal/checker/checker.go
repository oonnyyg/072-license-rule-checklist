package checker

import (
	"sort"
)

// Run evaluates every delivery root and returns the assembled report.
func Run(manifest *Manifest, rules *Rules) *Report {
	components := make(map[string]*Component, len(manifest.Components))
	for i := range manifest.Components {
		components[manifest.Components[i].ID] = &manifest.Components[i]
	}

	required := make(map[string][]string)
	known := make(map[string]bool)
	for _, lic := range rules.Licenses {
		known[lic.Code] = true
		req := append([]string(nil), lic.RequiredMaterials...)
		sort.Strings(req)
		required[lic.Code] = req
	}

	allowed := make(map[LicensePair]bool)
	for _, pair := range rules.AllowedPairs {
		allowed[pair] = true
	}

	report := &Report{Roots: make([]RootReport, 0, len(manifest.Roots))}
	for _, root := range manifest.Roots {
		report.Roots = append(report.Roots, checkRoot(root, components, known, required, allowed))
	}
	return report
}

func checkRoot(
	root string,
	components map[string]*Component,
	known map[string]bool,
	required map[string][]string,
	allowed map[LicensePair]bool,
) RootReport {
	paths := bfs(root, components)

	reachable := make([]string, 0, len(paths))
	for id := range paths {
		reachable = append(reachable, id)
	}
	sort.Strings(reachable)

	var issues []Issue
	for _, id := range reachable {
		comp := components[id]
		path := paths[id]

		if !known[comp.License] {
			issues = append(issues, Issue{
				Kind:      IssueUnknownLicense,
				Component: id,
				Path:      path,
			})
		} else if missing := missingMaterials(required[comp.License], comp.Materials); len(missing) > 0 {
			issues = append(issues, Issue{
				Kind:      IssueMissingMaterials,
				Component: id,
				Materials: missing,
				Path:      path,
			})
		}

		deps := append([]string(nil), comp.Dependencies...)
		sort.Strings(deps)
		for _, dep := range deps {
			if _, ok := paths[dep]; !ok {
				continue
			}
			child := components[dep]
			kind := IssueIncompatibleEdge
			if !known[comp.License] || !known[child.License] {
				kind = IssueUndecidableEdge
			} else if allowed[LicensePair{Parent: comp.License, Child: child.License}] {
				continue
			}
			issues = append(issues, Issue{
				Kind:       kind,
				Component:  id,
				Dependency: dep,
				Path:       path,
			})
		}
	}

	sortIssues(issues)
	return RootReport{Root: root, Reachable: reachable, Issues: issues}
}

func missingMaterials(required, attached []string) []string {
	have := make(map[string]bool, len(attached))
	for _, m := range attached {
		have[m] = true
	}
	var missing []string
	for _, m := range required {
		if !have[m] {
			missing = append(missing, m)
		}
	}
	return missing
}

// bfs returns the shortest path from root to every reachable component.
// Ties are broken by the lexicographically smallest sequence of component ids:
// visiting each node's neighbors in sorted order makes first discovery win.
func bfs(root string, components map[string]*Component) map[string][]string {
	paths := map[string][]string{root: {root}}
	queue := []string{root}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		next := append([]string(nil), components[id].Dependencies...)
		sort.Strings(next)
		for _, dep := range next {
			if _, seen := paths[dep]; seen {
				continue
			}
			path := append(append([]string(nil), paths[id]...), dep)
			paths[dep] = path
			queue = append(queue, dep)
		}
	}
	return paths
}

func sortIssues(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if a.Component != b.Component {
			return a.Component < b.Component
		}
		if a.Dependency != b.Dependency {
			return a.Dependency < b.Dependency
		}
		return a.Kind < b.Kind
	})
}
