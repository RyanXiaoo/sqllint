package rules

import (
	"strings"

	"github.com/pgplex/pgparser/nodes"
)

// ASTUnusedAlias flags table aliases that are defined in FROM/JOIN
// but never referenced anywhere in the query.
type ASTUnusedAlias struct{}

func (r ASTUnusedAlias) ID() string {
	return "unused-alias"
}

func (r ASTUnusedAlias) CheckAST(stmts []nodes.Node, sql string, lines []string) []Violation {
	var violations []Violation

	// The parser does not populate node source locations. An alias name can
	// recur across statements, so anchoring on the alias identifier is
	// unreliable; instead we count SELECTs in document order and report the
	// offending statement's SELECT line.
	selectSeen := 0
	for _, stmt := range stmts {
		if sel, ok := unwrapRawStmt(stmt).(*nodes.SelectStmt); ok {
			r.checkSelect(sel, sql, &selectSeen, &violations)
		}
	}

	return violations
}

func (r ASTUnusedAlias) checkSelect(sel *nodes.SelectStmt, sql string, selectSeen *int, violations *[]Violation) {
	if sel == nil {
		return
	}

	*selectSeen++
	line := findKeywordLineMasked(sql, "SELECT", *selectSeen)

	r.checkAliases(sel, line, violations)

	// Always recurse into subqueries so nested SELECTs are checked and counted
	// in document order, keeping the SELECT line numbering aligned.
	if sel.FromClause != nil {
		for _, from := range sel.FromClause.Items {
			r.walkFromNode(from, sql, selectSeen, violations)
		}
	}
	r.walkExpr(sel.WhereClause, sql, selectSeen, violations)
}

func (r ASTUnusedAlias) checkAliases(sel *nodes.SelectStmt, line int, violations *[]Violation) {
	if sel.FromClause == nil {
		return
	}

	var aliases []string
	for _, from := range sel.FromClause.Items {
		r.collectAliases(from, &aliases)
	}

	if len(aliases) < 2 {
		return
	}

	refs := make(map[string]bool)
	if sel.TargetList != nil {
		for _, target := range sel.TargetList.Items {
			r.collectRefs(target, refs)
		}
	}
	r.collectRefs(sel.WhereClause, refs)
	r.collectRefs(sel.HavingClause, refs)
	if sel.GroupClause != nil {
		for _, g := range sel.GroupClause.Items {
			r.collectRefs(g, refs)
		}
	}
	if sel.SortClause != nil {
		for _, s := range sel.SortClause.Items {
			r.collectRefs(s, refs)
		}
	}

	for _, alias := range aliases {
		if !refs[strings.ToLower(alias)] {
			*violations = append(*violations, Violation{
				RuleID:   r.ID(),
				Message:  "Table alias \"" + alias + "\" is defined but never referenced",
				Line:     line,
				Severity: SeverityWarning,
			})
		}
	}
}

func (r ASTUnusedAlias) walkFromNode(node nodes.Node, sql string, selectSeen *int, violations *[]Violation) {
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

func (r ASTUnusedAlias) walkExpr(node nodes.Node, sql string, selectSeen *int, violations *[]Violation) {
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

func (r ASTUnusedAlias) collectAliases(node nodes.Node, aliases *[]string) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *nodes.RangeVar:
		if n.Alias != nil && n.Alias.Aliasname != "" {
			*aliases = append(*aliases, n.Alias.Aliasname)
		}
	case *nodes.JoinExpr:
		r.collectAliases(n.Larg, aliases)
		r.collectAliases(n.Rarg, aliases)
		if n.Alias != nil && n.Alias.Aliasname != "" {
			*aliases = append(*aliases, n.Alias.Aliasname)
		}
	}
}

func (r ASTUnusedAlias) collectRefs(node nodes.Node, refs map[string]bool) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *nodes.ColumnRef:
		if n.Fields != nil && len(n.Fields.Items) >= 2 {
			if str, ok := n.Fields.Items[0].(*nodes.String); ok {
				refs[strings.ToLower(str.Str)] = true
			}
		}
	case *nodes.ResTarget:
		r.collectRefs(n.Val, refs)
	case *nodes.A_Expr:
		r.collectRefs(n.Lexpr, refs)
		r.collectRefs(n.Rexpr, refs)
	case *nodes.BoolExpr:
		if n.Args != nil {
			for _, arg := range n.Args.Items {
				r.collectRefs(arg, refs)
			}
		}
	case *nodes.FuncCall:
		if n.Args != nil {
			for _, arg := range n.Args.Items {
				r.collectRefs(arg, refs)
			}
		}
	case *nodes.SubLink:
		r.collectRefs(n.Testexpr, refs)
	case *nodes.SortBy:
		r.collectRefs(n.Node, refs)
	}
}
