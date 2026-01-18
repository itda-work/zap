//go:build windows

package cli

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr sets Windows-specific process attributes for daemon mode
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
