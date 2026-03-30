package unit

import (
	"testing"

	"github.com/inoki/sgreen/internal/session"
	"github.com/inoki/sgreen/internal/ui"
)

func TestFormatMessage(t *testing.T) {
	win := &session.Window{Number: "0", Title: "Test Window", CmdPath: "/bin/bash"}

	tests := []struct {
		name   string
		format string
		result string
	}{
		{
			name:   "window number",
			format: "Window %n",
			result: "Window 0",
		},
		{
			name:   "window title",
			format: "Title: %t",
			result: "Title: Test Window",
		},
		{
			name:   "mixed",
			format: "Window %n: %t",
			result: "Window 0: Test Window",
		},
		{
			name:   "bell",
			format: "Bell %G",
			result: "Bell \a",
		},
		{
			name:   "literal percent",
			format: "100%%",
			result: "100%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ui.FormatMessage(tt.format, win)
			if result != tt.result {
				t.Errorf("FormatMessage(%q) = %q, want %q", tt.format, result, tt.result)
			}
		})
	}
}

func TestFormatMessageEmptyWindow(t *testing.T) {
	win := &session.Window{}
	result := ui.FormatMessage("Window %n", win)
	if result != "Window " {
		t.Errorf("Expected empty window number, got %q", result)
	}
}

func TestFormatMessageWithEmptyTitle(t *testing.T) {
	win := &session.Window{Number: "0", Title: "", CmdPath: "/bin/bash"}
	result := ui.FormatMessage("Cmd: %t", win)
	if result != "Cmd: /bin/bash" {
		t.Errorf("FormatMessage with empty title = %q, want %q", result, "Cmd: /bin/bash")
	}
}

func TestStatusLineCreation(t *testing.T) {
	status := ui.NewStatusLine(true, "Session: %s Window: %n")
	if status == nil {
		t.Errorf("NewStatusLine() returned nil")
	}
}

func TestActivityMonitor(t *testing.T) {
	monitor := ui.NewActivityMonitor("Activity in window %n")
	if monitor == nil {
		t.Errorf("NewActivityMonitor() returned nil")
	}

	msg := monitor.GetMessage()
	if msg != "Activity in window %n" {
		t.Errorf("GetMessage() = %q, want %q", msg, "Activity in window %n")
	}

	monitor.Enable()
	if monitor.GetMessage() == "" {
		t.Errorf("GetMessage() returned empty after Enable()")
	}
}

func TestSilenceMonitor(t *testing.T) {
	monitor := ui.NewSilenceMonitor("Silence in window %n", 30)
	if monitor == nil {
		t.Errorf("NewSilenceMonitor() returned nil")
	}

	msg := monitor.GetMessage()
	if msg != "Silence in window %n" {
		t.Errorf("GetMessage() = %q, want %q", msg, "Silence in window %n")
	}

	monitor.Enable()
	if monitor.GetMessage() == "" {
		t.Errorf("GetMessage() returned empty after Enable()")
	}
}
