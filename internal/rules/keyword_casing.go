package rules

import (
	"fmt"
	"strings"
)

// keywords we check for consistent casing.
var sqlKeywords = []string{
	"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE",
	"JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "ON",
	"GROUP BY", "ORDER BY", "HAVING", "LIMIT", "OFFSET",
	"CREATE", "ALTER", "DROP", "TABLE", "INDEX",
	"AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE",
	"SET", "VALUES", "INTO", "AS", "DISTINCT", "UNION",
}

// KeywordCasing flags SQL keywords that aren't consistently uppercased or lowercased.
type KeywordCasing struct{}

func (r KeywordCasing) ID() string {
	return "keyword-casing"
}

func (r KeywordCasing) Check(sql string, lines []string) []Violation {
	var violations []Violation

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "--") {
			continue
		}

		for _, kw := range sqlKeywords {
			// Find case-insensitive occurrences of the keyword as a whole word
			lower := strings.ToLower(line)
			kwLower := strings.ToLower(kw)

			idx := 0
			for {
				pos := strings.Index(lower[idx:], kwLower)
				if pos < 0 {
					break
				}
				absPos := idx + pos
				endPos := absPos + len(kw)

				if endPos > len(line) {
					break
				}

				before := absPos == 0 || !isIdentChar(line[absPos-1])
				after := endPos >= len(line) || !isIdentChar(line[endPos])

				if before && after {
					actual := line[absPos:endPos]
					// Flag if it's neither fully upper nor fully lower
					if actual != strings.ToUpper(actual) && actual != strings.ToLower(actual) {
						violations = append(violations, Violation{
							RuleID:   r.ID(),
							Message:  fmt.Sprintf("Inconsistent keyword casing: %q (use %s or %s)", actual, strings.ToUpper(kw), strings.ToLower(kw)),
							Line:     i + 1,
							Severity: SeverityWarning,
						})
					}
				}

				idx = absPos + len(kw)
			}
		}
	}

	return violations
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// Fix replaces all mixed-case SQL keywords with their uppercase form.
func (r KeywordCasing) Fix(sql string, lines []string) string {
	result := make([]string, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			result[i] = line
			continue
		}
		fixed := line
		lower := strings.ToLower(line)
		for _, kw := range sqlKeywords {
			kwLower := strings.ToLower(kw)
			idx := 0
			for {
				pos := strings.Index(lower[idx:], kwLower)
				if pos < 0 {
					break
				}
				absPos := idx + pos
				endPos := absPos + len(kw)
				if endPos > len(fixed) {
					break
				}
				before := absPos == 0 || !isIdentChar(fixed[absPos-1])
				after := endPos >= len(fixed) || !isIdentChar(fixed[endPos])
				if before && after {
					actual := fixed[absPos:endPos]
					if actual != strings.ToUpper(actual) && actual != strings.ToLower(actual) {
						fixed = fixed[:absPos] + strings.ToUpper(kw) + fixed[endPos:]
						lower = strings.ToLower(fixed)
					}
				}
				idx = absPos + len(kw)
			}
		}
		result[i] = fixed
	}
	return strings.Join(result, "\n")
}
