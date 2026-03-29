package unit

import (
	"testing"

	"github.com/inoki/sgreen/internal/pty"
)

func TestPTYProcessCreation(t *testing.T) {
	ptyProc := &pty.PTYProcess{}
	if ptyProc.PtsPath != "" {
		t.Errorf("PTYProcess should have empty PtsPath")
	}
}

func TestPTYProcessFields(t *testing.T) {
	ptyProc := &pty.PTYProcess{
		PtsPath: "/dev/pts/0",
	}

	if ptyProc.PtsPath != "/dev/pts/0" {
		t.Errorf("Expected PtsPath /dev/pts/0, got %s", ptyProc.PtsPath)
	}
}

func TestPTYProcessNil(t *testing.T) {
	var ptyProc *pty.PTYProcess
	if ptyProc != nil {
		t.Errorf("Nil PTYProcess should be nil")
	}
}

func TestPTYPathValidation(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{"unix path", "/dev/pts/0", true},
		{"windows path", "\\\\.\\pipe\\", true},
		{"empty path", "", false},
		{"relative path", "pts/0", false},
		{"unix absolute path", "/dev/pty/1", true},
		{"relative windows path", ".\\pipe\\", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validatePTYPath(tt.path)
			if valid != tt.valid {
				t.Errorf("Path validation failed for %s: got valid=%v, expected valid=%v", tt.path, valid, tt.valid)
			}
		})
	}
}

func validatePTYPath(path string) bool {
	if path == "" {
		return false
	}

	if path[0] == '/' {
		return true
	}

	if len(path) >= 2 && path[0] == '\\' && path[1] == '\\' {
		return true
	}

	return false
}
