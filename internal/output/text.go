package output

import (
	"fmt"
	"io"

	"github.com/ryanxiao/go-sqllint/internal/linter"
)

// Text writes lint results in a human-readable format like:
//   query.sql:12 [warning] select-star: Avoid SELECT *; explicitly list the columns you need
func Text(w io.Writer, results []linter.Result) {
	count := 0
	for _, r := range results {
		for _, v := range r.Violations {
			fmt.Fprintf(w, "%s:%d [%s] %s: %s\n",
				r.File, v.Line, v.Severity, v.RuleID, v.Message,
			)
			count++
		}
	}

	if count == 0 {
		fmt.Fprintln(w, "No issues found.")
	} else {
		fmt.Fprintf(w, "\n%d issue(s) found.\n", count)
	}
}
