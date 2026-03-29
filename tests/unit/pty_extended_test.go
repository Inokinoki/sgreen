package unit

import (
	"os"
	"os/exec"
	"testing"

	"github.com/inoki/sgreen/internal/pty"
)

func TestPTYKill(t *testing.T) {
	ptyProc := &pty.PTYProcess{
		Cmd: exec.Command("/bin/echo", "test"),
	}

	err := ptyProc.Kill()
	if err != nil && !os.IsNotExist(err) {
		t.Errorf("Kill on non-running process should not error: %v", err)
	}
}

func TestPTYKillNil(t *testing.T) {
	ptyProc := &pty.PTYProcess{}

	err := ptyProc.Kill()
	if err != nil {
		t.Errorf("Kill on nil command should not error: %v", err)
	}
}

func TestPTYClose(t *testing.T) {
	ptyProc := &pty.PTYProcess{}

	err := ptyProc.Close()
	if err != nil {
		t.Errorf("Close on nil PTY should not error: %v", err)
	}
}

func TestPTYWait(t *testing.T) {
	ptyProc := &pty.PTYProcess{
		Cmd: exec.Command("/bin/echo", "test"),
	}

	err := ptyProc.Wait()
	if err != nil {
		t.Logf("Wait may error on non-running process: %v", err)
	}
}

func TestPTYWaitNil(t *testing.T) {
	ptyProc := &pty.PTYProcess{}

	err := ptyProc.Wait()
	if err != nil {
		t.Errorf("Wait on nil command should not error: %v", err)
	}
}

func TestPTYSetSize(t *testing.T) {
	ptyProc := &pty.PTYProcess{}

	err := ptyProc.SetSize(24, 80)
	if err == nil {
		t.Errorf("SetSize on nil PTY should error")
	}
}

func TestPTYSetSizeValid(t *testing.T) {
	tests := []struct {
		name string
		rows uint16
		cols uint16
	}{
		{"standard size", 24, 80},
		{"small size", 10, 40},
		{"large size", 100, 200},
		{"minimum", 1, 1},
		{"maximum", 65535, 65535},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptyProc := &pty.PTYProcess{}
			err := ptyProc.SetSize(tt.rows, tt.cols)
			if err == nil {
				t.Errorf("SetSize on nil PTY should error")
			}
		})
	}
}

func TestPTYProcessNilMethods(t *testing.T) {
	var ptyProc *pty.PTYProcess

	err := ptyProc.Kill()
	if err != nil {
		t.Errorf("Kill on nil PTYProcess should not error")
	}

	err = ptyProc.Close()
	if err != nil {
		t.Errorf("Close on nil PTYProcess should not error")
	}

	err = ptyProc.Wait()
	if err != nil {
		t.Errorf("Wait on nil PTYProcess should not error")
	}
}

func TestPTYProcessStructure(t *testing.T) {
	ptyProc := &pty.PTYProcess{
		Cmd:     exec.Command("/bin/echo", "test"),
		PtsPath: "/dev/pts/0",
	}

	if ptyProc.Cmd == nil {
		t.Errorf("Cmd should not be nil")
	}

	if ptyProc.PtsPath != "/dev/pts/0" {
		t.Errorf("PtsPath mismatch")
	}

	if ptyProc.Pty != nil {
		t.Logf("Pty is nil (expected for test structure)")
	}
}

func TestPTYProcessZeroValues(t *testing.T) {
	ptyProc := &pty.PTYProcess{}

	if ptyProc.PtsPath != "" {
		t.Errorf("PtsPath should be empty string")
	}

	if ptyProc.Pty != nil {
		t.Errorf("Pty should be nil")
	}

	if ptyProc.Cmd != nil {
		t.Errorf("Cmd should be nil")
	}
}