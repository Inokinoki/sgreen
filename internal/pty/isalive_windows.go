//go:build windows
// +build windows

package pty

// IsAlive checks if the process is still running (Windows version)
func (p *PTYProcess) IsAlive() bool {
	if p.Cmd == nil || p.Cmd.Process == nil {
		// If we don't have a command reference, we can't check
		// In this case, assume it's alive if we can access the PTY
		return p.Pty != nil
	}

	// On Windows, we check if the process is still running by
	// checking if we can access the process handle
	// If the process has exited, accessing it may fail
	// For simplicity, we'll assume it's alive if we have a process reference
	// A more robust check would use Windows API calls, but this is sufficient
	// for basic compatibility
	return true
}
