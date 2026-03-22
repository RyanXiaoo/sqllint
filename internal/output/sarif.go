package output

import (
	"encoding/json"
	"io"
	"path/filepath"

	"github.com/ryanxiao/go-sqllint/internal/linter"
	"github.com/ryanxiao/go-sqllint/internal/rules"
)

type sarifRoot struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
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
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string          `json:"id"`
	ShortDescription sarifMessage    `json:"shortDescription"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

// SARIF writes lint results in SARIF 2.1.0 format for GitHub Code Scanning.
func SARIF(w io.Writer, results []linter.Result) error {
	// Collect deduplicated rules, preserving first-seen message as shortDescription.
	ruleDescriptions := map[string]string{}
	var ruleOrder []string

	for _, r := range results {
		for _, v := range r.Violations {
			if _, seen := ruleDescriptions[v.RuleID]; !seen {
				ruleDescriptions[v.RuleID] = v.Message
				ruleOrder = append(ruleOrder, v.RuleID)
			}
		}
	}

	sarifRules := make([]sarifRule, 0, len(ruleOrder))
	for _, id := range ruleOrder {
		sarifRules = append(sarifRules, sarifRule{
			ID:               id,
			ShortDescription: sarifMessage{Text: ruleDescriptions[id]},
		})
	}

	var sarifResults []sarifResult
	for _, r := range results {
		uri := filepath.ToSlash(r.File)
		for _, v := range r.Violations {
			level := "warning"
			if v.Severity == rules.SeverityError {
				level = "error"
			}
			line := v.Line
			if line < 1 {
				line = 1
			}
			sarifResults = append(sarifResults, sarifResult{
				RuleID:  v.RuleID,
				Level:   level,
				Message: sarifMessage{Text: v.Message},
				Locations: []sarifLocation{{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{
							URI:       uri,
							URIBaseID: "%SRCROOT%",
						},
						Region: sarifRegion{StartLine: line},
					},
				}},
			})
		}
	}

	root := sarifRoot{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:           "go-sqllint",
					Version:        "0.1.0",
					InformationURI: "https://github.com/ryanxiao/go-sqllint",
					Rules:          sarifRules,
				},
			},
			Results: sarifResults,
		}},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(root)
}
