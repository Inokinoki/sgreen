package unit

import (
	"os/exec"
	"testing"

	"github.com/inoki/sgreen/internal/pty"
)

func TestPTYStart(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		args    []string
		wantErr bool
	}{
		{
			name:    "simple echo command",
			cmd:     "/bin/echo",
			args:    []string{"hello"},
			wantErr: false,
		},
		{
			name:    "sleep command",
			cmd:     "/bin/sleep",
			args:    []string{"0"},
			wantErr: false,
		},
		{
			name:    "empty command",
			cmd:     "",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "non-existent command",
			cmd:     "/non/existent/command",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptyProc, err := pty.Start(tt.cmd, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for command %s", tt.cmd)
				}
				return
			}

			if err != nil {
				t.Fatalf("Failed to start PTY: %v", err)
			}
			if ptyProc == nil {
				t.Fatalf("PTYProcess should not be nil")
			}

			if ptyProc.Cmd == nil {
				t.Errorf("PTYProcess.Cmd should not be nil")
			}

			if ptyProc.PtsPath == "" {
				t.Logf("PTYProcess.PtsPath is empty (may be expected for some commands)")
			}

			if ptyProc.Pty != nil {
				ptyProc.Close()
			}
		})
	}
}

func TestPTYStartWithEnv(t *testing.T) {
	tests := []struct {
		name         string
		cmd          string
		args         []string
		envOverrides map[string]string
		wantErr      bool
	}{
		{
			name:         "command with env override",
			cmd:          "/bin/echo",
			args:         []string{"test"},
			envOverrides: map[string]string{"TEST_VAR": "test_value"},
			wantErr:      false,
		},
		{
			name:         "command with empty env",
			cmd:          "/bin/echo",
			args:         []string{"hello"},
			envOverrides: nil,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptyProc, err := pty.StartWithEnv(tt.cmd, tt.args, tt.envOverrides)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for command %s", tt.cmd)
				}
				return
			}

			if err != nil {
				t.Fatalf("Failed to start PTY: %v", err)
			}
			if ptyProc == nil {
				t.Fatalf("PTYProcess should not be nil")
			}

			if ptyProc.Cmd == nil {
				t.Errorf("PTYProcess.Cmd should not be nil")
			}

			if tt.envOverrides != nil {
				for key := range tt.envOverrides {
					found := false
					for _, env := range ptyProc.Cmd.Env {
						if len(env) > len(key) && env[:len(key)] == key {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Environment variable %s not found", key)
					}
				}
			}

			if ptyProc.Pty != nil {
				ptyProc.Close()
			}
		})
	}
}

func TestPTYCommandCreation(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		args []string
	}{
		{"simple echo", "/bin/echo", []string{"hello"}},
		{"shell command", "/bin/sh", []string{"-c", "echo test"}},
		{"multi-args", "/bin/echo", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(tt.cmd, tt.args...)
			if cmd == nil {
				t.Errorf("Failed to create command: %s %v", tt.cmd, tt.args)
				return
			}
			if cmd.Path != tt.cmd {
				t.Errorf("Expected path %s, got %s", tt.cmd, cmd.Path)
			}
		})
	}
}

func TestPTYProcessStructureBasic(t *testing.T) {
	ptyProc := &pty.PTYProcess{
		Cmd:     exec.Command("/bin/echo", "test"),
		PtsPath: "/dev/pts/test",
	}

	if ptyProc.Cmd == nil {
		t.Errorf("PTYProcess.Cmd should not be nil")
	}
	if ptyProc.PtsPath != "/dev/pts/test" {
		t.Errorf("Expected PtsPath /dev/pts/test, got %s", ptyProc.PtsPath)
	}
}
