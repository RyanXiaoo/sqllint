package rules_test

import (
	"testing"

	"github.com/ryanxiao/go-sqllint/internal/rules"
)

// Table-driven tests are THE Go testing pattern. Instead of writing
// separate test functions for each case, you define a slice of test cases
// and loop over them. It's like @pytest.mark.parametrize but idiomatic.

func TestSelectStar(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{
			name: "flags SELECT *",
			sql:  "SELECT * FROM users;",
			want: 1,
		},
		{
			name: "allows explicit columns",
			sql:  "SELECT id, name FROM users;",
			want: 0,
		},
		{
			name: "skips comments",
			sql:  "-- SELECT * FROM users;",
			want: 0,
		},
		{
			name: "catches multiple occurrences",
			sql:  "SELECT * FROM users;\nSELECT * FROM orders;",
			want: 2,
		},
		{
			name: "flags lowercase select *",
			sql:  "select * from users;",
			want: 1,
		},
		{
			name: "allows COUNT(*)",
			sql:  "SELECT COUNT(*) FROM users;",
			want: 0,
		},
		{
			name: "flags SELECT * in subquery",
			sql:  "SELECT id FROM (SELECT * FROM users) sub;",
			want: 1,
		},
		{
			name: "flags SELECT with extra spaces before *",
			sql:  "SELECT  * FROM users;",
			want: 1,
		},
	}

	rule := rules.SelectStar{}
	for _, tt := range tests {
		// t.Run creates a subtest — each case shows up separately in output.
		// The tt := tt line is a Go gotcha — it captures the loop variable.
		// Without it, closures might reference the wrong iteration.
		t.Run(tt.name, func(t *testing.T) {
			lines := splitLines(tt.sql)
			got := rule.Check(tt.sql, lines)
			if len(got) != tt.want {
				t.Errorf("got %d violations, want %d", len(got), tt.want)
			}
		})
	}
}

func TestMissingWhere(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{
			name: "flags DELETE without WHERE",
			sql:  "DELETE FROM sessions;",
			want: 1,
		},
		{
			name: "flags UPDATE without WHERE",
			sql:  "UPDATE users SET active = 0;",
			want: 1,
		},
		{
			name: "allows DELETE with WHERE",
			sql:  "DELETE FROM sessions WHERE expired = true;",
			want: 0,
		},
		{
			name: "allows UPDATE with WHERE",
			sql:  "UPDATE users SET active = 0 WHERE id = 1;",
			want: 0,
		},
		{
			name: "ignores SELECT",
			sql:  "SELECT id FROM users;",
			want: 0,
		},
		{
			name: "flags multi-line DELETE without WHERE",
			sql:  "DELETE\nFROM sessions;",
			want: 1,
		},
		{
			name: "flags only the dangerous statement among multiple",
			sql:  "SELECT id FROM users;\nDELETE FROM sessions;",
			want: 1,
		},
		{
			name: "ignores INSERT",
			sql:  "INSERT INTO users (id, name) VALUES (1, 'test');",
			want: 0,
		},
	}

	rule := rules.MissingWhere{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := splitLines(tt.sql)
			got := rule.Check(tt.sql, lines)
			if len(got) != tt.want {
				t.Errorf("got %d violations, want %d", len(got), tt.want)
			}
		})
	}
}

func TestKeywordCasing(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{
			name: "allows all uppercase keywords",
			sql:  "SELECT id FROM users WHERE active = 1;",
			want: 0,
		},
		{
			name: "allows all lowercase keywords",
			sql:  "select id from users where active = 1;",
			want: 0,
		},
		{
			name: "flags mixed-case Select",
			sql:  "Select id FROM users;",
			want: 1,
		},
		{
			name: "flags multiple mixed-case keywords on one line",
			sql:  "Select id From users Where active = 1;",
			want: 3,
		},
		{
			name: "skips comment lines",
			sql:  "-- Select id From users;",
			want: 0,
		},
		{
			name: "does not match keyword inside identifier",
			sql:  "SELECT SELECTED FROM users;",
			want: 0,
		},
	}

	rule := rules.KeywordCasing{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := splitLines(tt.sql)
			got := rule.Check(tt.sql, lines)
			if len(got) != tt.want {
				t.Errorf("got %d violations, want %d", len(got), tt.want)
			}
		})
	}
}

func TestTrailingSemicolon(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{
			name: "allows statement ending with semicolon",
			sql:  "SELECT id FROM users;",
			want: 0,
		},
		{
			name: "flags missing semicolon",
			sql:  "SELECT id FROM users",
			want: 1,
		},
		{
			name: "allows trailing blank lines after semicolon",
			sql:  "SELECT id FROM users;\n\n",
			want: 0,
		},
		{
			name: "allows empty input",
			sql:  "",
			want: 0,
		},
		{
			name: "flags multi-line statement without trailing semicolon",
			sql:  "SELECT id\nFROM users\nWHERE active = 1",
			want: 1,
		},
	}

	rule := rules.TrailingSemicolon{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := splitLines(tt.sql)
			got := rule.Check(tt.sql, lines)
			if len(got) != tt.want {
				t.Errorf("got %d violations, want %d", len(got), tt.want)
			}
		})
	}
}

func TestLeadingWildcard(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{
			name: "flags LIKE with leading wildcard",
			sql:  "SELECT id FROM users WHERE name LIKE '%foo';",
			want: 1,
		},
		{
			name: "flags ILIKE with leading wildcard",
			sql:  "SELECT id FROM users WHERE name ILIKE '%foo';",
			want: 1,
		},
		{
			name: "allows LIKE with trailing wildcard only",
			sql:  "SELECT id FROM users WHERE name LIKE 'foo%';",
			want: 0,
		},
		{
			name: "allows LIKE with no wildcard",
			sql:  "SELECT id FROM users WHERE name LIKE 'foo';",
			want: 0,
		},
		{
			name: "flags LIKE with both leading and trailing wildcard",
			sql:  "SELECT id FROM users WHERE name LIKE '%foo%';",
			want: 1,
		},
	}

	rule := rules.LeadingWildcard{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := splitLines(tt.sql)
			got := rule.Check(tt.sql, lines)
			if len(got) != tt.want {
				t.Errorf("got %d violations, want %d", len(got), tt.want)
			}
		})
	}
}

func TestImplicitJoin(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{
			name: "flags comma-separated FROM tables",
			sql:  "SELECT * FROM users, orders WHERE users.id = orders.user_id;",
			want: 1,
		},
		{
			name: "allows explicit JOIN",
			sql:  "SELECT * FROM users JOIN orders ON users.id = orders.user_id;",
			want: 0,
		},
		{
			name: "allows single table",
			sql:  "SELECT * FROM users;",
			want: 0,
		},
		{
			name: "does not flag comma in WHERE clause without FROM on same line",
			sql:  "WHERE id IN (1, 2, 3)",
			want: 0,
		},
		{
			name: "flags multiple comma-separated tables",
			sql:  "SELECT * FROM users, orders, products;",
			want: 1,
		},
	}

	rule := rules.ImplicitJoin{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := splitLines(tt.sql)
			got := rule.Check(tt.sql, lines)
			if len(got) != tt.want {
				t.Errorf("got %d violations, want %d", len(got), tt.want)
			}
		})
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
