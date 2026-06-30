package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shadiff/internal/config"
	"shadiff/internal/model"

	"github.com/spf13/cobra"
)

func TestRunDiff_UsesConfigIgnoreOrder(t *testing.T) {
	withRuntimeConfig(t, func(cfg *config.AppConfig) {
		cfg.Diff.IgnoreOrder = true
	})
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "diff-case", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1,2]}`)},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "rep-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[2,1]}`)},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	diffSession = session.Name
	diffRulesFile = ""
	diffIgnoreOrder = false
	diffIgnoreHeaders = nil
	diffOutputFile = ""
	diffFailOn = "none"

	cmd := &cobra.Command{}
	cmd.Flags().Bool("ignore-order", false, "")
	cmd.Flags().StringArray("ignore-headers", nil, "")
	cmd.Flags().String("rules", "", "")
	cmd.Flags().String("output", "terminal", "")
	cmd.Flags().String("output-file", "", "")
	cmd.Flags().String("fail-on", "none", "")

	output := captureStdout(t, func() {
		if err := runDiff(cmd, nil); err != nil {
			t.Fatalf("runDiff() error: %v", err)
		}
	})
	if !strings.Contains(output, "[MATCH]") {
		t.Fatalf("output = %q, want MATCH", output)
	}
}

func TestRunDiff_OutputsJSON(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "diff-json-case", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1,2]}`)},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "rep-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1,2]}`)},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	diffSession = session.Name
	diffRulesFile = ""
	diffIgnoreOrder = false
	diffIgnoreHeaders = nil
	diffOutput = "terminal"
	diffOutputFile = ""
	diffFailOn = "none"

	cmd := &cobra.Command{}
	cmd.Flags().Bool("ignore-order", false, "")
	cmd.Flags().StringArray("ignore-headers", nil, "")
	cmd.Flags().String("rules", "", "")
	cmd.Flags().String("output", "json", "")
	cmd.Flags().String("output-file", "", "")
	cmd.Flags().String("fail-on", "none", "")

	output := captureStdout(t, func() {
		if err := runDiff(cmd, nil); err != nil {
			t.Fatalf("runDiff() error: %v", err)
		}
	})

	var report struct {
		Summary model.DiffSummary  `json:"summary"`
		Results []model.DiffResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("expected JSON output, got error: %v\noutput=%s", err, output)
	}
	if report.Summary.SessionID != session.ID {
		t.Fatalf("summary sessionID = %q, want %q", report.Summary.SessionID, session.ID)
	}
	if len(report.Results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(report.Results))
	}
}

func TestRunDiff_InvalidOutputReturnsError(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "diff-invalid-output", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1]}`)},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "rep-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1]}`)},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	diffSession = session.Name
	diffRulesFile = ""
	diffIgnoreOrder = false
	diffIgnoreHeaders = nil
	diffOutput = "terminal"
	diffOutputFile = ""
	diffFailOn = "none"

	cmd := &cobra.Command{}
	cmd.Flags().Bool("ignore-order", false, "")
	cmd.Flags().StringArray("ignore-headers", nil, "")
	cmd.Flags().String("rules", "", "")
	cmd.Flags().String("output", "xml", "")
	cmd.Flags().String("output-file", "", "")
	cmd.Flags().String("fail-on", "none", "")

	if err := runDiff(cmd, nil); err == nil {
		t.Fatal("expected invalid output format to return an error")
	} else if !strings.Contains(err.Error(), "unsupported diff output format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDiff_InvalidFailOnReturnsError(t *testing.T) {
	withRuntimeConfig(t, nil)

	diffSession = ""
	diffRulesFile = ""
	diffIgnoreOrder = false
	diffIgnoreHeaders = nil
	diffOutput = "terminal"
	diffOutputFile = ""
	diffFailOn = "none"

	cmd := &cobra.Command{}
	cmd.Flags().Bool("ignore-order", false, "")
	cmd.Flags().StringArray("ignore-headers", nil, "")
	cmd.Flags().String("rules", "", "")
	cmd.Flags().String("output", "terminal", "")
	cmd.Flags().String("output-file", "", "")
	cmd.Flags().String("fail-on", "panic", "")

	if err := runDiff(cmd, nil); err == nil {
		t.Fatal("expected invalid fail-on policy to return an error")
	} else if !strings.Contains(err.Error(), "unsupported diff fail-on policy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDiff_WritesOutputFile(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "diff-output-file", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1,2]}`)},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "rep-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1,2]}`)},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "diff.json")
	diffSession = session.Name
	diffRulesFile = ""
	diffIgnoreOrder = false
	diffIgnoreHeaders = nil
	diffOutput = "terminal"
	diffOutputFile = ""
	diffFailOn = "none"

	cmd := &cobra.Command{}
	cmd.Flags().Bool("ignore-order", false, "")
	cmd.Flags().StringArray("ignore-headers", nil, "")
	cmd.Flags().String("rules", "", "")
	cmd.Flags().String("output", "json", "")
	cmd.Flags().String("output-file", outputPath, "")
	cmd.Flags().String("fail-on", "none", "")

	stdout := captureStdout(t, func() {
		if err := runDiff(cmd, nil); err != nil {
			t.Fatalf("runDiff() error: %v", err)
		}
	})
	if !strings.Contains(stdout, "Diff output written") {
		t.Fatalf("stdout = %q, want output file confirmation", stdout)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	var report struct {
		Summary model.DiffSummary  `json:"summary"`
		Results []model.DiffResult `json:"results"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("expected JSON output file, got error: %v\noutput=%s", err, string(data))
	}
	if report.Summary.SessionID != session.ID {
		t.Fatalf("summary sessionID = %q, want %q", report.Summary.SessionID, session.ID)
	}
}

func TestRunDiff_FailOnDiffReturnsError(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "diff-fail-on-diff", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1]}`)},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "rep-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 201, Body: []byte(`{"items":[1]}`)},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	diffSession = session.Name
	diffRulesFile = ""
	diffIgnoreOrder = false
	diffIgnoreHeaders = nil
	diffOutput = "terminal"
	diffOutputFile = ""
	diffFailOn = "none"

	cmd := &cobra.Command{}
	cmd.Flags().Bool("ignore-order", false, "")
	cmd.Flags().StringArray("ignore-headers", nil, "")
	cmd.Flags().String("rules", "", "")
	cmd.Flags().String("output", "terminal", "")
	cmd.Flags().String("output-file", "", "")
	cmd.Flags().String("fail-on", "diff", "")

	output := captureStdout(t, func() {
		err := runDiff(cmd, nil)
		if err == nil || !strings.Contains(err.Error(), "records differ") {
			t.Fatalf("runDiff() error = %v, want fail-on diff error", err)
		}
	})
	if !strings.Contains(output, "[DIFF]") {
		t.Fatalf("output = %q, want diff report before failure", output)
	}
}

func TestRunDiff_FailOnErrorOnlyFailsForErrorSeverity(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "diff-fail-on-error", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/headers"},
		Response: model.HTTPResponse{
			StatusCode: 200,
			Headers:    map[string][]string{"X-Custom": {"old"}},
			Body:       []byte(`{"ok":true}`),
		},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "rep-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/headers"},
		Response: model.HTTPResponse{
			StatusCode: 200,
			Headers:    map[string][]string{"X-Custom": {"new"}},
			Body:       []byte(`{"ok":true}`),
		},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	diffSession = session.Name
	diffRulesFile = ""
	diffIgnoreOrder = false
	diffIgnoreHeaders = nil
	diffOutput = "terminal"
	diffOutputFile = ""
	diffFailOn = "none"

	cmd := &cobra.Command{}
	cmd.Flags().Bool("ignore-order", false, "")
	cmd.Flags().StringArray("ignore-headers", nil, "")
	cmd.Flags().String("rules", "", "")
	cmd.Flags().String("output", "terminal", "")
	cmd.Flags().String("output-file", "", "")
	cmd.Flags().String("fail-on", "error", "")

	if err := runDiff(cmd, nil); err != nil {
		t.Fatalf("runDiff() warning-only error = %v, want nil", err)
	}

	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "rep-1b",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/headers"},
		Response:  model.HTTPResponse{StatusCode: 500, Body: []byte(`{"ok":true}`)},
	}); err != nil {
		t.Fatalf("AppendReplayRecord(error case) error: %v", err)
	}

	if err := runDiff(cmd, nil); err == nil || !strings.Contains(err.Error(), "error differences") {
		t.Fatalf("runDiff() error = %v, want fail-on error policy", err)
	}
}
