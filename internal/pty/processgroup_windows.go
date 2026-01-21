//go:build windows
// +build windows

package pty

import "os/exec"

// setProcessGroup is a no-op on Windows
func setProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't have process groups in the same way
	// No-op
}

