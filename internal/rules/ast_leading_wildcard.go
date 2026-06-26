package rules

import (
	"strings"

	"github.com/pgplex/pgparser/nodes"
)

// ASTLeadingWildcard flags LIKE/ILIKE patterns that begin with '%' (e.g.
// LIKE '%foo'), which force a full scan because the index can't be used.
// Reading the parse tree rather than the raw line means the pattern is taken
// from the actual string constant, so it won't false-positive on a '%' that
// appears in a comment or an unrelated string literal.
type ASTLeadingWildcard struct{}

func (r ASTLeadingWildcard) ID() string {
	return "leading-wildcard"
}

func (r ASTLeadingWildcard) CheckAST(stmts []nodes.Node, sql string, lines []string) []Violation {
	var violations []Violation
	// The parser does not populate node source locations, so we resolve the
	// line by searching the text for the pattern literal. seen counts repeated
	// identical patterns so each one maps to its own occurrence. Comments are
	// masked (but not strings — the pattern lives inside a string) so a pattern
	// merely mentioned in a comment is not matched.
	searchText := maskNonCode(sql, false)
	seen := map[string]int{}
	for _, stmt := range stmts {
		r.walkStmt(unwrapRawStmt(stmt), searchText, seen, &violations)
	}
	return violations
}

func (r ASTLeadingWildcard) walkStmt(node nodes.Node, sql string, seen map[string]int, violations *[]Violation) {
	switch n := node.(type) {
	case *nodes.SelectStmt:
		r.walkExpr(n.WhereClause, sql, seen, violations)
		r.walkExpr(n.HavingClause, sql, seen, violations)
	case *nodes.DeleteStmt:
		r.walkExpr(n.WhereClause, sql, seen, violations)
	case *nodes.UpdateStmt:
		r.walkExpr(n.WhereClause, sql, seen, violations)
	}
}

func (r ASTLeadingWildcard) walkExpr(node nodes.Node, sql string, seen map[string]int, violations *[]Violation) {
	switch n := node.(type) {
	case *nodes.A_Expr:
		if n.Kind == nodes.AEXPR_LIKE || n.Kind == nodes.AEXPR_ILIKE {
			if pattern, ok := r.leadingWildcardPattern(n.Rexpr); ok {
				anchor := "'" + pattern
				seen[anchor]++
				*violations = append(*violations, Violation{
					RuleID:   r.ID(),
					Message:  "Leading wildcard in LIKE/ILIKE pattern prevents index usage",
					Line:     findKeywordLine(sql, anchor, seen[anchor]),
					Severity: SeverityWarning,
				})
			}
		}
		r.walkExpr(n.Lexpr, sql, seen, violations)
		r.walkExpr(n.Rexpr, sql, seen, violations)
	case *nodes.BoolExpr:
		if n.Args != nil {
			for _, arg := range n.Args.Items {
				r.walkExpr(arg, sql, seen, violations)
			}
		}
	case *nodes.SubLink:
		r.walkStmt(n.Subselect, sql, seen, violations)
	}
}

// leadingWildcardPattern returns the string constant if node is a string
// literal beginning with '%', along with whether it qualified.
func (r ASTLeadingWildcard) leadingWildcardPattern(node nodes.Node) (string, bool) {
	c, ok := node.(*nodes.A_Const)
	if !ok || c.Isnull {
		return "", false
	}
	s, ok := c.Val.(*nodes.String)
	if !ok || !strings.HasPrefix(s.Str, "%") {
		return "", false
	}
	return s.Str, true
}
