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
		name  string // description of the test case
		sql   string
		want  int // expected number of violations
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
