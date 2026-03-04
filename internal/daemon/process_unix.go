//go:build !windows

package daemon

import (
	"os"
	"os/exec"
	"syscall"
)

// Detach configures the command to run in a new session, detached from the
// parent terminal. On Unix this sets Setsid so the child gets its own
// session and process group.
func Detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}

// isProcessAlive checks whether a process with the given PID is running.
func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, sending signal 0 checks existence without affecting the process.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// SendStop sends an interrupt signal (SIGTERM) to the process.
func SendStop(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

// ForceKill forcefully terminates the process.
func ForceKill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGKILL)
}
