package licensecheck

import (
	"strings"
	"testing"
)

func TestLoadValid(t *testing.T) {
	in, err := LoadInput(strings.NewReader(`{
		"components": [
			{"id": "a", "license": "MIT", "dependsOn": ["b"], "materials": ["LICENSE"]},
			{"id": "b", "license": "GPL", "dependsOn": [], "materials": []}
		],
		"rules": [
			{"license": "MIT", "required": ["LICENSE"], "allowedPairs": [["MIT", "GPL"]]},
			{"license": "GPL", "required": ["LICENSE", "COPYING"], "allowedPairs": []}
		],
		"roots": ["a"]
	}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(in.Components) != 2 || len(in.Rules) != 2 || in.Roots[0] != "a" {
		t.Fatalf("unexpected parsed payload: %+v", in)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]string{
		"duplicate component": `{
			"components": [{"id": "a", "license": "X"}, {"id": "a", "license": "X"}],
			"rules": [], "roots": []}`,
		"empty component id": `{
			"components": [{"id": "", "license": "X"}], "rules": [], "roots": []}`,
		"duplicate rule": `{
			"components": [{"id": "a", "license": "X"}],
			"rules": [{"license": "X"}, {"license": "X"}], "roots": []}`,
		"unknown dependency": `{
			"components": [{"id": "a", "license": "X", "dependsOn": ["z"]}],
			"rules": [], "roots": []}`,
		"unknown root": `{
			"components": [{"id": "a", "license": "X"}], "rules": [], "roots": ["z"]}`,
		"duplicate root": `{
			"components": [{"id": "a", "license": "X"}], "rules": [], "roots": ["a", "a"]}`,
		"duplicate json key": `{
			"components": [{"id": "a", "id": "b", "license": "X"}],
			"rules": [], "roots": []}`,
		"invalid json": `{"components": [`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadInput(strings.NewReader(body)); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
