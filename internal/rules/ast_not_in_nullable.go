package rules

import "github.com/pgplex/pgparser/nodes"

// ASTNotInNullable flags NOT IN (subquery) patterns.
// When any value in the subquery result is NULL, NOT IN silently returns
// zero rows -- a common production bug. NOT EXISTS is the safe alternative.
type ASTNotInNullable struct{}

func (r ASTNotInNullable) ID() string {
	return "not-in-nullable"
}

func (r ASTNotInNullable) CheckAST(stmts []nodes.Node, sql string, lines []string) []Violation {
	var violations []Violation

	// The parser does not populate node source locations, so we resolve the
	// line by counting flagged occurrences and mapping to the Nth "NOT IN" in
	// the text (comments and strings masked out).
	notInSeen := 0
	for _, stmt := range stmts {
		node := unwrapRawStmt(stmt)
		r.walkStmt(node, sql, &notInSeen, &violations)
	}

	return violations
}

func (r ASTNotInNullable) walkStmt(node nodes.Node, sql string, notInSeen *int, violations *[]Violation) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *nodes.SelectStmt:
		r.walkExpr(n.WhereClause, sql, notInSeen, violations)
		r.walkExpr(n.HavingClause, sql, notInSeen, violations)
	case *nodes.DeleteStmt:
		r.walkExpr(n.WhereClause, sql, notInSeen, violations)
	case *nodes.UpdateStmt:
		r.walkExpr(n.WhereClause, sql, notInSeen, violations)
	}
}

func (r ASTNotInNullable) walkExpr(node nodes.Node, sql string, notInSeen *int, violations *[]Violation) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *nodes.BoolExpr:
		if n.Boolop == nodes.NOT_EXPR && n.Args != nil {
			for _, arg := range n.Args.Items {
				if sub, ok := arg.(*nodes.SubLink); ok {
					if nodes.SubLinkType(sub.SubLinkType) == nodes.ANY_SUBLINK {
						*notInSeen++
						*violations = append(*violations, Violation{
							RuleID:   r.ID(),
							Message:  "NOT IN (subquery) returns no rows if any subquery value is NULL; use NOT EXISTS instead",
							Line:     findKeywordLineMasked(sql, "NOT IN", *notInSeen),
							Severity: SeverityError,
						})
					}
				}
			}
		}
		if n.Args != nil {
			for _, arg := range n.Args.Items {
				r.walkExpr(arg, sql, notInSeen, violations)
			}
		}
	case *nodes.A_Expr:
		// A literal "NOT IN (1, 2, 3)" is AEXPR_IN with the "<>" operator. It is
		// safe (no nullable-subquery risk) so we do not flag it, but we still
		// count it so the Nth-"NOT IN" line mapping stays aligned with the text.
		if n.Kind == nodes.AEXPR_IN && isNotInOperator(n.Name) {
			*notInSeen++
		}
		r.walkExpr(n.Lexpr, sql, notInSeen, violations)
		r.walkExpr(n.Rexpr, sql, notInSeen, violations)
	case *nodes.SubLink:
		r.walkStmt(n.Subselect, sql, notInSeen, violations)
	}
}

// isNotInOperator reports whether an A_Expr operator name list is the "<>"
// operator used by NOT IN (as opposed to "=" used by IN).
func isNotInOperator(name *nodes.List) bool {
	if name == nil || len(name.Items) == 0 {
		return false
	}
	s, ok := name.Items[len(name.Items)-1].(*nodes.String)
	return ok && s.Str == "<>"
}
