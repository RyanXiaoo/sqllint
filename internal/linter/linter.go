package linter

import (
	"strings"

	"github.com/ryanxiao/go-sqllint/internal/rules"
)

// Linter holds the set of rules to run and executes them against SQL input.
type Linter struct {
	rules []rules.Rule
}

// New creates a Linter with the default set of rules.
// This is the Go convention instead of constructors — a plain function named New.
func New() *Linter {
	return &Linter{
		rules: []rules.Rule{
			rules.SelectStar{},
			rules.MissingWhere{},
			rules.KeywordCasing{},
			rules.TrailingSemicolon{},
			rules.LeadingWildcard{},
			rules.ImplicitJoin{},
			rules.AliasConsistency{},
		},
	}
}

// Result holds all violations for a single file.
type Result struct {
	File       string
	Violations []rules.Violation
}

// Lint analyzes the given SQL content and returns all violations found.
func (l *Linter) Lint(filename, sql string) Result {
	lines := strings.Split(sql, "\n")

	var all []rules.Violation
	for _, rule := range l.rules {
		violations := rule.Check(sql, lines)
		all = append(all, violations...)
	}

	return Result{
		File:       filename,
		Violations: all,
	}
}

// HasErrors returns true if any violation has error severity.
func (r Result) HasErrors() bool {
	for _, v := range r.Violations {
		if v.Severity == rules.SeverityError {
			return true
		}
	}
	return false
}
