package licensecheck

import "sort"

// Check analyses every delivery root and returns one report per root,
// preserving the declared root order.
func Check(in Input) []Report {
	comps := make(map[string]Component, len(in.Components))
	for _, c := range in.Components {
		comps[c.ID] = c
	}

	required := make(map[string][]string, len(in.Rules))
	allowed := make(map[[2]string]bool, len(in.Rules))
	for _, rl := range in.Rules {
		required[rl.License] = sortedUnique(rl.Required)
		for _, pair := range rl.AllowedPairs {
			allowed[pair] = true
		}
	}

	reports := make([]Report, 0, len(in.Roots))
	for _, root := range in.Roots {
		reports = append(reports, checkRoot(root, comps, required, allowed))
	}
	return reports
}

func checkRoot(root string, comps map[string]Component, required map[string][]string, allowed map[[2]string]bool) Report {
	paths := shortestPaths(root, comps)

	reachable := make([]string, 0, len(paths))
	for id := range paths {
		reachable = append(reachable, id)
	}
	sort.Strings(reachable)

	var problems []Problem
	visitedEdges := map[[2]string]bool{}

	for _, id := range reachable {
		c := comps[id]
		path := paths[id]

		if wants, known := required[c.License]; known {
			missing := missingMaterials(wants, c.Materials)
			if len(missing) > 0 {
				problems = append(problems, Problem{
					Kind:      "missing-materials",
					Component: id,
					Path:      path,
					Materials: missing,
				})
			}
		} else {
			problems = append(problems, Problem{
				Kind:      "unknown-license",
				Component: id,
				Path:      path,
			})
		}

		for _, child := range sortedCopy(c.DependsOn) {
			edgeKey := [2]string{id, child}
			if visitedEdges[edgeKey] {
				continue
			}
			visitedEdges[edgeKey] = true

			if _, ok := paths[child]; !ok {
				continue
			}

			_, parentKnown := required[c.License]
			childComp := comps[child]
			_, childKnown := required[childComp.License]

			kind := ""
			if !parentKnown || !childKnown {
				kind = "unknown-license-edge"
			} else if !allowed[[2]string{c.License, childComp.License}] {
				kind = "incompatible-license"
			}
			if kind != "" {
				problems = append(problems, Problem{
					Kind:      kind,
					Component: id,
					Path:      path,
					Edge:      &Edge{Parent: id, Child: child},
				})
			}
		}
	}

	sort.SliceStable(problems, func(i, j int) bool {
		a, b := problems[i], problems[j]
		if a.Component != b.Component {
			return a.Component < b.Component
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		ae, be := edgeKeyOf(a), edgeKeyOf(b)
		return ae < be
	})

	return Report{Root: root, Reachable: reachable, Problems: problems}
}

// shortestPaths performs BFS from root. Neighbours are expanded in
// lexicographic order, so the first discovered path to every node is the
// lexicographically smallest id sequence among shortest paths. Cycles are
// handled naturally by the visited set.
func shortestPaths(root string, comps map[string]Component) map[string][]string {
	paths := map[string][]string{root: {root}}
	queue := [][]string{{root}}

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		node := path[len(path)-1]

		for _, child := range sortedCopy(comps[node].DependsOn) {
			if _, seen := paths[child]; seen {
				continue
			}
			next := append(append([]string{}, path...), child)
			paths[child] = next
			queue = append(queue, next)
		}
	}
	return paths
}

func missingMaterials(wants, has []string) []string {
	provided := make(map[string]bool, len(has))
	for _, m := range has {
		provided[m] = true
	}
	var missing []string
	for _, m := range wants {
		if !provided[m] {
			missing = append(missing, m)
		}
	}
	return missing
}

func edgeKeyOf(p Problem) string {
	if p.Edge != nil {
		return p.Edge.Parent + "\x00" + p.Edge.Child
	}
	return ""
}

func sortedCopy(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}

func sortedUnique(in []string) []string {
	out := sortedCopy(in)
	if len(out) < 2 {
		return out
	}
	dedup := out[:0]
	for _, v := range out {
		if len(dedup) == 0 || dedup[len(dedup)-1] != v {
			dedup = append(dedup, v)
		}
	}
	return dedup
}
