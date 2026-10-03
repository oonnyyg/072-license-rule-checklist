package checker

// Manifest describes the delivered components and the requested delivery roots.
type Manifest struct {
	Components []Component `json:"components"`
	Roots      []string    `json:"roots"`
}

// Component is a single delivered component.
type Component struct {
	ID           string   `json:"id"`
	License      string   `json:"license"`
	Dependencies []string `json:"dependencies"`
	Materials    []string `json:"materials"`
}

// Rules describes per-license required materials and allowed license pairs.
type Rules struct {
	Licenses     []LicenseRule `json:"licenses"`
	AllowedPairs []LicensePair `json:"allowed_pairs"`
}

// LicenseRule lists the materials required for one license code.
type LicenseRule struct {
	Code              string   `json:"code"`
	RequiredMaterials []string `json:"required_materials"`
}

// LicensePair allows a parent component to carry a dependency of child license.
type LicensePair struct {
	Parent string `json:"parent"`
	Child  string `json:"child"`
}

// Report is the full output document.
type Report struct {
	Roots []RootReport `json:"roots"`
}

// RootReport covers everything reachable from one delivery root.
type RootReport struct {
	Root      string   `json:"root"`
	Reachable []string `json:"reachable"`
	Issues    []Issue  `json:"issues"`
}

// Issue kinds.
const (
	IssueMissingMaterials = "missing_materials"
	IssueUnknownLicense   = "unknown_license"
	IssueIncompatibleEdge = "incompatible_dependency"
	IssueUndecidableEdge  = "undecidable_dependency"
)

// Issue locates a component problem or a dependency-edge problem.
type Issue struct {
	Kind       string   `json:"kind"`
	Component  string   `json:"component"`
	Dependency string   `json:"dependency,omitempty"`
	Materials  []string `json:"missing_materials,omitempty"`
	Path       []string `json:"path"`
}
