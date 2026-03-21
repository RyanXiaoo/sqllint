package linter

import (
	"strings"

	"github.com/ryanxiao/go-sqllint/internal/config"
	"github.com/ryanxiao/go-sqllint/internal/rules"
)

// Linter holds the set of rules to run and executes them against SQL input.
type Linter struct {
	config config.Config
	rules  []rules.Rule
}

// New creates a Linter with the default set of rules.
// This is the Go convention instead of constructors — a plain function named New.
func New(cfg config.Config) *Linter {
	return &Linter{
		config: cfg,
		rules: []rules.Rule{
			rules.SelectStar{},
			rules.MissingWhere{},
			rules.KeywordCasing{},
			rules.TrailingSemicolon{},
			rules.LeadingWildcard{},
			rules.ImplicitJoin{},
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

	var filtered []rules.Violation
	for _, v := range all {
		// skip sqllint:ignore lines
		if strings.Contains(lines[v.Line-1], "sqllint:ignore") {
			continue
		}

		// check config for this rule
		if rc, ok := l.config.Rules[v.RuleID]; ok {
			if rc.Enabled != nil && !*rc.Enabled {
				continue
			}
			if rc.Severity == "error" {
				v.Severity = rules.SeverityError
			} else if rc.Severity == "warning" {
				v.Severity = rules.SeverityWarning
			}
		}

		filtered = append(filtered, v)
	}

	return Result{
		File: filename,
		Violations: filtered,
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
