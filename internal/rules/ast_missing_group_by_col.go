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

	for _, stmt := range stmts {
		node := unwrapRawStmt(stmt)
		if sel, ok := node.(*nodes.SelectStmt); ok {
			r.checkSelect(sel, sql, &violations)
		}
	}

	return violations
}

func (r ASTMissingGroupByCol) checkSelect(sel *nodes.SelectStmt, sql string, violations *[]Violation) {
	if sel == nil || sel.GroupClause == nil || sel.GroupClause.Len() == 0 {
		return
	}

	groupCols := make(map[string]bool)
	for _, g := range sel.GroupClause.Items {
		col := r.extractColName(g)
		if col != "" {
			groupCols[col] = true
		}
	}

	if sel.TargetList == nil {
		return
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
				Line:     locationToLine(sql, rt.Location),
				Severity: SeverityError,
			})
		}
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
