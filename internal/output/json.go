package output

import (
	"encoding/json"
	"io"

	"github.com/ryanxiao/go-sqllint/internal/linter"
)

// jsonViolation is the JSON-friendly representation of a lint violation.
// The struct tags (the `json:"..."` parts) control how Go marshals to JSON.
// This is like Pydantic field aliases but built into the language.
type jsonViolation struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// JSON writes lint results as a JSON array.
func JSON(w io.Writer, results []linter.Result) error {
	var out []jsonViolation

	for _, r := range results {
		for _, v := range r.Violations {
			out = append(out, jsonViolation{
				File:     r.File,
				Line:     v.Line,
				RuleID:   v.RuleID,
				Severity: v.Severity.String(),
				Message:  v.Message,
			})
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
