package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"example.com/license-rule-checklist/internal/checker"
)

func writeFile(t *testing.T, dir, name string, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExitCodes(t *testing.T) {
	dir := t.TempDir()

	cleanManifest := writeFile(t, dir, "clean.json", &checker.Manifest{
		Roots: []string{"a"},
		Components: []checker.Component{
			{ID: "a", License: "A", Materials: []string{"notice"}},
		},
	})
	badManifest := writeFile(t, dir, "bad.json", &checker.Manifest{
		Roots: []string{"a"},
		Components: []checker.Component{
			{ID: "a", License: "A"},
		},
	})
	dupManifest := writeFile(t, dir, "dup.json", &checker.Manifest{
		Roots: []string{"a"},
		Components: []checker.Component{
			{ID: "a", License: "A"},
			{ID: "a", License: "A"},
		},
	})
	rulesPath := writeFile(t, dir, "rules.json", &checker.Rules{
		Licenses: []checker.LicenseRule{{Code: "A", RequiredMaterials: []string{"notice"}}},
	})

	cleanOut := filepath.Join(dir, "clean-report.json")
	if code := runWith(cleanManifest, rulesPath, cleanOut); code != 0 {
		t.Fatalf("clean run exit code = %d, want 0", code)
	}
	if _, err := os.Stat(cleanOut); err != nil {
		t.Fatalf("clean report should be written: %v", err)
	}

	badOut := filepath.Join(dir, "bad-report.json")
	if code := runWith(badManifest, rulesPath, badOut); code != 1 {
		t.Fatalf("run with issues exit code = %d, want 1", code)
	}

	dupOut := filepath.Join(dir, "dup-report.json")
	if code := runWith(dupManifest, rulesPath, dupOut); code != 2 {
		t.Fatalf("input error exit code = %d, want 2", code)
	}
	if _, err := os.Stat(dupOut); !os.IsNotExist(err) {
		t.Fatalf("input error must not write report, stat err = %v", err)
	}
}

func runWith(manifest, rules, out string) int {
	oldArgs := os.Args
	os.Args = []string{"licensecheck", "-manifest", manifest, "-rules", rules, "-out", out}
	defer func() { os.Args = oldArgs }()
	return run()
}
