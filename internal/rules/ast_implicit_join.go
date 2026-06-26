package rules

import "github.com/pgplex/pgparser/nodes"

// ASTImplicitJoin flags comma-style joins (FROM a, b) and recommends explicit
// JOIN syntax. Working from the parse tree (rather than scanning for a comma
// after FROM) means it sees multi-line FROM clauses and never trips on commas
// inside strings, function calls, or comments: an explicit JOIN collapses into
// a single JoinExpr, so more than one FromClause item means a comma join.
type ASTImplicitJoin struct{}

func (r ASTImplicitJoin) ID() string {
	return "implicit-join"
}

func (r ASTImplicitJoin) CheckAST(stmts []nodes.Node, sql string, lines []string) []Violation {
	var violations []Violation
	// The parser does not populate node source locations, so we resolve the
	// line by counting SELECT statements in document order and mapping to the
	// Nth "SELECT" keyword in the text (counting SELECTs rather than FROMs
	// keeps the counter aligned with the keyword we search for, even when a
	// DELETE ... FROM appears earlier in the file). The reported line is the
	// start of the offending statement.
	selectSeen := 0
	for _, stmt := range stmts {
		r.walkStmt(unwrapRawStmt(stmt), sql, &selectSeen, &violations)
	}
	return violations
}

func (r ASTImplicitJoin) walkStmt(node nodes.Node, sql string, selectSeen *int, violations *[]Violation) {
	if sel, ok := node.(*nodes.SelectStmt); ok {
		r.checkSelect(sel, sql, selectSeen, violations)
	}
}

func (r ASTImplicitJoin) checkSelect(sel *nodes.SelectStmt, sql string, selectSeen *int, violations *[]Violation) {
	if sel == nil {
		return
	}

	*selectSeen++
	occurrence := *selectSeen

	if sel.FromClause != nil && sel.FromClause.Len() > 1 {
		*violations = append(*violations, Violation{
			RuleID:   r.ID(),
			Message:  "Avoid implicit joins (comma-separated FROM tables); use explicit JOIN instead",
			Line:     findKeywordLineMasked(sql, "SELECT", occurrence),
			Severity: SeverityWarning,
		})
	}

	// Descend into subqueries so nested SELECTs are checked too.
	if sel.FromClause != nil {
		for _, from := range sel.FromClause.Items {
			r.walkFromNode(from, sql, selectSeen, violations)
		}
	}
	r.walkExpr(sel.WhereClause, sql, selectSeen, violations)
}

func (r ASTImplicitJoin) walkFromNode(node nodes.Node, sql string, selectSeen *int, violations *[]Violation) {
	switch n := node.(type) {
	case *nodes.RangeSubselect:
		if sel, ok := n.Subquery.(*nodes.SelectStmt); ok {
			r.checkSelect(sel, sql, selectSeen, violations)
		}
	case *nodes.JoinExpr:
		r.walkFromNode(n.Larg, sql, selectSeen, violations)
		r.walkFromNode(n.Rarg, sql, selectSeen, violations)
	}
}

func (r ASTImplicitJoin) walkExpr(node nodes.Node, sql string, selectSeen *int, violations *[]Violation) {
	switch n := node.(type) {
	case *nodes.SubLink:
		if sel, ok := n.Subselect.(*nodes.SelectStmt); ok {
			r.checkSelect(sel, sql, selectSeen, violations)
		}
	case *nodes.BoolExpr:
		if n.Args != nil {
			for _, arg := range n.Args.Items {
				r.walkExpr(arg, sql, selectSeen, violations)
			}
		}
	case *nodes.A_Expr:
		r.walkExpr(n.Lexpr, sql, selectSeen, violations)
		r.walkExpr(n.Rexpr, sql, selectSeen, violations)
	}
}
