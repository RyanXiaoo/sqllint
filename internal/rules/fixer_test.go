package rules

import (
	"strings"
	"testing"
)

func TestKeywordCasingFix(t *testing.T) {
	rule := KeywordCasing{}
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "mixed case select",
			input: "Select id From users Where active = 1",
			want:  "SELECT id FROM users WHERE active = 1",
		},
		{
			name:  "already uppercase unchanged",
			input: "SELECT id FROM users WHERE active = 1",
			want:  "SELECT id FROM users WHERE active = 1",
		},
		{
			name:  "already lowercase unchanged",
			input: "select id from users where active = 1",
			want:  "select id from users where active = 1",
		},
		{
			name:  "mixed case join",
			input: "SELECT id FROM users Inner Join orders On users.id = orders.user_id",
			want:  "SELECT id FROM users INNER JOIN orders ON users.id = orders.user_id",
		},
		{
			name:  "skip comment lines",
			input: "-- Select From Where\nSELECT id FROM users",
			want:  "-- Select From Where\nSELECT id FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.input, "\n")
			got := rule.Fix(tt.input, lines)
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTrailingSemicolonFix(t *testing.T) {
	rule := TrailingSemicolon{}
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "add missing semicolon",
			input: "SELECT id FROM users",
			want:  "SELECT id FROM users;",
		},
		{
			name:  "already has semicolon",
			input: "SELECT id FROM users;",
			want:  "SELECT id FROM users;",
		},
		{
			name:  "trailing blank lines",
			input: "SELECT id FROM users\n\n",
			want:  "SELECT id FROM users;\n\n",
		},
		{
			name:  "multiline",
			input: "SELECT id\nFROM users",
			want:  "SELECT id\nFROM users;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.input, "\n")
			got := rule.Fix(tt.input, lines)
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}
