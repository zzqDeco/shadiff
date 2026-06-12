package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"shadiff/internal/capture/dbhook"
	"shadiff/internal/config"
	"shadiff/internal/daemon"
	"shadiff/internal/model"
	"shadiff/internal/storage"

	"github.com/spf13/cobra"
)

func withRuntimeConfig(t *testing.T, mutate func(*config.AppConfig)) string {
	t.Helper()

	dataDir := filepath.Join(t.TempDir(), "data")
	cfg := config.DefaultConfig()
	cfg.Storage.DataDir = dataDir
	cfg.Log.LogDir = filepath.Join(dataDir, "logs")
	cfg.Log.Level = "error"
	if mutate != nil {
		mutate(cfg)
	}

	oldRuntime := runtimeCtx
	oldQuiet := quiet
	oldVerbose := verbose
	t.Cleanup(func() {
		runtimeCtx = oldRuntime
		quiet = oldQuiet
		verbose = oldVerbose
	})

	runtimeCtx = &appRuntime{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Config:     cfg,
		DataDir:    dataDir,
		LogDir:     cfg.Log.LogDir,
	}
	quiet = true
	verbose = false
	return dataDir
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	_ = w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	_ = r.Close()
	return string(data)
}

func newStoreForRuntime(t *testing.T) *storage.FileStore {
	t.Helper()
	store, err := storage.NewFileStore(currentDataDir())
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}
	return store
}

func TestResolveSession_UsesLatestByName(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)

	first := &model.Session{Name: "demo", Status: model.SessionCompleted}
	second := &model.Session{Name: "demo", Status: model.SessionCompleted}
	if err := store.Create(first); err != nil {
		t.Fatalf("Create(first) error: %v", err)
	}
	if err := store.Create(second); err != nil {
		t.Fatalf("Create(second) error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := store.Update(second); err != nil {
		t.Fatalf("Update(second) error: %v", err)
	}

	id, err := resolveSession(store, "demo")
	if err != nil {
		t.Fatalf("resolveSession() error: %v", err)
	}
	if id != second.ID {
		t.Fatalf("resolved id = %q, want %q", id, second.ID)
	}
}

func TestFindSession_AmbiguousNameReturnsError(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)

	for i := 0; i < 2; i++ {
		if err := store.Create(&model.Session{Name: "same", Status: model.SessionCompleted}); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	if _, err := findSession(store, "same"); err == nil {
		t.Fatal("expected ambiguous session name to return an error")
	}
}

func TestRunReplay_ReplaysAndUpdatesSession(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "replay-case", Status: model.SessionRecording}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/hello"},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	replaySession = session.Name
	replayTarget = server.URL
	replayConcurrency = 0
	replayDelay = ""
	replayDBProxy = nil

	cmd := &cobra.Command{}
	cmd.Flags().IntVar(&replayConcurrency, "concurrency", 1, "")
	cmd.Flags().StringVar(&replayDelay, "delay", "", "")
	cmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "")

	if err := runReplay(cmd, nil); err != nil {
		t.Fatalf("runReplay() error: %v", err)
	}

	updated, err := store.Get(session.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if updated.Status != model.SessionReplayed {
		t.Fatalf("session status = %q, want %q", updated.Status, model.SessionReplayed)
	}

	replays, err := store.ListReplayRecords(session.ID)
	if err != nil {
		t.Fatalf("ListReplayRecords() error: %v", err)
	}
	if len(replays) != 1 {
		t.Fatalf("len(replays) = %d, want 1", len(replays))
	}
}

func TestRunReplay_ConfigDBProxyRequiresSerialConcurrency(t *testing.T) {
	withRuntimeConfig(t, func(cfg *config.AppConfig) {
		cfg.Replay.Concurrency = 2
		cfg.Replay.DBProxies = []config.DBProxyConfig{
			{Type: "mysql", ListenAddr: ":13307", TargetAddr: "127.0.0.1:3306"},
		}
	})

	replaySession = ""
	replayTarget = "http://example.com"
	replayConcurrency = 0
	replayDelay = ""
	replayDBProxy = nil

	cmd := &cobra.Command{}
	cmd.Flags().IntVar(&replayConcurrency, "concurrency", 1, "")
	cmd.Flags().StringVar(&replayDelay, "delay", "", "")
	cmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "")

	err := runReplay(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "require concurrency 1") {
		t.Fatalf("runReplay() error = %v, want config concurrency validation", err)
	}
}

func TestRunReplay_DBProxyRequiresSerialConcurrency(t *testing.T) {
	withRuntimeConfig(t, nil)

	replaySession = ""
	replayTarget = "http://example.com"
	replayConcurrency = 2
	replayDelay = ""
	replayDBProxy = nil

	cmd := &cobra.Command{}
	cmd.Flags().IntVar(&replayConcurrency, "concurrency", 1, "")
	cmd.Flags().StringVar(&replayDelay, "delay", "", "")
	cmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "")
	if err := cmd.Flags().Set("concurrency", "2"); err != nil {
		t.Fatalf("Set(concurrency) error: %v", err)
	}
	if err := cmd.Flags().Set("db-proxy", "mysql://:13307->127.0.0.1:3306"); err != nil {
		t.Fatalf("Set(db-proxy) error: %v", err)
	}

	err := runReplay(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "require concurrency 1") {
		t.Fatalf("runReplay() error = %v, want concurrency validation", err)
	}
}

func TestRunReplay_DBProxyStartsHooks(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "replay-db-proxy", Status: model.SessionRecording}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/hello"},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	fakeHook := &fakeDBHook{sideEffects: make(chan model.SideEffect)}
	oldFactory := newDBHook
	defer func() { newDBHook = oldFactory }()
	newDBHook = func(cfg dbhook.Config) (dbhook.DBHook, error) {
		return fakeHook, nil
	}

	replaySession = session.Name
	replayTarget = server.URL
	replayConcurrency = 1
	replayDelay = ""
	replayDBProxy = nil

	cmd := &cobra.Command{}
	cmd.Flags().IntVar(&replayConcurrency, "concurrency", 1, "")
	cmd.Flags().StringVar(&replayDelay, "delay", "", "")
	cmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "")
	if err := cmd.Flags().Set("db-proxy", "mysql://:13307->127.0.0.1:3306"); err != nil {
		t.Fatalf("Set(db-proxy) error: %v", err)
	}

	if err := runReplay(cmd, nil); err != nil {
		t.Fatalf("runReplay() error: %v", err)
	}
	if !fakeHook.startInvoked {
		t.Fatal("expected replay db hook to start")
	}
	if !fakeHook.stopped {
		t.Fatal("expected replay db hook to stop")
	}
}

func TestRunReplay_ConfigDBProxyStartsHooks(t *testing.T) {
	withRuntimeConfig(t, func(cfg *config.AppConfig) {
		cfg.Replay.DBProxies = []config.DBProxyConfig{
			{Type: "postgres", ListenAddr: ":15433", TargetAddr: "127.0.0.1:5432"},
		}
	})
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "replay-config-db-proxy", Status: model.SessionRecording}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/hello"},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	fakeHook := &fakeDBHook{sideEffects: make(chan model.SideEffect)}
	oldFactory := newDBHook
	defer func() { newDBHook = oldFactory }()
	var gotConfig dbhook.Config
	newDBHook = func(cfg dbhook.Config) (dbhook.DBHook, error) {
		gotConfig = cfg
		return fakeHook, nil
	}

	replaySession = session.Name
	replayTarget = server.URL
	replayConcurrency = 1
	replayDelay = ""
	replayDBProxy = nil

	cmd := &cobra.Command{}
	cmd.Flags().IntVar(&replayConcurrency, "concurrency", 1, "")
	cmd.Flags().StringVar(&replayDelay, "delay", "", "")
	cmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "")

	if err := runReplay(cmd, nil); err != nil {
		t.Fatalf("runReplay() error: %v", err)
	}
	if !fakeHook.startInvoked {
		t.Fatal("expected replay db hook to start from config")
	}
	if gotConfig.DBType != "postgres" || gotConfig.ListenAddr != ":15433" {
		t.Fatalf("unexpected hook config: %+v", gotConfig)
	}
	if !fakeHook.stopped {
		t.Fatal("expected replay db hook to stop")
	}
}

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

func TestRunSessionListShowDelete(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "session-case", Status: model.SessionCompleted}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	listOutput := captureStdout(t, func() {
		if err := runSessionList(&cobra.Command{}, nil); err != nil {
			t.Fatalf("runSessionList() error: %v", err)
		}
	})
	if !strings.Contains(listOutput, session.Name) {
		t.Fatalf("list output = %q, want session name", listOutput)
	}

	showOutput := captureStdout(t, func() {
		if err := runSessionShow(&cobra.Command{}, []string{session.ID}); err != nil {
			t.Fatalf("runSessionShow() error: %v", err)
		}
	})
	if !strings.Contains(showOutput, "ID:") {
		t.Fatalf("show output = %q, want session details", showOutput)
	}

	deleteOutput := captureStdout(t, func() {
		if err := runSessionDelete(&cobra.Command{}, []string{session.ID}); err != nil {
			t.Fatalf("runSessionDelete() error: %v", err)
		}
	})
	if !strings.Contains(deleteOutput, "Session deleted") {
		t.Fatalf("delete output = %q, want delete message", deleteOutput)
	}
}

func TestRunSessionInspect_PrintsArtifactAndSideEffectCounts(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "inspect-case", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "record-1",
		SessionID: session.ID,
		Sequence:  1,
		SideEffects: []model.SideEffect{
			model.NewSQLSideEffect("mysql", "SELECT 1", 0),
			model.NewMongoSideEffect(model.MongoSideEffect{Database: "app", Collection: "users", Operation: "find"}, 0),
		},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "replay-1",
		SessionID: session.ID,
		Sequence:  1,
		SideEffects: []model.SideEffect{
			model.NewRedisSideEffect("GET", "user:1", []string{"user:1"}, 0),
		},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}
	if err := store.SaveResults(session.ID, []model.DiffResult{{RecordID: "record-1", Sequence: 1}}); err != nil {
		t.Fatalf("SaveResults() error: %v", err)
	}

	sessionInspectFormat = "terminal"
	output := captureStdout(t, func() {
		if err := runSessionInspect(&cobra.Command{}, []string{session.ID}); err != nil {
			t.Fatalf("runSessionInspect() error: %v", err)
		}
	})
	for _, want := range []string{
		"Records: 1",
		"Replay records: 1",
		"Diff results: 1",
		"mysql: 1",
		"mongo: 1",
		"redis: 1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("inspect output missing %q:\n%s", want, output)
		}
	}
}

func TestRunSessionInspect_JSONWarningsForMissingReplayAndDiff(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "inspect-missing", Status: model.SessionCompleted}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{ID: "record-1", SessionID: session.ID}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	sessionInspectFormat = "json"
	var out strings.Builder
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runSessionInspect(cmd, []string{session.Name}); err != nil {
		t.Fatalf("runSessionInspect() error: %v", err)
	}

	var report sessionInspectReport
	if err := json.Unmarshal([]byte(out.String()), &report); err != nil {
		t.Fatalf("Unmarshal() error = %v\noutput=%s", err, out.String())
	}
	if report.RecordCount != 1 || report.ReplayRecordCount != 0 || report.DiffResultCount != 0 {
		t.Fatalf("unexpected counts: %+v", report)
	}
	if len(report.Warnings) != 2 {
		t.Fatalf("warnings = %v, want replay and diff warnings", report.Warnings)
	}
}

func TestRunRecordStatus_NoActiveSessions(t *testing.T) {
	withRuntimeConfig(t, nil)

	output := captureStdout(t, func() {
		if err := runRecordStatus(&cobra.Command{}, nil); err != nil {
			t.Fatalf("runRecordStatus() error: %v", err)
		}
	})
	if !strings.Contains(output, "No active recording sessions.") {
		t.Fatalf("output = %q, want no active sessions", output)
	}
}

func TestListRecordingSessions_PrintsActiveSession(t *testing.T) {
	dataDir := withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{
		Name:   "recording-case",
		Status: model.SessionRecording,
		Source: model.EndpointConfig{BaseURL: "http://old-api:8080"},
	}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := daemon.WritePID(filepath.Join(dataDir, "sessions", session.ID), os.Getpid()); err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}

	output := captureStdout(t, func() {
		if err := listRecordingSessions(store, dataDir); err != nil {
			t.Fatalf("listRecordingSessions() error: %v", err)
		}
	})
	if !strings.Contains(output, session.ID) {
		t.Fatalf("output = %q, want session id", output)
	}
	if !strings.Contains(output, "yes") {
		t.Fatalf("output = %q, want alive marker", output)
	}
}

func TestRunRecordStatus_ShowsSessionDetail(t *testing.T) {
	dataDir := withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{
		Name:        "detail-case",
		Status:      model.SessionRecording,
		RecordCount: 3,
		DaemonMode:  true,
		Source:      model.EndpointConfig{BaseURL: "http://old-api:8080"},
	}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := daemon.WritePID(filepath.Join(dataDir, "sessions", session.ID), os.Getpid()); err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}

	oldStatusSession := statusSession
	t.Cleanup(func() {
		statusSession = oldStatusSession
	})
	statusSession = session.ID

	output := captureStdout(t, func() {
		if err := runRecordStatus(&cobra.Command{}, nil); err != nil {
			t.Fatalf("runRecordStatus() error: %v", err)
		}
	})
	if !strings.Contains(output, "Session:") || !strings.Contains(output, session.ID) {
		t.Fatalf("output = %q, want session detail", output)
	}
	if !strings.Contains(output, "Process:  running") {
		t.Fatalf("output = %q, want running process line", output)
	}
}

func TestRunRecordStop_CleansStalePID(t *testing.T) {
	dataDir := withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{
		Name:   "stale-case",
		Status: model.SessionRecording,
		Source: model.EndpointConfig{BaseURL: "http://old-api:8080"},
	}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	sessionDir := filepath.Join(dataDir, "sessions", session.ID)
	if err := daemon.WritePID(sessionDir, 99999999); err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}

	oldStopSession := stopSession
	t.Cleanup(func() {
		stopSession = oldStopSession
	})
	stopSession = session.ID

	output := captureStdout(t, func() {
		if err := runRecordStop(&cobra.Command{}, nil); err != nil {
			t.Fatalf("runRecordStop() error: %v", err)
		}
	})
	if !strings.Contains(output, "Cleaned up stale pidfile") {
		t.Fatalf("output = %q, want stale pid cleanup message", output)
	}

	updated, err := store.Get(session.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if updated.Status != model.SessionCompleted {
		t.Fatalf("status = %q, want %q", updated.Status, model.SessionCompleted)
	}
	if updated.PID != 0 {
		t.Fatalf("PID = %d, want 0", updated.PID)
	}
	if _, err := os.Stat(filepath.Join(sessionDir, "pidfile")); !os.IsNotExist(err) {
		t.Fatalf("expected pidfile to be removed, err=%v", err)
	}
}
