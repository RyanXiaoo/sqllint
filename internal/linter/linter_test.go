package linter_test

import (
	"testing"

	"github.com/ryanxiao/go-sqllint/internal/linter"
	"github.com/ryanxiao/go-sqllint/internal/rules"
	"github.com/ryanxiao/go-sqllint/internal/config"
)

func TestHasErrors(t *testing.T) {
	tests := []struct {
		name       string
		violations []rules.Violation
		want       bool
	}{
		{
			name:       "no violations",
			violations: nil,
			want:       false,
		},
		{
			name: "warning only",
			violations: []rules.Violation{
				{RuleID: "select-star", Severity: rules.SeverityWarning},
			},
			want: false,
		},
		{
			name: "error present",
			violations: []rules.Violation{
				{RuleID: "missing-where", Severity: rules.SeverityError},
			},
			want: true,
		},
		{
			name: "mixed warning and error",
			violations: []rules.Violation{
				{RuleID: "select-star", Severity: rules.SeverityWarning},
				{RuleID: "missing-where", Severity: rules.SeverityError},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := linter.Result{File: "test.sql", Violations: tt.violations}
			if got := r.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasWarnings(t *testing.T) {
	tests := []struct {
		name       string
		violations []rules.Violation
		want       bool
	}{
		{
			name:       "no violations",
			violations: nil,
			want:       false,
		},
		{
			name: "error only",
			violations: []rules.Violation{
				{RuleID: "missing-where", Severity: rules.SeverityError},
			},
			want: false,
		},
		{
			name: "warning present",
			violations: []rules.Violation{
				{RuleID: "select-star", Severity: rules.SeverityWarning},
			},
			want: true,
		},
		{
			name: "mixed warning and error",
			violations: []rules.Violation{
				{RuleID: "select-star", Severity: rules.SeverityWarning},
				{RuleID: "missing-where", Severity: rules.SeverityError},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := linter.Result{File: "test.sql", Violations: tt.violations}
			if got := r.HasWarnings(); got != tt.want {
				t.Errorf("HasWarnings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLintExitConditions(t *testing.T) {
	cfg := config.Config{}
	l := linter.New(cfg)

	t.Run("SELECT * produces warning, no error", func(t *testing.T) {
		r := l.Lint("test.sql", "SELECT * FROM users;")
		if !r.HasWarnings() {
			t.Error("expected warnings for SELECT *")
		}
		if r.HasErrors() {
			t.Error("expected no errors for SELECT *")
		}
	})

	t.Run("DELETE without WHERE produces error", func(t *testing.T) {
		r := l.Lint("test.sql", "DELETE FROM users")
		if !r.HasErrors() {
			t.Error("expected error for DELETE without WHERE")
		}
	})

	t.Run("clean SQL produces no violations", func(t *testing.T) {
		r := l.Lint("test.sql", "SELECT id FROM users WHERE active = 1;")
		if r.HasErrors() || r.HasWarnings() {
			t.Errorf("expected clean result, got %d violations", len(r.Violations))
		}
	})
}
