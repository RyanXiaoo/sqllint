package rules
import "strings"

type LeadingWildcard struct{}

func (r LeadingWildcard) ID() string {
	return "leading-wildcard"
}

func (r LeadingWildcard) Check(sql string, lines []string) []Violation {
	var violations []Violation

	for i, line := range lines {
		line = strings.ToUpper(line)
		if (strings.Contains(line, "LIKE") || strings.Contains(line, "ILIKE")) &&  strings.Contains(line, "'%") {
			violations = append(violations, Violation {
				RuleID:   r.ID(),
				Message:  "Contains leading wildcard",
				Line:     i + 1, // 1-indexed for human readability
				Severity: SeverityWarning,
			})
		}
	}

	return violations
}