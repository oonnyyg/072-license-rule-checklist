// Command licensecheck performs an offline delivery checklist review.
//
// Usage:
//
//	go run ./cmd/licensecheck -input manifest.json -output report.json
//
// Exit codes: 0 = no problems, 1 = problems found, 2 = invalid input.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"example.com/license-rule-checklist/internal/licensecheck"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("licensecheck", flag.ContinueOnError)
	inputPath := fs.String("input", "", "path to the checklist JSON file")
	outputPath := fs.String("output", "", "path to write the JSON report")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *inputPath == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "both -input and -output are required")
		return 2
	}

	f, err := os.Open(*inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "input error: %v\n", err)
		return 2
	}
	in, err := licensecheck.LoadInput(f)
	f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "input error: %v\n", err)
		return 2
	}

	reports := licensecheck.Check(in)

	out, err := os.Create(*outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "output error: %v\n", err)
		return 2
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(reports); err != nil {
		out.Close()
		fmt.Fprintf(os.Stderr, "output error: %v\n", err)
		return 2
	}
	if err := out.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "output error: %v\n", err)
		return 2
	}

	for _, r := range reports {
		if len(r.Problems) > 0 {
			return 1
		}
	}
	return 0
}
