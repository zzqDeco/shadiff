package cmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"shadiff/internal/config"
	"shadiff/internal/storage"
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
