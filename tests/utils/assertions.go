package utils

import (
	"testing"

	"github.com/inoki/sgreen/internal/pty"
	"github.com/inoki/sgreen/internal/session"
)

func AssertPTYRunning(t *testing.T, ptyProc *pty.PTYProcess) {
	t.Helper()
	if ptyProc == nil || ptyProc.Cmd == nil || ptyProc.Cmd.Process == nil {
		t.Errorf("Expected PTY process to be running")
	}
}

func AssertSessionWindowCount(t *testing.T, s *session.Session, expected int) {
	t.Helper()
	if len(s.Windows) != expected {
		t.Errorf("Expected %d windows, got %d", expected, len(s.Windows))
	}
}

func AssertSessionHasPTY(t *testing.T, s *session.Session) {
	t.Helper()
	if s.GetPTYProcess() == nil {
		t.Errorf("Expected session to have PTY process")
	}
}

func AssertCurrentWindow(t *testing.T, s *session.Session, expected int) {
	t.Helper()
	if s.CurrentWindow != expected {
		t.Errorf("Expected current window to be %d, got %d", expected, s.CurrentWindow)
	}
}
