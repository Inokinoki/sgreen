//go:build windows
// +build windows

package daemon

import (
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

// notifySignals subscribes the daemon to termination signals. A detached
// Windows process has no console, so ctrl events never arrive; this is a
// best-effort subscription (shutdown leaves stale files for -wipe).
func notifySignals(ch chan<- os.Signal) {
	// os.Interrupt is the only emulated signal on Windows without a console.
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
}

// killPid terminates a process and its children (taskkill walks the tree;
// TerminateProcess alone would orphan grandchildren).
func killPid(pid int) {
	if pid <= 0 {
		return
	}
	_ = exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
}
