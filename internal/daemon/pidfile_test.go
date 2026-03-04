package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadPID(t *testing.T) {
	dir := t.TempDir()

	pid := 12345
	if err := WritePID(dir, pid); err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	got, err := ReadPID(dir)
	if err != nil {
		t.Fatalf("ReadPID failed: %v", err)
	}
	if got != pid {
		t.Errorf("ReadPID = %d, want %d", got, pid)
	}
}

func TestReadPID_NotExist(t *testing.T) {
	dir := t.TempDir()

	_, err := ReadPID(dir)
	if err == nil {
		t.Fatal("ReadPID should fail when no pidfile exists")
	}
}

func TestReadPID_InvalidContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pidfile")
	if err := os.WriteFile(path, []byte("not-a-number"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := ReadPID(dir)
	if err == nil {
		t.Fatal("ReadPID should fail on invalid content")
	}
}

func TestRemovePID(t *testing.T) {
	dir := t.TempDir()

	if err := WritePID(dir, 99); err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	if err := RemovePID(dir); err != nil {
		t.Fatalf("RemovePID failed: %v", err)
	}

	// File should no longer exist
	path := filepath.Join(dir, "pidfile")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("pidfile should be removed, got err: %v", err)
	}
}

func TestRemovePID_NotExist(t *testing.T) {
	dir := t.TempDir()

	// Should not error when pidfile doesn't exist
	if err := RemovePID(dir); err != nil {
		t.Fatalf("RemovePID on missing file should succeed, got: %v", err)
	}
}

func TestIsRunning_NoPidfile(t *testing.T) {
	dir := t.TempDir()

	if IsRunning(dir) {
		t.Error("IsRunning should return false when no pidfile exists")
	}
}

func TestIsRunning_CurrentProcess(t *testing.T) {
	dir := t.TempDir()

	// Write our own PID — this process is definitely alive
	if err := WritePID(dir, os.Getpid()); err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	if !IsRunning(dir) {
		t.Error("IsRunning should return true for current process")
	}
}

func TestIsRunning_DeadProcess(t *testing.T) {
	dir := t.TempDir()

	// Use a very high PID that's unlikely to exist
	if err := WritePID(dir, 99999999); err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	if IsRunning(dir) {
		t.Error("IsRunning should return false for dead process")
	}
}

func TestPIDFilePath(t *testing.T) {
	dir := "/tmp/test-session"
	want := filepath.Join(dir, "pidfile")
	got := PIDFilePath(dir)
	if got != want {
		t.Errorf("PIDFilePath = %q, want %q", got, want)
	}
}
