//go:build windows
// +build windows

package ui

import (
	"errors"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/inoki/sgreen/internal/session"
)

var (
	// ErrDetach is returned when the user detaches from a session
	ErrDetach = errors.New("detached from session")
)

// Attach attaches the current terminal to a session
// Note: Windows has limited PTY support, window size changes are not handled
func Attach(in *os.File, out *os.File, errOut *os.File, sess *session.Session) error {
	// Get PTY process
	ptyProc := sess.GetPTYProcess()
	if ptyProc == nil {
		return errors.New("PTY process not available")
	}

	// Save original terminal state
	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(in.Fd()), oldState)

	// Set initial window size (if supported)
	setWindowSize(in, sess)

	// Create a reader that detects detach sequence (Ctrl+A, d)
	detachReader := newDetachReader(in)

	// Copy from PTY to output
	go func() {
		io.Copy(out, ptyProc.Pty)
	}()

	// Copy from input to PTY, with detach detection
	_, err = io.Copy(ptyProc.Pty, detachReader)
	if err == ErrDetach {
		// User detached, this is normal
		return nil
	}

	return err
}

// setWindowSize sets the PTY window size to match the terminal
func setWindowSize(termFile *os.File, sess *session.Session) error {
	width, height, err := term.GetSize(int(termFile.Fd()))
	if err != nil {
		return err
	}

	ptyProc := sess.GetPTYProcess()
	if ptyProc == nil {
		return errors.New("PTY process not available")
	}

	return ptyProc.SetSize(uint16(height), uint16(width))
}

// detachReader wraps an io.Reader to detect the detach sequence (Ctrl+A, d)
type detachReader struct {
	reader   io.Reader
	state    int    // 0: normal, 1: saw Ctrl+A
	pending  []byte // bytes to output before reading more
}

func newDetachReader(reader io.Reader) *detachReader {
	return &detachReader{
		reader:  reader,
		state:   0,
		pending: make([]byte, 0, 2),
	}
}

func (dr *detachReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// First, output any pending bytes
	if len(dr.pending) > 0 {
		copied := copy(p, dr.pending)
		dr.pending = dr.pending[copied:]
		if copied > 0 {
			return copied, nil
		}
	}

	// Read one byte at a time to detect escape sequences
	buf := make([]byte, 1)
	read, err := dr.reader.Read(buf)
	if err != nil {
		return 0, err
	}

	if read == 0 {
		return 0, nil
	}

	b := buf[0]

	switch dr.state {
	case 0:
		// Normal state
		if b == 0x01 { // Ctrl+A
			dr.state = 1
			// Don't output Ctrl+A, wait for next character
			return 0, nil
		}
		// Normal byte
		p[0] = b
		return 1, nil

	case 1:
		// Saw Ctrl+A, waiting for 'd'
		if b == 'd' {
			// Detach sequence detected
			return 0, ErrDetach
		}
		// Not 'd', output the Ctrl+A we held back, then this byte
		dr.state = 0
		if len(p) >= 2 {
			p[0] = 0x01
			p[1] = b
			return 2, nil
		}
		// Buffer too small, output Ctrl+A and buffer the next byte
		p[0] = 0x01
		dr.pending = append(dr.pending, b)
		return 1, nil
	}

	return 0, nil
}

