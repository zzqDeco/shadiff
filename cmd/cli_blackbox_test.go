package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"shadiff/internal/config"
	"shadiff/internal/model"
	"shadiff/internal/storage"
)

var cliBinary string

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	repoRoot := filepath.Dir(wd)
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	buildDir, err := os.MkdirTemp("", "shadiff-cli-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(buildDir)

	cliBinary = filepath.Join(buildDir, "shadiff"+ext)
	build := exec.Command("go", "build", "-o", cliBinary, ".")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		panic(string(output))
	}

	os.Exit(m.Run())
}

func runCLI(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	cmd := exec.Command(cliBinary, args...)
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func writeConfigFile(t *testing.T, dataDir string, mutate func(*config.AppConfig)) string {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Storage.DataDir = dataDir
	cfg.Log.LogDir = filepath.Join(dataDir, "logs")
	cfg.Log.Level = "error"
	if mutate != nil {
		mutate(cfg)
	}

	path := filepath.Join(t.TempDir(), "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	return path
}

func createSessionStore(t *testing.T, dataDir string) *storage.FileStore {
	t.Helper()
	store, err := storage.NewFileStore(dataDir)
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}
	return store
}

func TestCLI_VersionCreatesConfigFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	stdout, stderr, err := runCLI(t, "--config", configPath, "version")
	if err != nil {
		t.Fatalf("runCLI() error: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "shadiff") {
		t.Fatalf("stdout = %q, want version output", stdout)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to be created: %v", err)
	}
}

func TestCLI_SessionListUsesConfiguredDataDir(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	configPath := writeConfigFile(t, dataDir, nil)

	stdout, stderr, err := runCLI(t, "--config", configPath, "session", "list")
	if err != nil {
		t.Fatalf("runCLI() error: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "No sessions found") {
		t.Fatalf("stdout = %q, want empty session message", stdout)
	}
}

func TestCLI_ReportGeneratesJSON(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	configPath := writeConfigFile(t, dataDir, nil)
	store := createSessionStore(t, dataDir)

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

	outputPath := filepath.Join(t.TempDir(), "report.json")
	stdout, stderr, err := runCLI(t, "--config", configPath, "report", "-s", session.Name, "-f", "json", "-o", outputPath)
	if err != nil {
		t.Fatalf("runCLI() error: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Report generated") {
		t.Fatalf("stdout = %q, want generated message", stdout)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if !strings.Contains(string(data), "\"sessionID\"") {
		t.Fatalf("report output = %q, want sessionID JSON", string(data))
	}
}

func TestCLI_DiffUsesConfigIgnoreOrder(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	configPath := writeConfigFile(t, dataDir, func(cfg *config.AppConfig) {
		cfg.Diff.IgnoreOrder = true
	})
	store := createSessionStore(t, dataDir)

	session := &model.Session{Name: "diff-case", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	original := &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1,2,3]}`)},
	}
	replayed := &model.Record{
		ID:        "replay-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   original.Request,
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[3,2,1]}`)},
	}
	if err := store.AppendRecord(session.ID, original); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, replayed); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	stdout, stderr, err := runCLI(t, "--config", configPath, "diff", "-s", session.Name)
	if err != nil {
		t.Fatalf("runCLI() error: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "[MATCH]") {
		t.Fatalf("stdout = %q, want MATCH result", stdout)
	}
}

func TestCLI_DiffOutputsJSON(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	configPath := writeConfigFile(t, dataDir, nil)
	store := createSessionStore(t, dataDir)

	session := &model.Session{Name: "diff-json-cli", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	original := &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1,2]}`)},
	}
	replayed := &model.Record{
		ID:        "replay-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   original.Request,
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1,2]}`)},
	}
	if err := store.AppendRecord(session.ID, original); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, replayed); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	stdout, stderr, err := runCLI(t, "--config", configPath, "diff", "-s", session.Name, "-o", "json")
	if err != nil {
		t.Fatalf("runCLI() error: %v\nstderr=%s", err, stderr)
	}

	var report struct {
		Summary model.DiffSummary  `json:"summary"`
		Results []model.DiffResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("expected JSON stdout, got error: %v\nstdout=%s", err, stdout)
	}
	if report.Summary.SessionID != session.ID {
		t.Fatalf("summary sessionID = %q, want %q", report.Summary.SessionID, session.ID)
	}
	if len(report.Results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(report.Results))
	}
}

func TestCLI_DiffWritesOutputFile(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	configPath := writeConfigFile(t, dataDir, nil)
	store := createSessionStore(t, dataDir)

	session := &model.Session{Name: "diff-output-file-cli", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	original := &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1,2]}`)},
	}
	replayed := &model.Record{
		ID:        "replay-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   original.Request,
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1,2]}`)},
	}
	if err := store.AppendRecord(session.ID, original); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, replayed); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "diff.json")
	stdout, stderr, err := runCLI(t, "--config", configPath, "diff", "-s", session.Name, "-o", "json", "--output-file", outputPath)
	if err != nil {
		t.Fatalf("runCLI() error: %v\nstderr=%s", err, stderr)
	}
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

func TestCLI_DiffFailOnDiffExitsNonZero(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	configPath := writeConfigFile(t, dataDir, nil)
	store := createSessionStore(t, dataDir)

	session := &model.Session{Name: "diff-fail-on-cli", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	original := &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"items":[1]}`)},
	}
	replayed := &model.Record{
		ID:        "replay-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   original.Request,
		Response:  model.HTTPResponse{StatusCode: 201, Body: []byte(`{"items":[1]}`)},
	}
	if err := store.AppendRecord(session.ID, original); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, replayed); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	stdout, stderr, err := runCLI(t, "--config", configPath, "diff", "-s", session.Name, "--fail-on", "none")
	if err != nil {
		t.Fatalf("runCLI(fail-on none) error: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "[DIFF]") {
		t.Fatalf("stdout = %q, want DIFF output", stdout)
	}

	_, stderr, err = runCLI(t, "--config", configPath, "diff", "-s", session.Name, "--fail-on", "diff")
	if err == nil {
		t.Fatal("expected fail-on diff to return a non-zero exit")
	}
	if !strings.Contains(stderr, "records differ") {
		t.Fatalf("stderr = %q, want fail-on diff message", stderr)
	}
}
