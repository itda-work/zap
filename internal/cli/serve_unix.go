//go:build unix

package cli

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr sets Unix-specific process attributes for daemon mode
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
	}
}
