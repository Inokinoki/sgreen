//go:build !windows
// +build !windows

package main

import (
	"os"
	"syscall"
)

// isProcessAliveByPID checks if a process is alive by PID. On Unix a
// signal-0 probe does the job without side effects.
func isProcessAliveByPID(pid int) bool {
	if pid == os.Getpid() {
		return true
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
