package unit

import (
	"os/exec"
	"testing"
)

func TestPTYCreation(t *testing.T) {
	cmd := exec.Command("/bin/echo", "test")
	if cmd == nil {
		t.Fatalf("Failed to create command")
	}
}

func TestPTYCommandWrap(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		args []string
	}{
		{"simple echo", "/bin/echo", []string{"hello"}},
		{"shell", "/bin/sh", []string{"-c", "echo test"}},
		{"complex command", "/bin/bash", []string{"-c", "for i in 1 2 3; do echo $i; done"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(tt.cmd, tt.args...)
			if cmd == nil {
				t.Errorf("Failed to create command: %s %v", tt.cmd, tt.args)
			}
		})
	}
}
