//go:build windows
// +build windows

// Package daemon: session relay daemons are not supported on Windows yet.
// The rest of the code falls back to in-process attach behavior.
package daemon

import (
	"errors"
	"fmt"

	"github.com/inoki/sgreen/internal/pty"
	"github.com/inoki/sgreen/internal/session"
)

// Supported reports whether session daemons are available on this platform.
func Supported() bool { return false }

// RunFromEnv is a no-op on Windows; this process is never a session daemon.
func RunFromEnv() bool { return false }

// Status describes a session daemon's live state.
type Status struct {
	Attached bool
	Count    int
	Current  int
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

// SendResizeWindow always fails on Windows.
func SendResizeWindow(socketPath string, winID int, rows, cols uint16) error { return ErrNoDaemon }

// OpenAttach always fails on Windows; callers treat this as "no daemon".
func OpenAttach(socketPath string, winID int) (conn any, err error) {
	return nil, ErrNoDaemon
}

// Controller is a stub on Windows.
type Controller struct{}

// NewController returns a stub controller on Windows.
func NewController(socketPath string) *Controller { return &Controller{} }

// CreateWindow always fails on Windows.
func (c *Controller) CreateWindow(cmdPath string, args []string, term string) (int, error) {
	return 0, ErrNoDaemon
}

// SwitchWindow always fails on Windows.
func (c *Controller) SwitchWindow(op string, arg string) (int, error) { return 0, ErrNoDaemon }

// KillWindow always fails on Windows.
func (c *Controller) KillWindow(winID int) error { return ErrNoDaemon }

// SetTitle always fails on Windows.
func (c *Controller) SetTitle(winID int, title string) error { return ErrNoDaemon }

// OpenWindowProcess always fails on Windows.
func (c *Controller) OpenWindowProcess(win *session.Window) (*pty.PTYProcess, error) {
	return nil, fmt.Errorf("session daemon not supported on windows")
}
