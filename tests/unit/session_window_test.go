package unit

import (
	"testing"

	"github.com/inoki/sgreen/internal/session"
)

func TestWindowCreation(t *testing.T) {
	s, err := session.New("test_windows", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer func() {
		if err := session.Delete(s.ID); err != nil {
			t.Logf("Failed to delete session: %v", err)
		}
	}()

	initialCount := len(s.Windows)

	win, err := s.CreateWindow("/bin/sh", []string{}, nil)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}
	if win == nil {
		t.Fatalf("Expected window to be created")
	}

	if len(s.Windows) != initialCount+1 {
		t.Errorf("Expected window count to increase by 1, got %d", len(s.Windows))
	}
}

func TestWindowDeletion(t *testing.T) {
	s, err := session.New("test_window_delete", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer func() {
		if err := session.Delete(s.ID); err != nil {
			t.Logf("Failed to delete session: %v", err)
		}
	}()

	win, err := s.CreateWindow("/bin/sh", []string{}, nil)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}

	initialCount := len(s.Windows)

	if err := win.Kill(); err != nil {
		t.Logf("Failed to kill window: %v", err)
	}

	if len(s.Windows) != initialCount {
		t.Errorf("Expected window count to remain same, got %d", len(s.Windows))
	}
}

func TestWindowTitle(t *testing.T) {
	s, err := session.New("test_window_title", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer func() {
		if err := session.Delete(s.ID); err != nil {
			t.Logf("Failed to delete session: %v", err)
		}
	}()

	win, err := s.CreateWindow("/bin/sh", []string{}, nil)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}

	if win == nil {
		t.Fatalf("Expected window to be created")
	}

	title := "Test Window Title"
	win.Title = title

	if win.Title != title {
		t.Errorf("Expected window title %s, got %s", title, win.Title)
	}
}

func TestWindowEncoding(t *testing.T) {
	s, err := session.New("test_window_encoding", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer func() {
		if err := session.Delete(s.ID); err != nil {
			t.Logf("Failed to delete session: %v", err)
		}
	}()

	encoding := "UTF-8"
	win, err := s.CreateWindow("/bin/sh", []string{}, nil)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}

	if win == nil {
		t.Fatalf("Expected window to be created")
	}

	if win.Encoding != "" {
		t.Logf("Default encoding is %s", win.Encoding)
	}

	win.Encoding = encoding

	if win.Encoding != encoding {
		t.Errorf("Expected window encoding %s, got %s", encoding, win.Encoding)
	}
}

func TestMultipleWindows(t *testing.T) {
	s, err := session.New("test_multiple_windows", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer func() {
		if err := session.Delete(s.ID); err != nil {
			t.Logf("Failed to delete session: %v", err)
		}
	}()

	windowCount := 5
	for i := 0; i < windowCount; i++ {
		_, err := s.CreateWindow("/bin/sh", []string{}, nil)
		if err != nil {
			t.Fatalf("Failed to create window %d: %v", i, err)
		}
	}

	if len(s.Windows) < windowCount {
		t.Errorf("Expected at least %d windows, got %d", windowCount, len(s.Windows))
	}
}

func TestWindowSwitchingBack(t *testing.T) {
	s, err := session.New("test_window_switching", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer func() {
		if err := session.Delete(s.ID); err != nil {
			t.Logf("Failed to delete session: %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		_, err := s.CreateWindow("/bin/sh", []string{}, nil)
		if err != nil {
			t.Fatalf("Failed to create window %d: %v", i, err)
		}
	}

	if len(s.Windows) < 4 {
		t.Fatalf("Expected at least 4 windows for switching test")
	}

	if err := s.SwitchToWindow("0"); err != nil {
		t.Fatalf("Failed to switch to window 0: %v", err)
	}

	if s.CurrentWindow != 0 {
		t.Errorf("Expected current window 0, got %d", s.CurrentWindow)
	}

	if err := s.SwitchToWindow("2"); err != nil {
		t.Fatalf("Failed to switch to window 2: %v", err)
	}

	if s.CurrentWindow != 2 {
		t.Errorf("Expected current window 2, got %d", s.CurrentWindow)
	}

	if err := s.SwitchToWindow("1"); err != nil {
		t.Fatalf("Failed to switch to window 1: %v", err)
	}

	if s.CurrentWindow != 1 {
		t.Errorf("Expected current window 1, got %d", s.CurrentWindow)
	}
}
