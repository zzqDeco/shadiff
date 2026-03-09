package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"shadiff/internal/config"
)

func TestInitRuntime_UsesExplicitConfigPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	dataDir := filepath.Join(dir, "data")
	logDir := filepath.Join(dir, "custom-logs")
	data := `{
  "storage": {"dataDir": "` + filepath.ToSlash(dataDir) + `"},
  "log": {"level": "warn", "logDir": "` + filepath.ToSlash(logDir) + `"}
}`
	if err := os.WriteFile(configPath, []byte(data), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	oldCfgFile := cfgFile
	oldRuntime := runtimeCtx
	oldVerbose := verbose
	oldQuiet := quiet
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		runtimeCtx = oldRuntime
		verbose = oldVerbose
		quiet = oldQuiet
	})

	cfgFile = configPath
	runtimeCtx = nil
	if err := initRuntime(); err != nil {
		t.Fatalf("initRuntime() error: %v", err)
	}

	if currentConfigPath() != configPath {
		t.Fatalf("config path = %q, want %q", currentConfigPath(), configPath)
	}
	if filepath.Clean(currentDataDir()) != filepath.Clean(dataDir) {
		t.Fatalf("data dir = %q, want %q", currentDataDir(), dataDir)
	}
	if filepath.Clean(currentLogDir()) != filepath.Clean(logDir) {
		t.Fatalf("log dir = %q, want %q", currentLogDir(), logDir)
	}
}

func TestEffectiveLogLevel_RespectsFlags(t *testing.T) {
	oldRuntime := runtimeCtx
	oldVerbose := verbose
	oldQuiet := quiet
	t.Cleanup(func() {
		runtimeCtx = oldRuntime
		verbose = oldVerbose
		quiet = oldQuiet
	})

	runtimeCtx = &appRuntime{
		Config: &config.AppConfig{
			Log: config.LogConfig{Level: "warn"},
		},
	}

	if got := effectiveLogLevel(); got != "warn" {
		t.Fatalf("effectiveLogLevel() = %q, want %q", got, "warn")
	}

	verbose = true
	if got := effectiveLogLevel(); got != "debug" {
		t.Fatalf("effectiveLogLevel() with verbose = %q, want %q", got, "debug")
	}

	quiet = true
	if got := effectiveLogLevel(); got != "error" {
		t.Fatalf("effectiveLogLevel() with quiet = %q, want %q", got, "error")
	}
}
