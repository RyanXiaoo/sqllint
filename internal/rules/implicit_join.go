package rules

import "strings"

type ImplicitJoin struct{}

func (r ImplicitJoin) ID() string {
	return "implicit-join"
}

func (r ImplicitJoin) Check(sql string, lines []string) []Violation {
	var violations []Violation

	for i, line := range lines {
		upper := strings.ToUpper(line)
		fromIdx := strings.Index(upper, "FROM")

		if fromIdx >= 0 {
			afterFrom := line[fromIdx+4:]
			if strings.Contains(afterFrom, ",") {
				violations = append(violations, Violation{
					RuleID:   r.ID(),
					Message:  "Avoid implicit joins (comma-separated FROM tables); use explicit JOIN instead",
					Line:     i + 1,
					Severity: SeverityWarning,
				})
			}
		}
	}
	return violations
}