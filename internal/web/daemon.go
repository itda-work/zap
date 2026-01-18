package web

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

const (
	pidFileName = ".zap-serve.pid"
	logFileName = ".zap-serve.log"
)

// DaemonInfo holds information about a running daemon
type DaemonInfo struct {
	PID     int
	Port    int
	Running bool
}

// GetPidFilePath returns the path to the PID file
func GetPidFilePath(issuesDir string) string {
	return filepath.Join(issuesDir, pidFileName)
}

// GetLogFilePath returns the path to the log file
func GetLogFilePath(issuesDir string) string {
	return filepath.Join(issuesDir, logFileName)
}

// WritePidFile writes the PID and port to the PID file
func WritePidFile(issuesDir string, pid, port int) error {
	pidFile := GetPidFilePath(issuesDir)
	content := fmt.Sprintf("%d\n%d\n", pid, port)
	return os.WriteFile(pidFile, []byte(content), 0644)
}

// ReadPidFile reads the PID and port from the PID file
func ReadPidFile(issuesDir string) (*DaemonInfo, error) {
	pidFile := GetPidFilePath(issuesDir)

	file, err := os.Open(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Read PID
	if !scanner.Scan() {
		return nil, fmt.Errorf("invalid PID file format")
	}
	pid, err := strconv.Atoi(scanner.Text())
	if err != nil {
		return nil, fmt.Errorf("invalid PID: %w", err)
	}

	// Read port
	port := 8080 // default
	if scanner.Scan() {
		port, _ = strconv.Atoi(scanner.Text())
	}

	info := &DaemonInfo{
		PID:     pid,
		Port:    port,
		Running: IsProcessRunning(pid),
	}

	return info, nil
}

// RemovePidFile removes the PID file
func RemovePidFile(issuesDir string) error {
	pidFile := GetPidFilePath(issuesDir)
	err := os.Remove(pidFile)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// IsProcessRunning checks if a process with the given PID is running
func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// StopDaemon stops the running daemon
func StopDaemon(issuesDir string) error {
	info, err := ReadPidFile(issuesDir)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	if info == nil {
		return fmt.Errorf("no daemon is running (PID file not found)")
	}

	if !info.Running {
		// Clean up stale PID file
		RemovePidFile(issuesDir)
		return fmt.Errorf("daemon is not running (stale PID file cleaned up)")
	}

	process, err := os.FindProcess(info.PID)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	// Remove PID file after stopping
	RemovePidFile(issuesDir)

	return nil
}

// GetDaemonStatus returns the status of the daemon
func GetDaemonStatus(issuesDir string) (*DaemonInfo, error) {
	info, err := ReadPidFile(issuesDir)
	if err != nil {
		return nil, err
	}

	if info == nil {
		return &DaemonInfo{Running: false}, nil
	}

	// Update running status
	info.Running = IsProcessRunning(info.PID)

	// Clean up stale PID file
	if !info.Running {
		RemovePidFile(issuesDir)
	}

	return info, nil
}
