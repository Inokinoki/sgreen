//go:build !windows
// +build !windows

package pty

import (
	"os/exec"
	"syscall"
)

// setProcessGroup sets the process group for the command
// This ensures child processes are in their own process group
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Create a new process group
	cmd.SysProcAttr.Setpgid = true
	// Set process group ID to 0 (creates new group)
	cmd.SysProcAttr.Pgid = 0
}
