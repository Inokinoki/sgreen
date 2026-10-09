//go:build !windows
// +build !windows

package ui

import (
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

// notifyResizeSignals subscribes to terminal resize notifications.
func notifyResizeSignals(ch chan<- os.Signal) {
	signal.Notify(ch, unix.SIGWINCH)
}

// notifyHangupSignals subscribes to terminal hangup (autodetach trigger).
func notifyHangupSignals(ch chan<- os.Signal) {
	signal.Notify(ch, unix.SIGHUP)
}

// notifyTermSignals subscribes to termination signals.
func notifyTermSignals(ch chan<- os.Signal) {
	signal.Notify(ch, unix.SIGTERM, unix.SIGINT)
}

// suspendSelf suspends the attach process (C-a s, SIGTSTP).
func suspendSelf() error {
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return proc.Signal(unix.SIGTSTP)
}
