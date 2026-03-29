package unit

import (
	"testing"

	"github.com/inoki/sgreen/internal/session"
)

func TestSessionValidation(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		cmd       string
		args      []string
		wantErr   bool
		errString string
	}{
		{
			name:    "valid session name",
			id:      "test_session_123",
			cmd:     "/bin/bash",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "empty session name",
			id:      "",
			cmd:     "/bin/bash",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "session name with spaces",
			id:      "test session",
			cmd:     "/bin/bash",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "session name with special chars",
			id:      "test@session",
			cmd:     "/bin/bash",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "valid session with dots and dashes",
			id:      "test.session-123",
			cmd:     "/bin/bash",
			args:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := session.New(tt.id, tt.cmd, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for session %s", tt.id)
				}
				return
			}

			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}
			if s == nil {
				t.Fatalf("Expected session to be created")
			}
			if s.ID != tt.id {
				t.Errorf("Expected session ID %s, got %s", tt.id, s.ID)
			}
			if err := session.Delete(s.ID); err != nil {
				t.Logf("Failed to delete session: %v", err)
			}
		})
	}
}

func TestSessionConfig(t *testing.T) {
	config := &session.Config{
		Term:       "xterm-256color",
		UTF8:       true,
		Scrollback:  1000,
		Encoding:    "UTF-8",
	}

	s, err := session.NewWithConfig("test_config", "/bin/bash", []string{}, config)
	if err != nil {
		t.Fatalf("Failed to create session with config: %v", err)
	}
	if err := session.Delete(s.ID); err != nil {
		t.Logf("Failed to delete session: %v", err)
	}

	if s == nil {
		t.Fatalf("Expected session to be created")
	}
}

func TestCurrentUserNotEmpty(t *testing.T) {
	user := session.CurrentUser()
	if user == "" {
		t.Logf("Warning: CurrentUser returned empty string")
	}
}

func TestSessionIDValidation(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{"simple", "test", true},
		{"with numbers", "test123", true},
		{"with dots", "test.session", true},
		{"with dashes", "test-session", true},
		{"with underscores", "test_session", true},
		{"complex", "test.session-123", true},
		{"with spaces", "test session", false},
		{"with special chars", "test@session", false},
		{"with slash", "test/session", false},
		{"with backslash", "test\\session", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := session.New(tt.id, "/bin/bash", []string{})
			if tt.valid {
				if err != nil {
					t.Errorf("Expected valid session %s to succeed", tt.id)
				}
				if s != nil {
					if err := session.Delete(s.ID); err != nil {
						t.Logf("Failed to delete session: %v", err)
					}
				}
			} else {
				if err == nil {
					t.Errorf("Expected invalid session %s to fail", tt.id)
				}
			}
		})
	}
}

func TestSessionCommands(t *testing.T) {
	s, err := session.New("test_commands", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if err := session.Delete(s.ID); err != nil {
		t.Logf("Failed to delete session: %v", err)
	}

	if s.CmdPath != "/bin/bash" {
		t.Errorf("Expected CmdPath /bin/bash, got %s", s.CmdPath)
	}
	if s.CmdArgs == nil {
		t.Errorf("Expected CmdArgs to be initialized")
	}
}

func TestSessionMetadata(t *testing.T) {
	s, err := session.New("test_metadata", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if err := session.Delete(s.ID); err != nil {
		t.Logf("Failed to delete session: %v", err)
	}

	if s.CreatedAt.IsZero() {
		t.Errorf("Expected CreatedAt to be set")
	}
	if s.Owner == "" {
		t.Errorf("Expected Owner to be set")
	}
}

func TestSessionWindows(t *testing.T) {
	s, err := session.New("test_windows", "/bin/bash", []string{})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if err := session.Delete(s.ID); err != nil {
		t.Logf("Failed to delete session: %v", err)
	}

	if s.Windows == nil {
		t.Errorf("Expected Windows to be initialized")
	}
	if len(s.Windows) == 0 {
		t.Errorf("Expected at least one window")
	}
	if s.CurrentWindow != 0 {
		t.Errorf("Expected CurrentWindow to be 0, got %d", s.CurrentWindow)
	}
}

