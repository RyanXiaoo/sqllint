package rules

import "github.com/pgplex/pgparser/nodes"

// Severity represents how serious a lint violation is.
type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	default:
		return "warning"
	}
}

// Violation represents a single lint finding.
type Violation struct {
	RuleID   string
	Message  string
	Line     int      // 1-indexed
	Severity Severity
}

// Rule is the interface for string-based lint rules (Phase 1).
type Rule interface {
	ID() string
	Check(sql string, lines []string) []Violation
}

// ASTRule is the interface for AST-based lint rules (Phase 2).
// These receive parsed statements from pgparser, enabling
// accurate analysis that string matching can't achieve.
type ASTRule interface {
	ID() string
	CheckAST(stmts []nodes.Node, sql string, lines []string) []Violation
}
