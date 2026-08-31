//go:build !windows
// +build !windows

package pty_test

import (
	"testing"

	"github.com/inoki/sgreen/internal/pty"
)

// TestPTYStart covers starting real processes on a PTY (migrated from
// tests/unit).
func TestPTYStart(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		args    []string
		wantErr bool
	}{
		{name: "simple echo command", cmd: "/bin/echo", args: []string{"hello"}},
		{name: "sleep command", cmd: "/bin/sleep", args: []string{"0"}},
		{name: "empty command", cmd: "", args: []string{}, wantErr: true},
		{name: "non-existent command", cmd: "/nonexistent/binary", args: []string{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptyProc, err := pty.Start(tt.cmd, tt.args)
			if tt.wantErr {
				if err == nil {
					_ = ptyProc.Kill()
					t.Fatalf("expected error for command %q", tt.cmd)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ptyProc.Cmd == nil || ptyProc.Cmd.Process == nil {
				_ = ptyProc.Kill()
				t.Fatalf("expected started process for %q", tt.cmd)
			}
			if ptyProc.Pty == nil {
				_ = ptyProc.Kill()
				t.Fatalf("expected non-nil PTY master for %q", tt.cmd)
			}
			_ = ptyProc.Kill()
		})
	}
}

// TestPTYStartWithEnv verifies environment overrides reach the child (the
// TERM default is asserted here; migrated from tests/unit).
func TestPTYStartWithEnv(t *testing.T) {
	tests := []struct {
		name         string
		cmd          string
		args         []string
		envOverrides map[string]string
	}{
		{name: "command with env override", cmd: "/bin/echo", args: []string{"test"}, envOverrides: map[string]string{"TEST_VAR": "test_value"}},
		{name: "command with empty env", cmd: "/bin/echo", args: []string{"hello"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptyProc, err := pty.StartWithEnv(tt.cmd, tt.args, tt.envOverrides)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer func() { _ = ptyProc.Kill() }()

			for key, want := range tt.envOverrides {
				got := ptyProc.Cmd.Environ()
				found := false
				prefix := key + "="
				for _, kv := range got {
					if len(kv) > len(prefix) && kv[:len(prefix)] == prefix {
						found = kv == prefix+want
						break
					}
				}
				if !found {
					t.Errorf("environment override %s=%s not applied to child", key, want)
				}
			}
		})
	}
}
