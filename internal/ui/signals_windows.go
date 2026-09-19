//go:build windows
// +build windows

package ui

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
)

// notifyResizeSignals is a no-op on Windows: console size events arrive
// through the input stream, not signals (polling-based handling can be
// added later).
func notifyResizeSignals(chan<- os.Signal) {}

// notifyHangupSignals subscribes to the closest Windows equivalent of a
// hangup: the console ctrl events the runtime emulates as os.Interrupt.
func notifyHangupSignals(ch chan<- os.Signal) {
	signal.Notify(ch, os.Interrupt)
}

// notifyTermSignals subscribes to termination signals.
func notifyTermSignals(ch chan<- os.Signal) {
	signal.Notify(ch, syscall.SIGTERM, os.Interrupt)
}

// suspendSelf has no Windows equivalent (job control is Unix-only).
func suspendSelf() error {
	return errors.New("suspend is not supported on this platform")
}
