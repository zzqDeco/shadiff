package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shadiff/internal/model"

	"github.com/spf13/cobra"
)

func TestRunReport_WritesJSONFile(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "report-case", Status: model.SessionCompleted}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.SaveResults(session.ID, []model.DiffResult{{
		RecordID: "rec-1",
		Sequence: 1,
		Match:    true,
		Request:  model.HTTPRequest{Method: "GET", Path: "/health"},
	}}); err != nil {
		t.Fatalf("SaveResults() error: %v", err)
	}

	reportSession = session.Name
	reportFormat = "json"
	reportOutput = filepath.Join(t.TempDir(), "report.json")

	if err := runReport(&cobra.Command{}, nil); err != nil {
		t.Fatalf("runReport() error: %v", err)
	}

	data, err := os.ReadFile(reportOutput)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if !strings.Contains(string(data), "\"sessionID\"") {
		t.Fatalf("report output = %q, want sessionID", string(data))
	}
}
