//go:build !windows
// +build !windows

package pty

import (
	"syscall"
)

// IsAlive checks if the process is still running (Unix version)
func (p *PTYProcess) IsAlive() bool {
	if p.Cmd == nil || p.Cmd.Process == nil {
		// If we don't have a command reference, we can't check
		// In this case, assume it's alive if we can access the PTY
		return p.Pty != nil
	}
	
	// Check if process is alive by sending signal 0 (doesn't actually send a signal)
	err := p.Cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

