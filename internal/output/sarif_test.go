package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ryanxiao/go-sqllint/internal/linter"
	"github.com/ryanxiao/go-sqllint/internal/output"
	"github.com/ryanxiao/go-sqllint/internal/rules"
)

// sarifDoc mirrors the top-level SARIF structure for unmarshalling in tests.
type sarifDoc struct {
	Version string    `json:"version"`
	Schema  string    `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name  string      `json:"name"`
	Rules []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string `json:"id"`
	ShortDescription struct {
		Text string `json:"text"`
	} `json:"shortDescription"`
}

type sarifResult struct {
	RuleID    string `json:"ruleId"`
	Level     string `json:"level"`
	Message   struct{ Text string `json:"text"` } `json:"message"`
	Locations []struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				URI       string `json:"uri"`
				URIBaseID string `json:"uriBaseId"`
			} `json:"artifactLocation"`
			Region struct {
				StartLine int `json:"startLine"`
			} `json:"region"`
		} `json:"physicalLocation"`
	} `json:"locations"`
}

func parseSARIF(t *testing.T, w *bytes.Buffer) sarifDoc {
	t.Helper()
	var doc sarifDoc
	if err := json.Unmarshal(w.Bytes(), &doc); err != nil {
		t.Fatalf("SARIF output is not valid JSON: %v\n%s", err, w.String())
	}
	return doc
}

func TestSARIFVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := output.SARIF(&buf, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := parseSARIF(t, &buf)
	if doc.Version != "2.1.0" {
		t.Errorf("version = %q, want 2.1.0", doc.Version)
	}
	if !strings.Contains(doc.Schema, "sarif-2.1.0") {
		t.Errorf("schema = %q, want sarif-2.1.0 schema", doc.Schema)
	}
}

func TestSARIFEmptyResults(t *testing.T) {
	var buf bytes.Buffer
	if err := output.SARIF(&buf, []linter.Result{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := parseSARIF(t, &buf)
	if len(doc.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(doc.Runs))
	}
	if len(doc.Runs[0].Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(doc.Runs[0].Results))
	}
}

func TestSARIFSeverityMapping(t *testing.T) {
	results := []linter.Result{{
		File: "test.sql",
		Violations: []rules.Violation{
			{RuleID: "select-star", Message: "Avoid SELECT *", Line: 1, Severity: rules.SeverityWarning},
			{RuleID: "missing-where", Message: "DELETE without WHERE", Line: 2, Severity: rules.SeverityError},
		},
	}}

	var buf bytes.Buffer
	if err := output.SARIF(&buf, results); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := parseSARIF(t, &buf)
	run := doc.Runs[0]

	if len(run.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(run.Results))
	}

	for _, r := range run.Results {
		switch r.RuleID {
		case "select-star":
			if r.Level != "warning" {
				t.Errorf("select-star level = %q, want warning", r.Level)
			}
		case "missing-where":
			if r.Level != "error" {
				t.Errorf("missing-where level = %q, want error", r.Level)
			}
		default:
			t.Errorf("unexpected ruleId %q", r.RuleID)
		}
	}
}

func TestSARIFRuleDeduplication(t *testing.T) {
	// Two violations of the same rule across two files — driver.rules should deduplicate.
	results := []linter.Result{
		{
			File: "a.sql",
			Violations: []rules.Violation{
				{RuleID: "select-star", Message: "Avoid SELECT *", Line: 1, Severity: rules.SeverityWarning},
			},
		},
		{
			File: "b.sql",
			Violations: []rules.Violation{
				{RuleID: "select-star", Message: "Avoid SELECT *", Line: 3, Severity: rules.SeverityWarning},
			},
		},
	}

	var buf bytes.Buffer
	if err := output.SARIF(&buf, results); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := parseSARIF(t, &buf)
	driverRules := doc.Runs[0].Tool.Driver.Rules

	if len(driverRules) != 1 {
		t.Errorf("driver.rules length = %d, want 1 (deduplicated)", len(driverRules))
	}
}

func TestSARIFLocation(t *testing.T) {
	results := []linter.Result{{
		File: "subdir/test.sql",
		Violations: []rules.Violation{
			{RuleID: "select-star", Message: "Avoid SELECT *", Line: 5, Severity: rules.SeverityWarning},
		},
	}}

	var buf bytes.Buffer
	if err := output.SARIF(&buf, results); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := parseSARIF(t, &buf)
	r := doc.Runs[0].Results[0]

	loc := r.Locations[0].PhysicalLocation
	if loc.ArtifactLocation.URI != "subdir/test.sql" {
		t.Errorf("uri = %q, want subdir/test.sql", loc.ArtifactLocation.URI)
	}
	if loc.ArtifactLocation.URIBaseID != "%SRCROOT%" {
		t.Errorf("uriBaseId = %q, want %%SRCROOT%%", loc.ArtifactLocation.URIBaseID)
	}
	if loc.Region.StartLine != 5 {
		t.Errorf("startLine = %d, want 5", loc.Region.StartLine)
	}
}

func TestSARIFWindowsPathConversion(t *testing.T) {
	results := []linter.Result{{
		File: `subdir\test.sql`,
		Violations: []rules.Violation{
			{RuleID: "select-star", Message: "msg", Line: 1, Severity: rules.SeverityWarning},
		},
	}}

	var buf bytes.Buffer
	if err := output.SARIF(&buf, results); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := parseSARIF(t, &buf)
	uri := doc.Runs[0].Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI
	if strings.Contains(uri, `\`) {
		t.Errorf("URI contains backslash: %q", uri)
	}
}

func TestSARIFLineFloorAt1(t *testing.T) {
	// A violation with line 0 should be clamped to 1 in SARIF output.
	results := []linter.Result{{
		File: "test.sql",
		Violations: []rules.Violation{
			{RuleID: "select-star", Message: "msg", Line: 0, Severity: rules.SeverityWarning},
		},
	}}

	var buf bytes.Buffer
	if err := output.SARIF(&buf, results); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := parseSARIF(t, &buf)
	line := doc.Runs[0].Results[0].Locations[0].PhysicalLocation.Region.StartLine
	if line < 1 {
		t.Errorf("startLine = %d, want >= 1", line)
	}
}
