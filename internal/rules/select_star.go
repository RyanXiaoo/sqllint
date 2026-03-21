package rules

import "strings"

// SelectStar flags any SELECT * usage.
// Notice: no "class" keyword, no inheritance. Just a struct that happens
// to have the methods that satisfy the Rule interface.
type SelectStar struct{}

func (r SelectStar) ID() string {
	return "select-star"
}

func (r SelectStar) Check(sql string, lines []string) []Violation {
	var violations []Violation // nil slice — like an empty list in Python

	for i, line := range lines {
		upper := strings.ToUpper(strings.TrimSpace(line))

		// Skip SQL comments
		if strings.HasPrefix(upper, "--") {
			continue
		}

		if strings.Contains(upper, "SELECT *") || strings.Contains(upper, "SELECT  *") {
			violations = append(violations, Violation{
				RuleID:   r.ID(),
				Message:  "Avoid SELECT *; explicitly list the columns you need",
				Line:     i + 1, // 1-indexed for human readability
				Severity: SeverityWarning,
			})
		}
	}

	return violations
}
