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

	for _, stmt := range stmts {
		node := unwrapRawStmt(stmt)
		sel, ok := node.(*nodes.SelectStmt)
		if !ok {
			continue
		}
		r.checkSelect(sel, sql, &violations)
	}

	return violations
}

type aliasInfo struct {
	name     string
	location nodes.ParseLoc
}

func (r ASTUnusedAlias) checkSelect(sel *nodes.SelectStmt, sql string, violations *[]Violation) {
	if sel == nil || sel.FromClause == nil {
		return
	}

	var aliases []aliasInfo
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
		if !refs[strings.ToLower(alias.name)] {
			*violations = append(*violations, Violation{
				RuleID:   r.ID(),
				Message:  "Table alias \"" + alias.name + "\" is defined but never referenced",
				Line:     locationToLine(sql, alias.location),
				Severity: SeverityWarning,
			})
		}
	}
}

func (r ASTUnusedAlias) collectAliases(node nodes.Node, aliases *[]aliasInfo) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *nodes.RangeVar:
		if n.Alias != nil && n.Alias.Aliasname != "" {
			*aliases = append(*aliases, aliasInfo{
				name:     n.Alias.Aliasname,
				location: n.Location,
			})
		}
	case *nodes.JoinExpr:
		r.collectAliases(n.Larg, aliases)
		r.collectAliases(n.Rarg, aliases)
		if n.Alias != nil && n.Alias.Aliasname != "" {
			*aliases = append(*aliases, aliasInfo{
				name:     n.Alias.Aliasname,
				location: nodes.ParseLoc(n.Rtindex),
			})
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
