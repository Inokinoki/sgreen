package unit

import (
	"testing"

	"github.com/inoki/sgreen/internal/ui"
)

func TestStatusLine(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []interface{}
		result string
	}{
		{
			name:   "simple status",
			format: "Running: %s",
			args:   []interface{}{"test"},
			result: "Running: test",
		},
		{
			name:   "status with multiple args",
			format: "Session: %s (Window: %d)",
			args:   []interface{}{"mysession", 1},
			result: "Session: mysession (Window: 1)",
		},
		{
			name:   "empty status",
			format: "",
			args:   []interface{}{},
			result: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ui.FormatStatus(tt.format, tt.args...)
			if result != tt.result {
				t.Errorf("FormatStatus() = %q, want %q", result, tt.result)
			}
		})
	}
}

func TestStatusUpdate(t *testing.T) {
	status := &ui.StatusLine{}
	if status == nil {
		t.Errorf("StatusLine should not be nil")
	}

	status.SetMessage("test message")
	if status.Message != "test message" {
		t.Errorf("Expected message 'test message', got %s", status.Message)
	}

	status.SetSession("test_session")
	if status.Session != "test_session" {
		t.Errorf("Expected session 'test_session', got %s", status.Session)
	}

	status.SetWindow(1)
	if status.Window != 1 {
		t.Errorf("Expected window 1, got %d", status.Window)
	}
}

func TestStatusComponents(t *testing.T) {
	status := &ui.StatusLine{
		Session: "session1",
		Window:  2,
		Message: "status message",
	}

	if status.Session != "session1" {
		t.Errorf("Expected session 'session1', got %s", status.Session)
	}
	if status.Window != 2 {
		t.Errorf("Expected window 2, got %d", status.Window)
	}
	if status.Message != "status message" {
		t.Errorf("Expected message 'status message', got %s", status.Message)
	}
}
