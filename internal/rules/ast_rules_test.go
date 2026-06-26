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
	tree, err := parser.Parse(sql)
	if err != nil {
		t.Fatalf("parse error for %q: %v", sql, err)
	}
	return len(rule.CheckAST(tree.Items, sql, splitLines(sql)))
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
