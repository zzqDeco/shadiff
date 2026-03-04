package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const pidFileName = "pidfile"

// WritePID writes the process ID to the session's PID file.
func WritePID(sessionDir string, pid int) error {
	path := filepath.Join(sessionDir, pidFileName)
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

// ReadPID reads the process ID from the session's PID file.
func ReadPID(sessionDir string) (int, error) {
	path := filepath.Join(sessionDir, pidFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read pidfile: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse pidfile: %w", err)
	}
	return pid, nil
}

// RemovePID removes the PID file from the session directory.
func RemovePID(sessionDir string) error {
	path := filepath.Join(sessionDir, pidFileName)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// IsRunning checks whether the daemon process for the session is still alive.
func IsRunning(sessionDir string) bool {
	pid, err := ReadPID(sessionDir)
	if err != nil {
		return false
	}
	return isProcessAlive(pid)
}

// PIDFilePath returns the full path to the PID file for the given session directory.
func PIDFilePath(sessionDir string) string {
	return filepath.Join(sessionDir, pidFileName)
}
