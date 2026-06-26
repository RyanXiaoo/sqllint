package rules_test

import (
	"testing"

	"github.com/pgplex/pgparser/parser"
	"github.com/ryanxiao/sqllint/internal/rules"
)

// runAST parses sql and runs an AST rule over the parsed statements,
// mirroring how internal/linter wires AST rules in production.
func runAST(t *testing.T, rule rules.ASTRule, sql string) int {
	t.Helper()
	return len(runASTViolations(t, rule, sql))
}

func runASTViolations(t *testing.T, rule rules.ASTRule, sql string) []rules.Violation {
	t.Helper()
	tree, err := parser.Parse(sql)
	if err != nil {
		t.Fatalf("parse error for %q: %v", sql, err)
	}
	return rule.CheckAST(tree.Items, sql, splitLines(sql))
}

func TestASTSelectStar(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{"flags SELECT *", "SELECT * FROM users;", 1},
		{"allows explicit columns", "SELECT id, name FROM users;", 0},
		{"allows COUNT(*)", "SELECT COUNT(*) FROM users;", 0},
		{"flags lowercase select *", "select * from users;", 1},
		{"flags SELECT * in subquery", "SELECT id FROM (SELECT * FROM users) sub;", 1},
		{"catches multiple occurrences", "SELECT * FROM users;\nSELECT * FROM orders;", 2},
	}

	rule := rules.ASTSelectStar{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runAST(t, rule, tt.sql); got != tt.want {
				t.Errorf("got %d violations, want %d", got, tt.want)
			}
		})
	}
}

func TestASTMissingWhere(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{"flags DELETE without WHERE", "DELETE FROM sessions;", 1},
		{"flags UPDATE without WHERE", "UPDATE users SET active = 0;", 1},
		{"allows DELETE with WHERE", "DELETE FROM sessions WHERE expired = true;", 0},
		{"allows UPDATE with WHERE", "UPDATE users SET active = 0 WHERE id = 1;", 0},
		{"ignores SELECT", "SELECT id FROM users;", 0},
		{"ignores INSERT", "INSERT INTO users (id, name) VALUES (1, 'test');", 0},
		{"flags multi-line DELETE without WHERE", "DELETE\nFROM sessions;", 1},
	}

	rule := rules.ASTMissingWhere{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runAST(t, rule, tt.sql); got != tt.want {
				t.Errorf("got %d violations, want %d", got, tt.want)
			}
		})
	}
}

func TestASTNotInNullable(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{
			"flags NOT IN (subquery)",
			"SELECT id FROM users WHERE id NOT IN (SELECT user_id FROM banned_users);",
			1,
		},
		{
			"allows NOT EXISTS",
			"SELECT id FROM users WHERE NOT EXISTS (SELECT 1 FROM banned_users WHERE banned_users.user_id = users.id);",
			0,
		},
		{
			"allows plain IN (subquery)",
			"SELECT id FROM users WHERE id IN (SELECT user_id FROM banned_users);",
			0,
		},
	}

	rule := rules.ASTNotInNullable{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runAST(t, rule, tt.sql); got != tt.want {
				t.Errorf("got %d violations, want %d", got, tt.want)
			}
		})
	}
}

func TestASTUnusedAlias(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{
			"flags alias referenced only in JOIN ON",
			"SELECT u.id, u.name\nFROM users AS u\nJOIN orders AS o ON u.id = o.user_id\nWHERE u.active = 1;",
			1,
		},
		{
			"allows aliases referenced in target list",
			"SELECT u.id, o.total\nFROM users AS u\nJOIN orders AS o ON u.id = o.user_id;",
			0,
		},
	}

	rule := rules.ASTUnusedAlias{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runAST(t, rule, tt.sql); got != tt.want {
				t.Errorf("got %d violations, want %d", got, tt.want)
			}
		})
	}
}

func TestASTImplicitJoin(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{"flags comma-separated FROM tables", "SELECT * FROM users, orders WHERE users.id = orders.user_id;", 1},
		{"flags three comma-separated tables", "SELECT * FROM users, orders, products;", 1},
		{"allows explicit JOIN", "SELECT * FROM users JOIN orders ON users.id = orders.user_id;", 0},
		{"allows single table", "SELECT * FROM users;", 0},
		// Cases the old line-based rule got wrong:
		{"flags multi-line comma join", "SELECT u.id, o.total\nFROM users u,\n     orders o\nWHERE u.id = o.user_id;", 1},
		{"does not flag comma inside IN list", "SELECT id FROM users WHERE id IN (1, 2, 3);", 0},
		{"does not flag comma in function args", "SELECT coalesce(a, b) FROM users;", 0},
	}

	rule := rules.ASTImplicitJoin{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runAST(t, rule, tt.sql); got != tt.want {
				t.Errorf("got %d violations, want %d", got, tt.want)
			}
		})
	}
}

func TestASTLeadingWildcard(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{"flags LIKE with leading wildcard", "SELECT id FROM users WHERE name LIKE '%foo';", 1},
		{"flags ILIKE with leading wildcard", "SELECT id FROM users WHERE name ILIKE '%foo';", 1},
		{"flags leading and trailing wildcard", "SELECT id FROM users WHERE name LIKE '%foo%';", 1},
		{"allows trailing wildcard only", "SELECT id FROM users WHERE name LIKE 'foo%';", 0},
		{"allows no wildcard", "SELECT id FROM users WHERE name LIKE 'foo';", 0},
		// Cases the old substring rule got wrong:
		{"does not flag '%' inside a non-LIKE string literal", "SELECT id FROM users WHERE note = 'progress: 50%';", 0},
		{"flags multi-line LIKE pattern", "SELECT id\nFROM users\nWHERE name LIKE '%foo';", 1},
	}

	rule := rules.ASTLeadingWildcard{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runAST(t, rule, tt.sql); got != tt.want {
				t.Errorf("got %d violations, want %d", got, tt.want)
			}
		})
	}
}

// TestASTMigratedRuleLines guards against the line numbers regressing to 1:
// the parser does not emit source locations, so these rules resolve lines by
// searching the text. Each line below is the actual line of the offending SQL.
func TestASTMigratedRuleLines(t *testing.T) {
	t.Run("implicit-join reports the start line of each offending statement", func(t *testing.T) {
		// A DELETE ... FROM sits between the two implicit joins to prove the
		// line resolution is not thrown off by a non-SELECT FROM keyword.
		sql := "SELECT u.id, o.total\nFROM users u, orders o\nWHERE u.id = o.user_id;\n\nDELETE FROM logs;\n\nSELECT a.id\nFROM accounts a, ledgers l\nWHERE a.id = l.account_id;"
		got := runASTViolations(t, rules.ASTImplicitJoin{}, sql)
		wantLines := []int{1, 7}
		if len(got) != len(wantLines) {
			t.Fatalf("got %d violations, want %d", len(got), len(wantLines))
		}
		for i, w := range wantLines {
			if got[i].Line != w {
				t.Errorf("violation %d line = %d, want %d", i, got[i].Line, w)
			}
		}
	})

	t.Run("leading-wildcard reports the pattern line of each statement", func(t *testing.T) {
		sql := "SELECT id\nFROM users\nWHERE name LIKE '%john';\n\nSELECT id\nFROM products\nWHERE descr ILIKE '%widget%';"
		got := runASTViolations(t, rules.ASTLeadingWildcard{}, sql)
		wantLines := []int{3, 7}
		if len(got) != len(wantLines) {
			t.Fatalf("got %d violations, want %d", len(got), len(wantLines))
		}
		for i, w := range wantLines {
			if got[i].Line != w {
				t.Errorf("violation %d line = %d, want %d", i, got[i].Line, w)
			}
		}
	})
}

// TestASTLineResolutionAcrossFixtures locks in correct line numbers for the
// remaining AST rules. The parser emits no source locations, and these inputs
// deliberately repeat identifiers and include subqueries/comments — the cases
// that naive text anchoring gets wrong.
func TestASTLineResolutionAcrossFixtures(t *testing.T) {
	t.Run("not-in-nullable past a comment and a safe literal NOT IN", func(t *testing.T) {
		// Line 4 has a literal NOT IN (safe); line 6 has the flagged subquery.
		sql := "-- a NOT IN mentioned in a comment\nSELECT id\nFROM users\nWHERE tag NOT IN ('a', 'b')\n  AND active\n  AND id NOT IN (SELECT user_id FROM banned);"
		got := runASTViolations(t, rules.ASTNotInNullable{}, sql)
		if len(got) != 1 {
			t.Fatalf("got %d violations, want 1", len(got))
		}
		if got[0].Line != 6 {
			t.Errorf("line = %d, want 6", got[0].Line)
		}
	})

	t.Run("missing-group-by-col after a subquery and a recurring column name", func(t *testing.T) {
		// "name" also appears on line 1; the offending statement starts line 5.
		sql := "SELECT id, name\nFROM users\nWHERE id IN (SELECT user_id FROM banned);\n\nSELECT dept, name, count(*)\nFROM employees\nGROUP BY dept;"
		got := runASTViolations(t, rules.ASTMissingGroupByCol{}, sql)
		if len(got) != 1 {
			t.Fatalf("got %d violations, want 1", len(got))
		}
		if got[0].Line != 5 {
			t.Errorf("line = %d, want 5", got[0].Line)
		}
	})

	t.Run("unused-alias after a subquery-bearing statement", func(t *testing.T) {
		// The unused alias is in the second statement, which starts on line 5.
		sql := "SELECT id\nFROM users\nWHERE id IN (SELECT user_id FROM banned);\n\nSELECT u.id\nFROM users u\nJOIN orders o ON u.id = o.user_id;"
		got := runASTViolations(t, rules.ASTUnusedAlias{}, sql)
		if len(got) != 1 {
			t.Fatalf("got %d violations, want 1", len(got))
		}
		if got[0].Line != 5 {
			t.Errorf("line = %d, want 5", got[0].Line)
		}
	})
}

func TestASTMissingGroupByCol(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{
			"flags non-grouped non-aggregate column",
			"SELECT department, name, COUNT(*)\nFROM employees\nGROUP BY department;",
			1,
		},
		{
			"allows grouped column with aggregate",
			"SELECT department, COUNT(*) AS total\nFROM employees\nGROUP BY department;",
			0,
		},
		{
			"ignores queries without GROUP BY",
			"SELECT id, name FROM users;",
			0,
		},
	}

	rule := rules.ASTMissingGroupByCol{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runAST(t, rule, tt.sql); got != tt.want {
				t.Errorf("got %d violations, want %d", got, tt.want)
			}
		})
	}
}
