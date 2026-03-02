package reporter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"shadiff/internal/model"
)

// sampleData returns test fixtures for reporter tests.
func sampleData() ([]model.DiffResult, model.DiffSummary) {
	results := []model.DiffResult{
		{
			RecordID: "rec-001",
			Sequence: 1,
			Request: model.HTTPRequest{
				Method: "GET",
				Path:   "/api/users",
			},
			Match: true,
		},
		{
			RecordID: "rec-002",
			Sequence: 2,
			Request: model.HTTPRequest{
				Method: "POST",
				Path:   "/api/orders",
			},
			Match: false,
			Differences: []model.Difference{
				{
					Kind:     model.DiffStatusCode,
					Path:     "status_code",
					Expected: 200,
					Actual:   500,
					Message:  "status code mismatch",
					Severity: model.SeverityError,
				},
				{
					Kind:     model.DiffBodyField,
					Path:     "body.data.id",
					Expected: "abc",
					Actual:   "xyz",
					Message:  "field value changed",
					Severity: model.SeverityWarning,
					Ignored:  true,
					Rule:     "ignore-id",
				},
			},
		},
	}

	summary := model.DiffSummary{
		SessionID:   "session-test",
		TotalCount:  2,
		MatchCount:  1,
		DiffCount:   1,
		ErrorCount:  1,
		IgnoreCount: 1,
		MatchRate:   0.5,
	}

	return results, summary
}

// --- NewReporter factory tests ---

func TestNewReporter_Terminal(t *testing.T) {
	r, err := NewReporter("terminal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := r.(*TerminalReporter); !ok {
		t.Errorf("expected *TerminalReporter, got %T", r)
	}
}

func TestNewReporter_EmptyDefaultsToTerminal(t *testing.T) {
	r, err := NewReporter("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := r.(*TerminalReporter); !ok {
		t.Errorf("expected *TerminalReporter for empty format, got %T", r)
	}
}

func TestNewReporter_JSON(t *testing.T) {
	r, err := NewReporter("json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := r.(*JSONReporter); !ok {
		t.Errorf("expected *JSONReporter, got %T", r)
	}
}

func TestNewReporter_HTML(t *testing.T) {
	r, err := NewReporter("html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := r.(*HTMLReporter); !ok {
		t.Errorf("expected *HTMLReporter, got %T", r)
	}
}

func TestNewReporter_UnknownFormat(t *testing.T) {
	_, err := NewReporter("xml")
	if err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention 'unsupported', got: %v", err)
	}
}

// --- JSONReporter.Generate tests ---

func TestJSONReporter_Generate_ValidJSON(t *testing.T) {
	results, summary := sampleData()
	r := &JSONReporter{}
	var buf bytes.Buffer

	err := r.Generate(results, summary, &buf)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Must be valid JSON
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	// Must contain "summary" and "results" keys
	if _, ok := parsed["summary"]; !ok {
		t.Error("JSON output missing 'summary' key")
	}
	if _, ok := parsed["results"]; !ok {
		t.Error("JSON output missing 'results' key")
	}
}

func TestJSONReporter_Generate_Content(t *testing.T) {
	results, summary := sampleData()
	r := &JSONReporter{}
	var buf bytes.Buffer

	if err := r.Generate(results, summary, &buf); err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Decode the full report structure
	var report struct {
		Summary model.DiffSummary  `json:"summary"`
		Results []model.DiffResult `json:"results"`
	}
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if report.Summary.SessionID != "session-test" {
		t.Errorf("sessionID: got %q, want %q", report.Summary.SessionID, "session-test")
	}
	if report.Summary.TotalCount != 2 {
		t.Errorf("totalCount: got %d, want %d", report.Summary.TotalCount, 2)
	}
	if len(report.Results) != 2 {
		t.Errorf("results count: got %d, want %d", len(report.Results), 2)
	}
	if report.Results[0].Match != true {
		t.Error("first result should be a match")
	}
	if report.Results[1].Match != false {
		t.Error("second result should not be a match")
	}
}

func TestJSONReporter_Generate_EmptyResults(t *testing.T) {
	r := &JSONReporter{}
	var buf bytes.Buffer

	err := r.Generate(nil, model.DiffSummary{}, &buf)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

// --- HTMLReporter.Generate tests ---

func TestHTMLReporter_Generate_ValidHTML(t *testing.T) {
	results, summary := sampleData()
	r := &HTMLReporter{}
	var buf bytes.Buffer

	err := r.Generate(results, summary, &buf)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	output := buf.String()

	// Check for essential HTML structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("output missing DOCTYPE")
	}
	if !strings.Contains(output, "<html") {
		t.Error("output missing <html> tag")
	}
	if !strings.Contains(output, "Shadiff Report") {
		t.Error("output missing report title")
	}
}

func TestHTMLReporter_Generate_Content(t *testing.T) {
	results, summary := sampleData()
	r := &HTMLReporter{}
	var buf bytes.Buffer

	if err := r.Generate(results, summary, &buf); err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	output := buf.String()

	// Check that result data appears in output
	if !strings.Contains(output, "GET") {
		t.Error("output missing GET method")
	}
	if !strings.Contains(output, "/api/users") {
		t.Error("output missing /api/users path")
	}
	if !strings.Contains(output, "/api/orders") {
		t.Error("output missing /api/orders path")
	}
	if !strings.Contains(output, "MATCH") {
		t.Error("output missing MATCH label")
	}
	if !strings.Contains(output, "DIFF") {
		t.Error("output missing DIFF label")
	}
	// Match rate 50.0%
	if !strings.Contains(output, "50.0") {
		t.Error("output missing match rate percentage")
	}
}

// --- TerminalReporter.Generate tests ---

func TestTerminalReporter_Generate_Content(t *testing.T) {
	results, summary := sampleData()
	r := &TerminalReporter{}
	var buf bytes.Buffer

	err := r.Generate(results, summary, &buf)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	output := buf.String()

	// Report header
	if !strings.Contains(output, "Shadiff Report") {
		t.Error("output missing report title")
	}

	// Methods and paths
	if !strings.Contains(output, "GET") {
		t.Error("output missing GET method")
	}
	if !strings.Contains(output, "/api/users") {
		t.Error("output missing /api/users path")
	}
	if !strings.Contains(output, "POST") {
		t.Error("output missing POST method")
	}
	if !strings.Contains(output, "/api/orders") {
		t.Error("output missing /api/orders path")
	}

	// Match/diff labels
	if !strings.Contains(output, "[MATCH]") {
		t.Error("output missing [MATCH] label")
	}
	if !strings.Contains(output, "[DIFF]") {
		t.Error("output missing [DIFF] label")
	}

	// Summary stats
	if !strings.Contains(output, "2 records") {
		t.Error("output missing total records count")
	}
	if !strings.Contains(output, "1 matched") {
		t.Error("output missing matched count")
	}
	if !strings.Contains(output, "1 differences") {
		t.Error("output missing diff count")
	}

	// Match rate
	if !strings.Contains(output, "50.0%") {
		t.Error("output missing match rate")
	}
}

func TestTerminalReporter_Generate_IgnoredDifference(t *testing.T) {
	results, summary := sampleData()
	r := &TerminalReporter{}
	var buf bytes.Buffer

	if err := r.Generate(results, summary, &buf); err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	output := buf.String()

	// Ignored difference should mention the rule
	if !strings.Contains(output, "ignored") {
		t.Error("output missing 'ignored' label for ignored difference")
	}
	if !strings.Contains(output, "ignore-id") {
		t.Error("output missing rule name for ignored difference")
	}
}

func TestTerminalReporter_Generate_ErrorCount(t *testing.T) {
	results, summary := sampleData()
	r := &TerminalReporter{}
	var buf bytes.Buffer

	if err := r.Generate(results, summary, &buf); err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "1 error-level") {
		t.Error("output missing error count line")
	}
	if !strings.Contains(output, "1 differences (rule matched)") {
		t.Error("output missing ignore count line")
	}
}
