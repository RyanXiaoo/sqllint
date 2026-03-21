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

	for _, stmt := range stmts {
		node := unwrapRawStmt(stmt)
		r.walkStmt(node, sql, &violations)
	}

	return violations
}

func (r ASTNotInNullable) walkStmt(node nodes.Node, sql string, violations *[]Violation) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *nodes.SelectStmt:
		r.walkExpr(n.WhereClause, sql, violations)
		r.walkExpr(n.HavingClause, sql, violations)
	case *nodes.DeleteStmt:
		r.walkExpr(n.WhereClause, sql, violations)
	case *nodes.UpdateStmt:
		r.walkExpr(n.WhereClause, sql, violations)
	}
}

func (r ASTNotInNullable) walkExpr(node nodes.Node, sql string, violations *[]Violation) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *nodes.BoolExpr:
		if n.Boolop == nodes.NOT_EXPR && n.Args != nil {
			for _, arg := range n.Args.Items {
				if sub, ok := arg.(*nodes.SubLink); ok {
					if nodes.SubLinkType(sub.SubLinkType) == nodes.ANY_SUBLINK {
						*violations = append(*violations, Violation{
							RuleID:   r.ID(),
							Message:  "NOT IN (subquery) returns no rows if any subquery value is NULL; use NOT EXISTS instead",
							Line:     locationToLine(sql, sub.Location),
							Severity: SeverityError,
						})
					}
				}
			}
		}
		if n.Args != nil {
			for _, arg := range n.Args.Items {
				r.walkExpr(arg, sql, violations)
			}
		}
	case *nodes.A_Expr:
		r.walkExpr(n.Lexpr, sql, violations)
		r.walkExpr(n.Rexpr, sql, violations)
	case *nodes.SubLink:
		r.walkStmt(n.Subselect, sql, violations)
	}
}
