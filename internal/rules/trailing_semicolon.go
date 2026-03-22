package rules
import "strings"

type TrailingSemicolon struct{}

func (r TrailingSemicolon) ID() string {
	return "trailing-semicolon"
}

func (r TrailingSemicolon) Check(sql string, lines []string) []Violation {
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		} else if strings.HasSuffix(line, ";") {
			return nil
		} else {
			return []Violation{{
				RuleID:   r.ID(),
				Message:  "Missing ; on last line",
				Line:     i + 1,
				Severity: SeverityWarning,
			}}
		}
	}
	return nil
}

// Fix appends a semicolon to the last non-empty line if missing.
func (r TrailingSemicolon) Fix(sql string, lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		if !strings.HasSuffix(trimmed, ";") {
			// Append ; to the raw line, preserving any trailing whitespace
			trailing := len(lines[i]) - len(strings.TrimRight(lines[i], " \t"))
			lines[i] = strings.TrimRight(lines[i], " \t") + ";" + lines[i][len(lines[i])-trailing:]
		}
		break
	}
	return strings.Join(lines, "\n")
}
