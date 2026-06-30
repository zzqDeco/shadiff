package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shadiff/internal/daemon"
	"shadiff/internal/model"

	"github.com/spf13/cobra"
)

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
