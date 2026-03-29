package unit

import (
	"syscall"
	"testing"

	"github.com/inoki/sgreen/internal/session"
)

func TestSessionCreate(t *testing.T) {
	s, err := session.New("test", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if s == nil {
		t.Fatalf("Expected session to be created")
	}
	if s.ID != "test" {
		t.Errorf("Expected session ID to be 'test', got '%s'", s.ID)
	}
	if err := session.Delete(s.ID); err != nil {
		t.Logf("Failed to delete session: %v", err)
	}
}

func TestSessionCreateWithEmptyName(t *testing.T) {
	_, err := session.New("", "/bin/bash", []string{})
	if err == nil {
		t.Errorf("Expected error when creating session with empty name")
	}
}

func TestSessionLifecycle(t *testing.T) {
	s, err := session.New("test", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if err := session.Delete(s.ID); err != nil {
		t.Logf("Failed to delete session: %v", err)
	}

	if len(s.Windows) == 0 {
		t.Errorf("Expected at least one window in session")
	}
}

func TestWindowManagement(t *testing.T) {
	s, err := session.New("test", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if err := session.Delete(s.ID); err != nil {
		t.Logf("Failed to delete session: %v", err)
	}

	initialCount := len(s.Windows)
	if initialCount == 0 {
		t.Errorf("Expected at least one window")
	}

	win, err := s.CreateWindow("/bin/sh", []string{}, nil)
	if err != nil {
		t.Fatalf("Failed to create new window: %v", err)
	}
	if win == nil {
		t.Fatalf("Expected window to be created")
	}

	if len(s.Windows) != initialCount+1 {
		t.Errorf("Expected window count to increase by 1, got %d", len(s.Windows))
	}
}

func TestWindowSwitching(t *testing.T) {
	s, err := session.New("test", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if err := session.Delete(s.ID); err != nil {
		t.Logf("Failed to delete session: %v", err)
	}

	if _, err := s.CreateWindow("/bin/sh", []string{}, nil); err != nil {
		t.Logf("Failed to create window: %v", err)
	}

	if len(s.Windows) < 2 {
		t.Fatalf("Expected at least 2 windows for switching test")
	}

	err = s.SwitchToWindow("1")
	if err != nil {
		t.Fatalf("Failed to switch to window 1: %v", err)
	}

	if s.CurrentWindow != 1 {
		t.Errorf("Expected current window to be 1, got %d", s.CurrentWindow)
	}
}

func TestWindowSwitchingInvalidIndex(t *testing.T) {
	s, err := session.New("test", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if err := session.Delete(s.ID); err != nil {
		t.Logf("Failed to delete session: %v", err)
	}

	err = s.SwitchToWindow("999")
	if err == nil {
		t.Errorf("Expected error when switching to invalid window")
	}
}

func TestCurrentUser(t *testing.T) {
	user := session.CurrentUser()
	if user == "" {
		t.Logf("Warning: CurrentUser returned empty string")
	}
}

func TestDetectEncodingFromLocale(t *testing.T) {
	encoding := session.DetectEncodingFromLocale()
	if encoding == "" {
		t.Errorf("DetectEncodingFromLocale should return a default encoding")
	}
}

func TestIsResourceExhausted(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"ENOSPC", syscall.ENOSPC, true},
		{"EMFILE", syscall.EMFILE, true},
		{"ENFILE", syscall.ENFILE, true},
		{"other error", syscall.EINVAL, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := session.IsResourceExhausted(tt.err)
			if result != tt.expected {
				t.Errorf("IsResourceExhausted(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}
