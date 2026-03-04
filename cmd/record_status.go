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

var statusSession string

var recordStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show recording session status",
	Long: `Show the status of daemon recording sessions.

Without -s: list all sessions with status "recording" and whether the process is alive.
With -s: show details for a specific session.

Examples:
  shadiff record status
  shadiff record status -s my-session`,
	RunE: runRecordStatus,
}

func init() {
	recordStatusCmd.Flags().StringVarP(&statusSession, "session", "s", "", "Session name or ID (optional)")
	recordCmd.AddCommand(recordStatusCmd)
}

func runRecordStatus(cmd *cobra.Command, args []string) error {
	homeDir, _ := os.UserHomeDir()
	dataDir := homeDir + "/.shadiff"

	store, err := storage.NewFileStore(dataDir)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	if statusSession != "" {
		return showSessionDetail(store, dataDir, statusSession)
	}

	return listRecordingSessions(store, dataDir)
}

func listRecordingSessions(store *storage.FileStore, dataDir string) error {
	sessions, err := store.List(&model.SessionFilter{Status: string(model.SessionRecording)})
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No active recording sessions.")
		return nil
	}

	fmt.Printf("%-10s %-25s %-8s %-8s %s\n", "ID", "NAME", "PID", "ALIVE", "CREATED")
	fmt.Printf("%-10s %-25s %-8s %-8s %s\n", "---", "---", "---", "---", "---")

	for _, s := range sessions {
		sessionDir := filepath.Join(dataDir, "sessions", s.ID)
		pid, _ := daemon.ReadPID(sessionDir)
		alive := daemon.IsRunning(sessionDir)

		aliveStr := "no"
		if alive {
			aliveStr = "yes"
		}

		pidStr := "-"
		if pid > 0 {
			pidStr = fmt.Sprintf("%d", pid)
		}

		created := time.UnixMilli(s.CreatedAt).Format("2006-01-02 15:04:05")
		name := s.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		fmt.Printf("%-10s %-25s %-8s %-8s %s\n", s.ID, name, pidStr, aliveStr, created)
	}

	return nil
}

func showSessionDetail(store *storage.FileStore, dataDir string, nameOrID string) error {
	session, err := findSession(store, nameOrID)
	if err != nil {
		return err
	}

	sessionDir := filepath.Join(dataDir, "sessions", session.ID)
	pid, _ := daemon.ReadPID(sessionDir)
	alive := daemon.IsRunning(sessionDir)

	fmt.Printf("Session:  %s\n", session.ID)
	fmt.Printf("Name:     %s\n", session.Name)
	fmt.Printf("Status:   %s\n", session.Status)
	fmt.Printf("Daemon:   %v\n", session.DaemonMode)

	if pid > 0 {
		fmt.Printf("PID:      %d\n", pid)
		if alive {
			fmt.Printf("Process:  running\n")
		} else {
			fmt.Printf("Process:  dead (stale pidfile)\n")
		}
	} else {
		fmt.Printf("PID:      -\n")
	}

	fmt.Printf("Records:  %d\n", session.RecordCount)
	fmt.Printf("Target:   %s\n", session.Source.BaseURL)

	created := time.UnixMilli(session.CreatedAt).Format("2006-01-02 15:04:05")
	fmt.Printf("Created:  %s\n", created)

	if session.CreatedAt > 0 {
		uptime := time.Since(time.UnixMilli(session.CreatedAt)).Truncate(time.Second)
		fmt.Printf("Uptime:   %s\n", uptime)
	}

	return nil
}
