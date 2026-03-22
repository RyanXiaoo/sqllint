package rules

// Fixer is an optional interface a Rule may implement.
// Fix returns the full SQL string with the violation corrected.
// If nothing needs fixing, it returns sql unchanged.
type Fixer interface {
	Fix(sql string, lines []string) string
}
