package rules

// Severity represents how serious a lint violation is.
type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
)

// String returns the human-readable name of a severity level.
// This is your first Go "method on a type" — similar to __str__ in Python,
// but attached to any type, not just classes.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	default:
		return "warning"
	}
}

// Violation represents a single lint finding.
type Violation struct {
	RuleID   string   // e.g. "select-star"
	Message  string   // human-readable explanation
	Line     int      // 1-indexed line number
	Severity Severity // warning or error
}

// Rule is the interface every lint rule must implement.
// This is the Go equivalent of an abstract base class in Python.
// Any struct that has these two methods automatically satisfies the interface —
// no "implements" keyword needed. This is called "structural typing."
type Rule interface {
	// ID returns a unique short identifier for the rule (e.g. "select-star").
	ID() string

	// Check takes the full SQL string and its lines, and returns any violations found.
	// We pass both the raw SQL and pre-split lines so rules can work with either.
	Check(sql string, lines []string) []Violation
}
