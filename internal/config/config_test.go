package config

import (
	"os"
	"path/filepath"
	"testing"
)

// --- DefaultConfig ---

func TestDefaultConfig_ReturnsNonNil(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
}

func TestDefaultConfig_CaptureDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Capture.ListenAddr != ":18080" {
		t.Errorf("Capture.ListenAddr = %q, want %q", cfg.Capture.ListenAddr, ":18080")
	}
	if cfg.Capture.MaxBodySize != 10*1024*1024 {
		t.Errorf("Capture.MaxBodySize = %d, want %d", cfg.Capture.MaxBodySize, 10*1024*1024)
	}
	if cfg.Capture.ExcludePaths != nil {
		t.Errorf("Capture.ExcludePaths = %v, want nil", cfg.Capture.ExcludePaths)
	}
	if cfg.Capture.DBProxies != nil {
		t.Errorf("Capture.DBProxies = %v, want nil", cfg.Capture.DBProxies)
	}
}

func TestDefaultConfig_ReplayDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Replay.Concurrency != 1 {
		t.Errorf("Replay.Concurrency = %d, want 1", cfg.Replay.Concurrency)
	}
	if cfg.Replay.Timeout != "30s" {
		t.Errorf("Replay.Timeout = %q, want %q", cfg.Replay.Timeout, "30s")
	}
	if cfg.Replay.RetryCount != 0 {
		t.Errorf("Replay.RetryCount = %d, want 0", cfg.Replay.RetryCount)
	}
	if cfg.Replay.DelayMs != 0 {
		t.Errorf("Replay.DelayMs = %d, want 0", cfg.Replay.DelayMs)
	}
}

func TestDefaultConfig_DiffDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Diff.MaxDiffs != 1000 {
		t.Errorf("Diff.MaxDiffs = %d, want 1000", cfg.Diff.MaxDiffs)
	}
	if !cfg.Diff.IgnoreOrder {
		// Default is false (zero value)
	}
	expectedHeaders := []string{"Date", "X-Request-Id", "X-Trace-Id", "Server", "Content-Length"}
	if len(cfg.Diff.IgnoreHeaders) != len(expectedHeaders) {
		t.Fatalf("len(Diff.IgnoreHeaders) = %d, want %d", len(cfg.Diff.IgnoreHeaders), len(expectedHeaders))
	}
	for i, h := range expectedHeaders {
		if cfg.Diff.IgnoreHeaders[i] != h {
			t.Errorf("Diff.IgnoreHeaders[%d] = %q, want %q", i, cfg.Diff.IgnoreHeaders[i], h)
		}
	}
}

func TestDefaultConfig_StorageDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Storage.MaxSessions != 100 {
		t.Errorf("Storage.MaxSessions = %d, want 100", cfg.Storage.MaxSessions)
	}
	if cfg.Storage.DataDir != "" {
		t.Errorf("Storage.DataDir = %q, want empty", cfg.Storage.DataDir)
	}
}

func TestDefaultConfig_LogDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
	if cfg.Log.LogDir != "" {
		t.Errorf("Log.LogDir = %q, want empty", cfg.Log.LogDir)
	}
}

func TestDefaultConfig_IndependentInstances(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()
	cfg1.Capture.ListenAddr = ":9999"
	if cfg2.Capture.ListenAddr == ":9999" {
		t.Error("DefaultConfig should return independent instances")
	}
}

// --- Store ---

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s := &Store{
		path:   filepath.Join(dir, "config.json"),
		config: DefaultConfig(),
	}
	return s, dir
}

func TestStore_SaveAndLoad(t *testing.T) {
	s, _ := newTestStore(t)

	// Modify config before saving
	s.config.Capture.ListenAddr = ":9999"
	s.config.Log.Level = "debug"

	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Create a new store pointing to the same file
	s2 := &Store{
		path:   s.path,
		config: DefaultConfig(),
	}
	if err := s2.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if s2.config.Capture.ListenAddr != ":9999" {
		t.Errorf("after Load, ListenAddr = %q, want %q", s2.config.Capture.ListenAddr, ":9999")
	}
	if s2.config.Log.Level != "debug" {
		t.Errorf("after Load, Log.Level = %q, want %q", s2.config.Log.Level, "debug")
	}
}

func TestStore_LoadNonExistentFile(t *testing.T) {
	s := &Store{
		path:   filepath.Join(t.TempDir(), "nonexistent", "config.json"),
		config: DefaultConfig(),
	}
	err := s.Load()
	if err == nil {
		t.Error("Load() should return error for nonexistent file")
	}
}

func TestStore_LoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("failed to write invalid json: %v", err)
	}
	s := &Store{
		path:   path,
		config: DefaultConfig(),
	}
	err := s.Load()
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestStore_LoadPreservesDefaults(t *testing.T) {
	// Save a partial config and verify that defaults fill in the gaps
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	partialJSON := []byte(`{"capture":{"listenAddr":":7777"}}`)
	if err := os.WriteFile(path, partialJSON, 0644); err != nil {
		t.Fatalf("failed to write partial json: %v", err)
	}
	s := &Store{
		path:   path,
		config: DefaultConfig(),
	}
	if err := s.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if s.config.Capture.ListenAddr != ":7777" {
		t.Errorf("ListenAddr = %q, want %q", s.config.Capture.ListenAddr, ":7777")
	}
	// Defaults should be preserved for fields not in the partial JSON
	if s.config.Replay.Concurrency != 1 {
		t.Errorf("Replay.Concurrency = %d, want 1 (default)", s.config.Replay.Concurrency)
	}
	if s.config.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q (default)", s.config.Log.Level, "info")
	}
}

func TestStore_Get_ReturnsCopy(t *testing.T) {
	s, _ := newTestStore(t)
	cfg1 := s.Get()
	cfg1.Capture.ListenAddr = ":changed"

	cfg2 := s.Get()
	if cfg2.Capture.ListenAddr == ":changed" {
		t.Error("Get() should return a copy, not a reference to the internal config")
	}
}

func TestStore_Get_ReflectsCurrentState(t *testing.T) {
	s, _ := newTestStore(t)
	cfg := s.Get()
	if cfg.Capture.ListenAddr != ":18080" {
		t.Errorf("Get().Capture.ListenAddr = %q, want %q", cfg.Capture.ListenAddr, ":18080")
	}
}

func TestStore_Update_ModifiesAndPersists(t *testing.T) {
	s, _ := newTestStore(t)

	err := s.Update(func(cfg *AppConfig) {
		cfg.Replay.Concurrency = 8
		cfg.Capture.ListenAddr = ":5555"
	})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	// Verify in-memory state
	cfg := s.Get()
	if cfg.Replay.Concurrency != 8 {
		t.Errorf("after Update, Replay.Concurrency = %d, want 8", cfg.Replay.Concurrency)
	}
	if cfg.Capture.ListenAddr != ":5555" {
		t.Errorf("after Update, Capture.ListenAddr = %q, want %q", cfg.Capture.ListenAddr, ":5555")
	}

	// Verify persistence by loading from file
	s2 := &Store{
		path:   s.path,
		config: DefaultConfig(),
	}
	if err := s2.Load(); err != nil {
		t.Fatalf("Load() error after Update: %v", err)
	}
	cfg2 := s2.Get()
	if cfg2.Replay.Concurrency != 8 {
		t.Errorf("after Load, Replay.Concurrency = %d, want 8", cfg2.Replay.Concurrency)
	}
}

func TestStore_RoundTrip(t *testing.T) {
	s, _ := newTestStore(t)

	// Update
	err := s.Update(func(cfg *AppConfig) {
		cfg.Capture.ListenAddr = ":11111"
		cfg.Diff.MaxDiffs = 500
		cfg.Storage.MaxSessions = 50
		cfg.Replay.Timeout = "10s"
		cfg.Log.Level = "error"
	})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	// Save is implicit in Update, but call Save explicitly too
	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load into new store
	s2 := &Store{
		path:   s.path,
		config: DefaultConfig(),
	}
	if err := s2.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cfg := s2.Get()
	if cfg.Capture.ListenAddr != ":11111" {
		t.Errorf("ListenAddr = %q, want %q", cfg.Capture.ListenAddr, ":11111")
	}
	if cfg.Diff.MaxDiffs != 500 {
		t.Errorf("MaxDiffs = %d, want 500", cfg.Diff.MaxDiffs)
	}
	if cfg.Storage.MaxSessions != 50 {
		t.Errorf("MaxSessions = %d, want 50", cfg.Storage.MaxSessions)
	}
	if cfg.Replay.Timeout != "10s" {
		t.Errorf("Timeout = %q, want %q", cfg.Replay.Timeout, "10s")
	}
	if cfg.Log.Level != "error" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "error")
	}
}

// --- DataDir ---

func TestStore_DataDir_DefaultsToHomeShadiff(t *testing.T) {
	s, _ := newTestStore(t)
	dir := s.DataDir()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error: %v", err)
	}
	expected := filepath.Join(homeDir, ".shadiff")
	if dir != expected {
		t.Errorf("DataDir() = %q, want %q", dir, expected)
	}
}

func TestStore_DataDir_UsesConfiguredValue(t *testing.T) {
	s, _ := newTestStore(t)
	customDir := filepath.Join(t.TempDir(), "custom-data")
	err := s.Update(func(cfg *AppConfig) {
		cfg.Storage.DataDir = customDir
	})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	if s.DataDir() != customDir {
		t.Errorf("DataDir() = %q, want %q", s.DataDir(), customDir)
	}
}

func TestStore_DataDir_EmptyFallsBackToDefault(t *testing.T) {
	s, _ := newTestStore(t)
	// Explicitly set to empty
	err := s.Update(func(cfg *AppConfig) {
		cfg.Storage.DataDir = ""
	})
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".shadiff")
	if s.DataDir() != expected {
		t.Errorf("DataDir() = %q, want %q", s.DataDir(), expected)
	}
}

// --- Config structs ---

func TestRule_FieldAssignment(t *testing.T) {
	r := Rule{
		Name:    "ignore-timestamps",
		Kind:    "ignore",
		Paths:   []string{"body.createdAt", "body.updatedAt"},
		Pattern: `\d{13}`,
		Matcher: "regex",
	}
	if r.Name != "ignore-timestamps" {
		t.Errorf("Name = %q, want %q", r.Name, "ignore-timestamps")
	}
	if r.Kind != "ignore" {
		t.Errorf("Kind = %q, want %q", r.Kind, "ignore")
	}
	if len(r.Paths) != 2 {
		t.Fatalf("len(Paths) = %d, want 2", len(r.Paths))
	}
	if r.Pattern != `\d{13}` {
		t.Errorf("Pattern = %q, want expected", r.Pattern)
	}
	if r.Matcher != "regex" {
		t.Errorf("Matcher = %q, want %q", r.Matcher, "regex")
	}
}

func TestDBProxyConfig_FieldAssignment(t *testing.T) {
	dbc := DBProxyConfig{
		Type:       "mysql",
		ListenAddr: ":13306",
		TargetAddr: "localhost:3306",
	}
	if dbc.Type != "mysql" {
		t.Errorf("Type = %q, want %q", dbc.Type, "mysql")
	}
	if dbc.ListenAddr != ":13306" {
		t.Errorf("ListenAddr = %q, want %q", dbc.ListenAddr, ":13306")
	}
	if dbc.TargetAddr != "localhost:3306" {
		t.Errorf("TargetAddr = %q, want %q", dbc.TargetAddr, "localhost:3306")
	}
}
