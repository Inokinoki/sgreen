package unit

import (
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
	defer session.Delete(s.ID)
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
	defer session.Delete(s.ID)

	if len(s.Windows) == 0 {
		t.Errorf("Expected at least one window in session")
	}
}

func TestWindowManagement(t *testing.T) {
	s, err := session.New("test", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer session.Delete(s.ID)

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
	defer session.Delete(s.ID)

	s.CreateWindow("/bin/sh", []string{}, nil)

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
	defer session.Delete(s.ID)

	err = s.SwitchToWindow("999")
	if err == nil {
		t.Errorf("Expected error when switching to invalid window")
	}
}
