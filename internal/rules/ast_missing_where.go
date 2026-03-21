package rules

import "github.com/pgplex/pgparser/nodes"

type ASTMissingWhere struct{}

func (r ASTMissingWhere) ID() string {
	return "missing-where"
}

func (r ASTMissingWhere) CheckAST(stmts []nodes.Node, sql string, lines []string) []Violation {
	var violations []Violation
	deleteCount := 0
	updateCount := 0

	for _, stmt := range stmts {
		node := unwrapRawStmt(stmt)

		switch n := node.(type) {
		case *nodes.DeleteStmt:
			deleteCount++
			if n.WhereClause == nil {
				violations = append(violations, Violation{
					RuleID:   r.ID(),
					Message:  "DELETE without WHERE clause affects all rows",
					Line:     findKeywordLine(sql, "DELETE", deleteCount),
					Severity: SeverityError,
				})
			}
		case *nodes.UpdateStmt:
			updateCount++
			if n.WhereClause == nil {
				violations = append(violations, Violation{
					RuleID:   r.ID(),
					Message:  "UPDATE without WHERE clause affects all rows",
					Line:     findKeywordLine(sql, "UPDATE", updateCount),
					Severity: SeverityError,
				})
			}
		}
	}

	return violations
}
