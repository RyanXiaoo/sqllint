package rules

import (
	"strings"

	"github.com/pgplex/pgparser/nodes"
)

// ASTMissingGroupByCol flags SELECT columns that are neither in the GROUP BY
// clause nor wrapped in an aggregate function (COUNT, SUM, AVG, MIN, MAX).
type ASTMissingGroupByCol struct{}

func (r ASTMissingGroupByCol) ID() string {
	return "missing-group-by-col"
}

var aggregateFuncs = map[string]bool{
	"count": true, "sum": true, "avg": true,
	"min": true, "max": true, "array_agg": true,
	"string_agg": true, "bool_and": true, "bool_or": true,
	"every": true, "json_agg": true, "jsonb_agg": true,
}

func (r ASTMissingGroupByCol) CheckAST(stmts []nodes.Node, sql string, lines []string) []Violation {
	var violations []Violation

	// The parser does not populate node source locations. A column name can
	// recur across statements, so anchoring on the column identifier is
	// unreliable; instead we count SELECTs in document order (recursing into
	// subqueries) and report the offending statement's SELECT line.
	selectSeen := 0
	for _, stmt := range stmts {
		r.walkStmt(unwrapRawStmt(stmt), sql, &selectSeen, &violations)
	}

	return violations
}

func (r ASTMissingGroupByCol) walkStmt(node nodes.Node, sql string, selectSeen *int, violations *[]Violation) {
	if sel, ok := node.(*nodes.SelectStmt); ok {
		r.checkSelect(sel, sql, selectSeen, violations)
	}
}

func (r ASTMissingGroupByCol) checkSelect(sel *nodes.SelectStmt, sql string, selectSeen *int, violations *[]Violation) {
	if sel == nil {
		return
	}

	*selectSeen++
	occurrence := *selectSeen
	line := findKeywordLineMasked(sql, "SELECT", occurrence)

	r.checkTargets(sel, line, violations)

	// Descend into subqueries so nested SELECTs are checked and counted in
	// document order.
	if sel.FromClause != nil {
		for _, from := range sel.FromClause.Items {
			r.walkFromNode(from, sql, selectSeen, violations)
		}
	}
	r.walkExpr(sel.WhereClause, sql, selectSeen, violations)
}

func (r ASTMissingGroupByCol) checkTargets(sel *nodes.SelectStmt, line int, violations *[]Violation) {
	if sel.GroupClause == nil || sel.GroupClause.Len() == 0 || sel.TargetList == nil {
		return
	}

	groupCols := make(map[string]bool)
	for _, g := range sel.GroupClause.Items {
		col := r.extractColName(g)
		if col != "" {
			groupCols[col] = true
		}
	}

	for _, item := range sel.TargetList.Items {
		rt, ok := item.(*nodes.ResTarget)
		if !ok || rt.Val == nil {
			continue
		}

		if r.isAggregate(rt.Val) {
			continue
		}

		col := r.extractColName(rt.Val)
		if col != "" && !groupCols[col] {
			*violations = append(*violations, Violation{
				RuleID:   r.ID(),
				Message:  "Column \"" + col + "\" must appear in GROUP BY or be used in an aggregate function",
				Line:     line,
				Severity: SeverityError,
			})
		}
	}
}

func (r ASTMissingGroupByCol) walkFromNode(node nodes.Node, sql string, selectSeen *int, violations *[]Violation) {
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

func (r ASTMissingGroupByCol) walkExpr(node nodes.Node, sql string, selectSeen *int, violations *[]Violation) {
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

func (r ASTMissingGroupByCol) extractColName(node nodes.Node) string {
	if node == nil {
		return ""
	}

	ref, ok := node.(*nodes.ColumnRef)
	if !ok || ref.Fields == nil {
		return ""
	}

	items := ref.Fields.Items
	if len(items) > 0 {
		if s, ok := items[len(items)-1].(*nodes.String); ok {
			return strings.ToLower(s.Str)
		}
	}

	return ""
}

func (r ASTMissingGroupByCol) isAggregate(node nodes.Node) bool {
	if node == nil {
		return false
	}

	fc, ok := node.(*nodes.FuncCall)
	if !ok {
		return false
	}

	if fc.Funcname != nil {
		for _, name := range fc.Funcname.Items {
			if s, ok := name.(*nodes.String); ok {
				if aggregateFuncs[strings.ToLower(s.Str)] {
					return true
				}
			}
		}
	}

	return fc.AggStar
}
