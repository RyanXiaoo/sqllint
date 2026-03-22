package linter

import (
	"fmt"
	"os"
	"strings"

	"github.com/pgplex/pgparser/parser"
	"github.com/ryanxiao/go-sqllint/internal/config"
	"github.com/ryanxiao/go-sqllint/internal/rules"
)

type Linter struct {
	config   config.Config
	rules    []rules.Rule
	astRules []rules.ASTRule
}

func New(cfg config.Config) *Linter {
	return &Linter{
		config: cfg,
		rules: []rules.Rule{
			rules.KeywordCasing{},
			rules.TrailingSemicolon{},
			rules.LeadingWildcard{},
			rules.ImplicitJoin{},
		},
		astRules: []rules.ASTRule{
			rules.ASTSelectStar{},
			rules.ASTMissingWhere{},
			rules.ASTNotInNullable{},
			rules.ASTUnusedAlias{},
			rules.ASTMissingGroupByCol{},
		},
	}
}

type Result struct {
	File       string
	Violations []rules.Violation
}

func (l *Linter) Lint(filename, sql string) Result {
	lines := strings.Split(sql, "\n")

	var all []rules.Violation

	for _, rule := range l.rules {
		violations := rule.Check(sql, lines)
		all = append(all, violations...)
	}

	tree, err := parser.Parse(sql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AST parse warning for %s: %v (falling back to string rules only)\n", filename, err)
	} else if tree != nil {
		for _, rule := range l.astRules {
			violations := rule.CheckAST(tree.Items, sql, lines)
			all = append(all, violations...)
		}
	}

	var filtered []rules.Violation
	for _, v := range all {
		if v.Line > 0 && v.Line <= len(lines) && strings.Contains(lines[v.Line-1], "sqllint:ignore") {
			continue
		}

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
		File:       filename,
		Violations: filtered,
	}
}

func (r Result) HasErrors() bool {
	for _, v := range r.Violations {
		if v.Severity == rules.SeverityError {
			return true
		}
	}
	return false
}

func (r Result) HasWarnings() bool {
	for _, v := range r.Violations {
		if v.Severity == rules.SeverityWarning {
			return true
		}
	}
	return false
}
