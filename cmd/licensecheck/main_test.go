package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const cleanInput = `{
	"components": [
		{"id": "a", "license": "MIT", "dependsOn": ["b"], "materials": ["LICENSE"]},
		{"id": "b", "license": "GPL", "dependsOn": [], "materials": ["LICENSE"]}
	],
	"rules": [
		{"license": "MIT", "required": ["LICENSE"], "allowedPairs": [["MIT", "GPL"]]},
		{"license": "GPL", "required": ["LICENSE"], "allowedPairs": []}
	],
	"roots": ["a"]
}`

const badInput = `{
	"components": [
		{"id": "a", "license": "MIT", "dependsOn": ["b"], "materials": []},
		{"id": "b", "license": "GPL", "dependsOn": [], "materials": ["LICENSE"]}
	],
	"rules": [
		{"license": "MIT", "required": ["LICENSE"], "allowedPairs": []},
		{"license": "GPL", "required": ["LICENSE"], "allowedPairs": []}
	],
	"roots": ["a"]
}`

func writeFile(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestRunExitCodesAndReport(t *testing.T) {
	in := writeFile(t, "in.json", cleanInput)
	out := filepath.Join(t.TempDir(), "report.json")

	if code := run([]string{"-input", in, "-output", out}); code != 0 {
		t.Fatalf("clean input: want exit 0, got %d", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("report not written: %v", err)
	}
	var reports []map[string]any
	if err := json.Unmarshal(data, &reports); err != nil {
		t.Fatalf("report is not JSON: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("want 1 report, got %d", len(reports))
	}

	badIn := writeFile(t, "bad.json", badInput)
	if code := run([]string{"-input", badIn, "-output", out}); code != 1 {
		t.Fatalf("problems: want exit 1, got %d", code)
	}
}

func TestRunInvalidInputWritesNothing(t *testing.T) {
	dir := t.TempDir()
	in := writeFile(t, "in.json", `{"components": [{"id": "a"}], "rules": [], "roots": ["ghost"]}`)
	out := filepath.Join(dir, "report.json")

	if code := run([]string{"-input", in, "-output", out}); code != 2 {
		t.Fatalf("invalid input: want exit 2, got %d", code)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("report must not be written on invalid input, stat err=%v", err)
	}

	if code := run([]string{"-input", in}); code != 2 {
		t.Fatalf("missing flag: want exit 2, got %d", code)
	}
}
