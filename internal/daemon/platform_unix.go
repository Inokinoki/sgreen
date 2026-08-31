//go:build !windows
// +build !windows

package daemon

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// notifySignals subscribes the daemon to termination signals.
func notifySignals(ch chan<- os.Signal) {
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
}

// setMasterSize applies a window size to the PTY master.
func setMasterSize(master *os.File, rows, cols uint16) error {
	return unix.IoctlSetWinsize(int(master.Fd()), unix.TIOCSWINSZ, &unix.Winsize{
		Row: rows,
		Col: cols,
	})
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
