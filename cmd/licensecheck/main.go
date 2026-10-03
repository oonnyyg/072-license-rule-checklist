package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"example.com/license-rule-checklist/internal/checker"
)

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("licensecheck", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	manifestPath := fs.String("manifest", "", "path to the JSON component manifest")
	rulesPath := fs.String("rules", "", "path to the JSON license rules")
	outputPath := fs.String("out", "", "path to write the JSON report")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}

	if *manifestPath == "" || *rulesPath == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "usage: licensecheck -manifest <manifest.json> -rules <rules.json> -out <report.json>")
		return 2
	}

	manifest, rules, err := checker.Load(*manifestPath, *rulesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "input error: %v\n", err)
		return 2
	}

	report := checker.Run(manifest, rules)
	if err := writeJSON(*outputPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write report: %v\n", err)
		return 2
	}

	if hasIssues(report) {
		return 1
	}
	return 0
}

func writeJSON(path string, report *checker.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func hasIssues(report *checker.Report) bool {
	for _, root := range report.Roots {
		if len(root.Issues) > 0 {
			return true
		}
	}
	return false
}
