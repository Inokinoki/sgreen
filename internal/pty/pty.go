package pty

import (
	"io"
	"os"
	"os/exec"
)

// PTYProcess represents a PTY process with its command and PTY file
type PTYProcess struct {
	Cmd     *exec.Cmd
	Pty     *os.File
	PtsPath string // Path to the PTY slave device

	// PtyRead is the read side of the master endpoint. On Unix the master
	// fd is bidirectional and this stays nil; Windows ConPTY exposes the
	// master as two one-way pipes.
	PtyRead *os.File

	// conptyResizer resizes the pseudo console on Windows (nil elsewhere).
	conptyResizer func(rows, cols uint16) error

	// remote is set when this endpoint is a connection to a session daemon
	// rather than a locally-owned master fd.
	remote *Remote
}

// NewRemoteProcess builds a PTYProcess backed by a daemon connection.
func NewRemoteProcess(remote *Remote, ptsPath string) *PTYProcess {
	return &PTYProcess{remote: remote, PtsPath: ptsPath}
}

// IsRemote reports whether this endpoint relays through a session daemon.
func (p *PTYProcess) IsRemote() bool { return p != nil && p.remote != nil }

// DataConn returns the data endpoint used for relay: the daemon socket
// connection when remote, otherwise the local PTY master.
func (p *PTYProcess) DataConn() io.ReadWriteCloser {
	if p.remote != nil {
		return p.remote
	}
	if p.PtyRead != nil {
		return &pipePair{r: p.PtyRead, w: p.Pty}
	}
	return p.Pty
}

// pipePair joins two one-way pipes into one ReadWriteCloser (Windows
// ConPTY shape).
type pipePair struct {
	r      *os.File
	w      *os.File
	closed bool
}

func (pp *pipePair) Read(p []byte) (int, error)  { return pp.r.Read(p) }
func (pp *pipePair) Write(p []byte) (int, error) { return pp.w.Write(p) }
func (pp *pipePair) Close() error {
	if pp.closed {
		return nil
	}
	pp.closed = true
	err1 := pp.r.Close()
	err2 := pp.w.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

// Start creates a new PTY process with the given command and arguments
func Start(cmdPath string, args []string) (*PTYProcess, error) {
	return StartWithEnv(cmdPath, args, nil)
}

// Pipe connects the client's input/output to the PTY
// It copies data bidirectionally between client and PTY
func (p *PTYProcess) Pipe(clientIn io.Reader, clientOut io.Writer) error {
	// Copy from client input to PTY
	go func() {
		_, _ = io.Copy(p.DataConn(), clientIn)
	}()

	// Copy from PTY to client output (main loop)
	_, err := io.Copy(clientOut, p.DataConn())
	return err
}

// SetSize sets the size of the PTY
func (p *PTYProcess) SetSize(rows, cols uint16) error {
	if p.remote != nil {
		return p.remote.Resize(rows, cols)
	}
	return p.setLocalSize(rows, cols)
}

// Close closes the PTY file
func (p *PTYProcess) Close() error {
	if p.remote != nil {
		return p.remote.Close()
	}
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
	if p.remote != nil {
		return p.remote.Quit()
	}
	if p.Cmd != nil && p.Cmd.Process != nil {
		return p.Cmd.Process.Kill()
	}
	return nil
}
