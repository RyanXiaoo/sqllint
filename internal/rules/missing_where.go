package rules

import "strings"

// MissingWhere flags DELETE or UPDATE statements that don't have a WHERE clause.
// This is a simple line-range heuristic for Phase 1 — Phase 2 replaces it with AST walking.
type MissingWhere struct{}

func (r MissingWhere) ID() string {
	return "missing-where"
}

func (r MissingWhere) Check(sql string, lines []string) []Violation {
	var violations []Violation

	// Join and split on semicolons to get individual statements.
	// This is intentionally naive — it'll break on semicolons inside strings.
	// That's fine for Phase 1; real parsing comes in Phase 2.
	full := strings.Join(lines, "\n")
	statements := strings.Split(full, ";")

	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}

		upper := strings.ToUpper(trimmed)

		isDangerous := strings.HasPrefix(upper, "DELETE") || strings.HasPrefix(upper, "UPDATE")
		if !isDangerous {
			continue
		}

		if !strings.Contains(upper, "WHERE") {
			// Find which line this statement starts on
			lineNum := findLineNumber(full, stmt)
			violations = append(violations, Violation{
				RuleID:   r.ID(),
				Message:  "DELETE/UPDATE without WHERE clause affects all rows",
				Line:     lineNum,
				Severity: SeverityError,
			})
		}
	}

	return violations
}

// findLineNumber returns the 1-indexed line number where substr first appears in text.
func findLineNumber(text, substr string) int {
	idx := strings.Index(text, substr)
	if idx < 0 {
		return 1
	}
	return strings.Count(text[:idx], "\n") + 1
}
