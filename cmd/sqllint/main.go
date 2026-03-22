package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ryanxiao/go-sqllint/internal/config"
	"github.com/ryanxiao/go-sqllint/internal/linter"
	"github.com/ryanxiao/go-sqllint/internal/output"
)

type fileResult struct {
	result linter.Result
	err    error
}

func main() {
	format := flag.String("format", "text", "Output format: text, json, or sarif")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: sqllint [flags] [file ...]\n\n")
		fmt.Fprintf(os.Stderr, "If no files are given, reads from stdin.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	cfg, err := config.Load(".sqllint.yaml")
	if err != nil {
		cfg = config.Config{}
	}
	l := linter.New(cfg)

	var results []linter.Result

	files := flag.Args()
	if len(files) == 0 {
		sql, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		results = append(results, l.Lint("<stdin>", string(sql)))
	} else {
		// Expand all glob patterns into a flat list of paths.
		var paths []string
		for _, pattern := range files {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Bad pattern %q: %v\n", pattern, err)
				os.Exit(1)
			}
			if matches == nil {
				matches = []string{pattern}
			}
			paths = append(paths, matches...)
		}

		// Fan-out: lint files concurrently.
		ch := make(chan fileResult, len(paths))
		var wg sync.WaitGroup
		for _, p := range paths {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				sql, err := os.ReadFile(p)
				if err != nil {
					ch <- fileResult{err: err}
					return
				}
				ch <- fileResult{result: l.Lint(p, string(sql))}
			}(p)
		}
		go func() { wg.Wait(); close(ch) }()

		for fr := range ch {
			if fr.err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file: %v\n", fr.err)
				os.Exit(1)
			}
			results = append(results, fr.result)
		}

		// Sort for deterministic output.
		sort.Slice(results, func(i, j int) bool {
			return results[i].File < results[j].File
		})
	}

	// Output results in the requested format.
	switch strings.ToLower(*format) {
	case "json":
		if err := output.JSON(os.Stdout, results); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JSON: %v\n", err)
			os.Exit(1)
		}
	case "sarif":
		if err := output.SARIF(os.Stdout, results); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing SARIF: %v\n", err)
			os.Exit(1)
		}
	default:
		output.Text(os.Stdout, results)
	}

	// Exit code 1 = errors, 2 = warnings only, 0 = clean.
	hasErrors, hasWarnings := false, false
	for _, r := range results {
		if r.HasErrors() {
			hasErrors = true
		}
		if r.HasWarnings() {
			hasWarnings = true
		}
	}
	if hasErrors {
		os.Exit(1)
	}
	if hasWarnings {
		os.Exit(2)
	}
}
