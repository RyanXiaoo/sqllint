package rules

import "github.com/pgplex/pgparser/nodes"

type ASTSelectStar struct{}

func (r ASTSelectStar) ID() string {
	return "select-star"
}

func (r ASTSelectStar) CheckAST(stmts []nodes.Node, sql string, lines []string) []Violation {
	var violations []Violation
	selectCount := 0

	for _, stmt := range stmts {
		node := unwrapRawStmt(stmt)
		if sel, ok := node.(*nodes.SelectStmt); ok {
			selectCount++
			r.checkSelect(sel, sql, lines, selectCount, &violations)
		}
	}

	return violations
}

func (r ASTSelectStar) checkSelect(sel *nodes.SelectStmt, sql string, lines []string, occurrence int, violations *[]Violation) {
	if sel == nil || sel.TargetList == nil {
		return
	}

	for _, item := range sel.TargetList.Items {
		rt, ok := item.(*nodes.ResTarget)
		if !ok || rt.Val == nil {
			continue
		}

		ref, ok := rt.Val.(*nodes.ColumnRef)
		if !ok || ref.Fields == nil {
			continue
		}

		for _, field := range ref.Fields.Items {
			if _, ok := field.(*nodes.A_Star); ok {
				line := findKeywordLine(sql, "SELECT", occurrence)
				*violations = append(*violations, Violation{
					RuleID:   r.ID(),
					Message:  "Avoid SELECT *; explicitly list the columns you need",
					Line:     line,
					Severity: SeverityWarning,
				})
			}
		}
	}

	if sel.FromClause != nil {
		for _, from := range sel.FromClause.Items {
			r.walkFromNode(from, sql, violations)
		}
	}

	r.walkExpr(sel.WhereClause, sql, violations)
}

func (r ASTSelectStar) walkFromNode(node nodes.Node, sql string, violations *[]Violation) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *nodes.RangeSubselect:
		if n.Subquery != nil {
			if sel, ok := n.Subquery.(*nodes.SelectStmt); ok {
				r.checkSelect(sel, sql, nil, 1, violations)
			}
		}
	case *nodes.JoinExpr:
		r.walkFromNode(n.Larg, sql, violations)
		r.walkFromNode(n.Rarg, sql, violations)
	}
}

func (r ASTSelectStar) walkExpr(node nodes.Node, sql string, violations *[]Violation) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *nodes.SubLink:
		if sel, ok := n.Subselect.(*nodes.SelectStmt); ok {
			r.checkSelect(sel, sql, nil, 1, violations)
		}
	case *nodes.BoolExpr:
		if n.Args != nil {
			for _, arg := range n.Args.Items {
				r.walkExpr(arg, sql, violations)
			}
		}
	case *nodes.A_Expr:
		r.walkExpr(n.Lexpr, sql, violations)
		r.walkExpr(n.Rexpr, sql, violations)
	}
}
