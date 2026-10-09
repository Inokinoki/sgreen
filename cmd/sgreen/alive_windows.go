//go:build windows
// +build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// isProcessAliveByPID checks if a process is alive by PID. os.Process
// signaling on Windows only supports Kill, so a signal-0 probe would
// always report dead; open a query-only handle instead.
func isProcessAliveByPID(pid int) bool {
	if pid == os.Getpid() {
		return true
	}
	if pid <= 0 {
		return false
	}
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	var code uint32
	if err := windows.GetExitCodeProcess(handle, &code); err != nil {
		return false
	}
	// STILL_ACTIVE is 259; a live process reports it (a coincidental real
	// exit code of 259 is indistinguishable, same caveat as everywhere).
	return code == 259
}
