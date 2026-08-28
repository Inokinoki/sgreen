//go:build windows
// +build windows

// Package daemon: session relay daemons are not supported on Windows yet.
// The rest of the code falls back to in-process attach behavior.
package daemon

import (
	"errors"
	"net"
)

// Supported reports whether session daemons are available on this platform.
func Supported() bool { return false }

// RunFromEnv is a no-op on Windows; this process is never a session daemon.
func RunFromEnv() bool { return false }

// Status describes a session daemon's live state.
type Status struct {
	Attached bool
	Count    int
}

// ErrNoDaemon indicates no session daemon is listening on the socket.
var ErrNoDaemon = errors.New("no session daemon")

// QueryStatus always fails on Windows.
func QueryStatus(socketPath string) (*Status, error) { return nil, ErrNoDaemon }

// SendDetach always fails on Windows.
func SendDetach(socketPath string, power bool) error { return ErrNoDaemon }

// SendQuit always fails on Windows.
func SendQuit(socketPath string) error { return ErrNoDaemon }

// SendStuff always fails on Windows.
func SendStuff(socketPath string, data string) error { return ErrNoDaemon }

// SendResize always fails on Windows.
func SendResize(socketPath string, rows, cols uint16) error { return ErrNoDaemon }

// OpenAttach always fails on Windows.
func OpenAttach(socketPath string) (net.Conn, error) { return nil, ErrNoDaemon }
