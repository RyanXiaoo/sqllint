package rules
import "strings"

type TrailingSemicolon struct{}

func (r TrailingSemicolon) ID() string {
	return "trailing-semicolon"
}

func (r TrailingSemicolon) Check(sql string, lines []string) [] Violation {
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
