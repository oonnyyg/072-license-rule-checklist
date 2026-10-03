package checker

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads and validates the component manifest and the license rules.
func Load(manifestPath, rulesPath string) (*Manifest, *Rules, error) {
	manifest := &Manifest{}
	if err := readJSON(manifestPath, manifest); err != nil {
		return nil, nil, fmt.Errorf("manifest: %w", err)
	}
	rules := &Rules{}
	if err := readJSON(rulesPath, rules); err != nil {
		return nil, nil, fmt.Errorf("rules: %w", err)
	}
	if err := Validate(manifest, rules); err != nil {
		return nil, nil, err
	}
	return manifest, rules, nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}
	return nil
}

// Validate checks the structural input constraints. Any failure is an input
// error: the caller must exit with code 2 and must not write a report.
func Validate(manifest *Manifest, rules *Rules) error {
	seenComponents := make(map[string]bool)
	for _, comp := range manifest.Components {
		if comp.ID == "" {
			return fmt.Errorf("component with empty id")
		}
		if seenComponents[comp.ID] {
			return fmt.Errorf("duplicate component id: %s", comp.ID)
		}
		seenComponents[comp.ID] = true
	}

	seenLicenses := make(map[string]bool)
	for _, lic := range rules.Licenses {
		if lic.Code == "" {
			return fmt.Errorf("license rule with empty code")
		}
		if seenLicenses[lic.Code] {
			return fmt.Errorf("duplicate license rule: %s", lic.Code)
		}
		seenLicenses[lic.Code] = true
	}

	for _, comp := range manifest.Components {
		seenDeps := make(map[string]bool)
		for _, dep := range comp.Dependencies {
			if !seenComponents[dep] {
				return fmt.Errorf("component %s references unknown dependency: %s", comp.ID, dep)
			}
			if seenDeps[dep] {
				return fmt.Errorf("component %s lists dependency %s more than once", comp.ID, dep)
			}
			seenDeps[dep] = true
		}
	}

	seenRoots := make(map[string]bool)
	for _, root := range manifest.Roots {
		if !seenComponents[root] {
			return fmt.Errorf("unknown delivery root: %s", root)
		}
		if seenRoots[root] {
			return fmt.Errorf("delivery root listed more than once: %s", root)
		}
		seenRoots[root] = true
	}
	return nil
}
