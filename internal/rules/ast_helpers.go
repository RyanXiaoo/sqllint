package rules

import (
	"strings"

	"github.com/pgplex/pgparser/nodes"
)

// locationToLine converts a byte offset from the PostgreSQL parser
// into a 1-indexed line number within the SQL string.
func locationToLine(sql string, loc nodes.ParseLoc) int {
	if int(loc) <= 0 || int(loc) > len(sql) {
		return 1
	}
	return strings.Count(sql[:int(loc)], "\n") + 1
}

// unwrapRawStmt returns the inner statement if the node is a RawStmt,
// otherwise returns the node itself.
func unwrapRawStmt(node nodes.Node) nodes.Node {
	if raw, ok := node.(*nodes.RawStmt); ok && raw.Stmt != nil {
		return raw.Stmt
	}
	return node
}

// findKeywordLine finds the 1-indexed line number of the Nth occurrence
// of a keyword in the SQL string. This is used as a fallback when the
// parser doesn't provide location data.
func findKeywordLine(sql string, keyword string, occurrence int) int {
	upper := strings.ToUpper(sql)
	kw := strings.ToUpper(keyword)
	idx := 0
	for n := 0; n < occurrence; n++ {
		pos := strings.Index(upper[idx:], kw)
		if pos < 0 {
			return 1
		}
		idx += pos
		if n < occurrence-1 {
			idx += len(kw)
		}
	}
	return strings.Count(sql[:idx], "\n") + 1
}
