//go:build !windows
// +build !windows

package daemon

import (
	"os"
	"os/signal"
	"syscall"
)

// notifySignals subscribes the daemon to termination signals.
func notifySignals(ch chan<- os.Signal) {
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
}

// killPid kills a process and its process group. The session shell is
// wrapped in nohup, so closing the master alone would not stop it.
func killPid(pid int) {
	if pid <= 0 {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
