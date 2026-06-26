package rules

import (
	"strings"

	"github.com/pgplex/pgparser/nodes"
)

// unwrapRawStmt returns the inner statement if the node is a RawStmt,
// otherwise returns the node itself.
func unwrapRawStmt(node nodes.Node) nodes.Node {
	if raw, ok := node.(*nodes.RawStmt); ok && raw.Stmt != nil {
		return raw.Stmt
	}
	return node
}

// findKeywordLine finds the 1-indexed line number of the Nth occurrence
// of a keyword in the SQL string. This is the parser version pgparser ships
// does not populate node source locations, so rules resolve lines by searching
// the text instead.
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

// findKeywordLineMasked is like findKeywordLine but ignores occurrences inside
// comments and string literals, so e.g. a "SELECT" mentioned in a "-- comment"
// is not counted. Masking preserves length and newline positions, so the
// resulting line number still refers to the original SQL.
func findKeywordLineMasked(sql, keyword string, occurrence int) int {
	return findKeywordLine(maskNonCode(sql, true), keyword, occurrence)
}

// maskNonCode replaces the contents of line comments (-- ...), block comments
// (/* ... */) and, when maskStrings is true, single-quoted string literals with
// spaces. Newlines and overall length are preserved so byte offsets and line
// numbers map back to the original text unchanged. String literals are always
// scanned (even when not masked) so that a "--" or quote inside a string is not
// mistaken for the start of a comment.
func maskNonCode(sql string, maskStrings bool) string {
	out := []byte(sql)
	n := len(out)
	blank := func(i int) {
		if out[i] != '\n' {
			out[i] = ' '
		}
	}
	for i := 0; i < n; {
		switch {
		case out[i] == '-' && i+1 < n && out[i+1] == '-':
			for i < n && out[i] != '\n' {
				blank(i)
				i++
			}
		case out[i] == '/' && i+1 < n && out[i+1] == '*':
			blank(i)
			blank(i + 1)
			i += 2
			for i < n && !(out[i] == '*' && i+1 < n && out[i+1] == '/') {
				blank(i)
				i++
			}
			if i < n {
				blank(i)
				if i+1 < n {
					blank(i + 1)
				}
				i += 2
			}
		case out[i] == '\'':
			if maskStrings {
				blank(i)
			}
			i++
			for i < n {
				if out[i] == '\'' {
					if i+1 < n && out[i+1] == '\'' { // escaped '' inside string
						if maskStrings {
							blank(i)
							blank(i + 1)
						}
						i += 2
						continue
					}
					if maskStrings {
						blank(i)
					}
					i++
					break
				}
				if maskStrings {
					blank(i)
				}
				i++
			}
		default:
			i++
		}
	}
	return string(out)
}
