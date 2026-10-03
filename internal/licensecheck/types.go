package licensecheck

// Input is the complete checklist payload read from disk.
type Input struct {
	Components []Component `json:"components"`
	Rules      []Rule      `json:"rules"`
	Roots      []string    `json:"roots"`
}

// Component describes one delivered software component.
type Component struct {
	ID        string   `json:"id"`
	License   string   `json:"license"`
	DependsOn []string `json:"dependsOn"`
	Materials []string `json:"materials"`
}

// Rule states which materials a license requires and which directed
// license pairs (parent -> child) are allowed to coexist.
type Rule struct {
	License      string      `json:"license"`
	Required     []string    `json:"required"`
	AllowedPairs [][2]string `json:"allowedPairs"`
}

// Report is the JSON document written for one delivery root.
type Report struct {
	Root      string    `json:"root"`
	Reachable []string  `json:"reachable"`
	Problems  []Problem `json:"problems"`
}

// Problem locates a missing material or an incompatible/unjudgable edge.
type Problem struct {
	Kind      string   `json:"kind"`
	Component string   `json:"component"`
	Path      []string `json:"path"`
	Materials []string `json:"materials,omitempty"`
	Edge      *Edge    `json:"edge,omitempty"`
}

// Edge identifies a dependency edge from parent to child.
type Edge struct {
	Parent string `json:"parent"`
	Child  string `json:"child"`
}
