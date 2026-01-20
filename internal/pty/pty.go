package pty

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// PTYProcess represents a PTY process with its command and PTY file
type PTYProcess struct {
	Cmd *exec.Cmd
	Pty *os.File
}

// Start creates a new PTY process with the given command and arguments
func Start(cmdPath string, args []string) (*PTYProcess, error) {
	cmd := exec.Command(cmdPath, args...)
	cmd.Env = os.Environ()

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	return &PTYProcess{
		Cmd: cmd,
		Pty: ptyFile,
	}, nil
}

// Pipe connects the client's input/output to the PTY
// It copies data bidirectionally between client and PTY
func (p *PTYProcess) Pipe(clientIn io.Reader, clientOut io.Writer) error {
	// Copy from client input to PTY
	go func() {
		io.Copy(p.Pty, clientIn)
	}()

	// Copy from PTY to client output (main loop)
	_, err := io.Copy(clientOut, p.Pty)
	return err
}

// SetSize sets the size of the PTY
func (p *PTYProcess) SetSize(rows, cols uint16) error {
	return pty.Setsize(p.Pty, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// Close closes the PTY file
func (p *PTYProcess) Close() error {
	if p.Pty != nil {
		return p.Pty.Close()
	}
	return nil
}

// Wait waits for the command to finish
func (p *PTYProcess) Wait() error {
	if p.Cmd != nil {
		return p.Cmd.Wait()
	}
	return nil
}

// Kill kills the underlying process
func (p *PTYProcess) Kill() error {
	if p.Cmd != nil && p.Cmd.Process != nil {
		return p.Cmd.Process.Kill()
	}
	return nil
}

