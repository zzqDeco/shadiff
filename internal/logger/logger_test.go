package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readLatestLog(t *testing.T, dir string) string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	return string(data)
}

func TestInit_RespectsLogLevel(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, "warn"); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	defer Close()

	Debug("debug-msg")
	Info("info-msg")
	Warn("warn-msg")
	Close()

	content := readLatestLog(t, dir)
	if strings.Contains(content, "debug-msg") {
		t.Fatal("expected debug message to be filtered out")
	}
	if strings.Contains(content, "info-msg") {
		t.Fatal("expected info message to be filtered out")
	}
	if !strings.Contains(content, "warn-msg") {
		t.Fatal("expected warn message to be written")
	}
}

func TestInit_DaemonModeWritesToFile(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, "info", true); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	defer Close()

	Info("daemon-msg")
	Close()

	content := readLatestLog(t, dir)
	if !strings.Contains(content, "daemon-msg") {
		t.Fatal("expected daemon log to be written to file")
	}
}

func TestL_ReturnsDefaultWhenUninitialized(t *testing.T) {
	Close()
	if L() == nil {
		t.Fatal("expected default logger when instance is nil")
	}
}

func TestConvenienceLoggingHelpersWriteEntries(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, "info"); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	defer Close()

	CaptureEvent("captured", "path", "/ok")
	ReplayEvent("replayed", "path", "/ok")
	DiffEvent("diffed", "path", "/ok")
	DBHookEvent("started", "mysql", "listen", ":13306")
	SessionEvent("updated", "sess-1")
	Error("oops", errTest)
	Close()

	content := readLatestLog(t, dir)
	for _, want := range []string{"[CAPTURE]", "[REPLAY]", "[DIFF]", "[DBHOOK]", "[SESSION]", "oops"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected log to contain %q\ncontent=%s", want, content)
		}
	}
}
