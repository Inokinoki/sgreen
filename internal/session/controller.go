package session

import (
	"github.com/inoki/sgreen/internal/pty"
)

// WindowController routes window lifecycle operations to the process that
// owns the windows' PTY masters — the session daemon. When Session.Controller
// is set (attach through the daemon), the session's own mutation methods
// (CreateWindow, NextWindow, SwitchToWindow, KillCurrentWindow, ...) forward
// to it instead of acting locally, so every caller — attach loop, command
// prompt, window list — transparently operates on the daemon-owned state.
//
// The interface lives here (not in internal/daemon) to avoid an import
// cycle; internal/daemon provides the implementation.
type WindowController interface {
	// CreateWindow starts a new program in a daemon-owned PTY and returns
	// the new window's ID.
	CreateWindow(cmdPath string, args []string, term string) (int, error)
	// SwitchWindow moves the session cursor; op is one of "next", "prev",
	// "toggle", "select" (arg holds the window number for "select").
	// It returns the resulting current window ID.
	SwitchWindow(op string, arg string) (int, error)
	// KillWindow terminates the given window's program and removes it.
	KillWindow(winID int) error
	// SetTitle renames a window.
	SetTitle(winID int, title string) error
	// OpenWindowProcess returns a relay endpoint (ATTACH connection wrapped
	// in a PTYProcess) for the given window's PTY.
	OpenWindowProcess(win *Window) (*pty.PTYProcess, error)
}

// RefreshFromDisk re-reads this session's file and updates the mutable
// window state in place. Attachers use it after daemon-side mutations
// (window created/killed/switched elsewhere) to resync their view.
func (s *Session) RefreshFromDisk() error {
	fresh, err := loadFromDisk(s.ID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.Windows = fresh.Windows
	s.CurrentWindow = fresh.CurrentWindow
	s.LastWindow = fresh.LastWindow
	s.Attached = fresh.Attached
	s.DaemonPid = fresh.DaemonPid
	s.mu.Unlock()
	return nil
}

// DetectLocaleEncoding re-exports the locale-based encoding detection used
// when creating windows, for the session daemon.
func DetectLocaleEncoding() string {
	return detectEncodingFromLocale()
}

// WindowNumberString re-exports window-number formatting (0-9 then A-Z).
func WindowNumberString(id int) string {
	return windowNumberToString(id)
}
