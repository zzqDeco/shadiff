package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"shadiff/internal/daemon"
	"shadiff/internal/model"
	"shadiff/internal/storage"

	"github.com/spf13/cobra"
)

var stopSession string

var recordStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a daemon recording session",
	Long: `Stop a recording session that is running as a background daemon.

Examples:
  shadiff record stop -s my-session
  shadiff record stop -s a1b2c3d4`,
	RunE: runRecordStop,
}

func init() {
	recordStopCmd.Flags().StringVarP(&stopSession, "session", "s", "", "Session name or ID (required)")
	recordStopCmd.MarkFlagRequired("session")
	recordCmd.AddCommand(recordStopCmd)
}

func runRecordStop(cmd *cobra.Command, args []string) error {
	homeDir, _ := os.UserHomeDir()
	dataDir := homeDir + "/.shadiff"

	store, err := storage.NewFileStore(dataDir)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	// Resolve session by ID or name
	session, err := findSession(store, stopSession)
	if err != nil {
		return err
	}

	sessionDir := filepath.Join(dataDir, "sessions", session.ID)

	// Read PID
	pid, err := daemon.ReadPID(sessionDir)
	if err != nil {
		return fmt.Errorf("no pidfile found for session %s — is it running as a daemon?", session.ID)
	}

	// Check if process is alive
	if !daemon.IsRunning(sessionDir) {
		// Stale PID file — clean up
		daemon.RemovePID(sessionDir)
		session.Status = model.SessionCompleted
		session.PID = 0
		store.Update(session)
		fmt.Printf("Session %s: process (PID %d) is no longer running. Cleaned up stale pidfile.\n", session.ID, pid)
		return nil
	}

	// Send stop signal
	fmt.Printf("Sending stop signal to session %s (PID %d)...\n", session.ID, pid)
	if err := daemon.SendStop(pid); err != nil {
		return fmt.Errorf("failed to send stop signal: %w", err)
	}

	// Wait for process to exit (poll up to 10 seconds)
	stopped := false
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if !daemon.IsRunning(sessionDir) {
			stopped = true
			break
		}
	}

	if !stopped {
		// Force kill
		fmt.Printf("Process did not exit gracefully, force killing PID %d...\n", pid)
		if err := daemon.ForceKill(pid); err != nil {
			return fmt.Errorf("failed to force kill: %w", err)
		}
		daemon.RemovePID(sessionDir)
		session.Status = model.SessionCompleted
		session.PID = 0
		store.Update(session)
	}

	fmt.Printf("Recording stopped for session %s\n", session.ID)
	return nil
}

// findSession finds a session by exact ID match or name match, returning the full Session object.
func findSession(store *storage.FileStore, nameOrID string) (*model.Session, error) {
	// Try direct ID lookup first
	session, err := store.Get(nameOrID)
	if err == nil {
		return session, nil
	}

	// Fall back to name search
	sessions, err := store.List(&model.SessionFilter{Name: nameOrID})
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		return nil, fmt.Errorf("no session found matching %q", nameOrID)
	}

	if len(sessions) > 1 {
		fmt.Printf("Multiple sessions match %q:\n", nameOrID)
		for _, s := range sessions {
			fmt.Printf("  %s  %s  [%s]\n", s.ID, s.Name, s.Status)
		}
		return nil, fmt.Errorf("please specify the exact session ID")
	}

	return &sessions[0], nil
}
