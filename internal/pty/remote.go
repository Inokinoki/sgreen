package pty

import (
	"net"
	"sync"
	"time"
)

// Remote is a live connection to a session daemon. When a PTYProcess
// carries a Remote, its data endpoint is the daemon socket (the master fd
// lives in the daemon process) and control operations are forwarded to the
// daemon. Control implementations are injected as functions to avoid a
// dependency cycle with the daemon package.
type Remote struct {
	conn net.Conn // ATTACH data connection

	resizeFn func(rows, cols uint16) error
	aliveFn  func() bool
	quitFn   func() error

	ctrlMu sync.Mutex
}

// RemoteControl carries the daemon-side implementations of the control
// operations available on a Remote endpoint.
type RemoteControl struct {
	Resize func(rows, cols uint16) error
	Alive  func() bool
	Quit   func() error
}

// NewRemote wraps an ATTACH connection to a session daemon.
func NewRemote(conn net.Conn, ctrl RemoteControl) *Remote {
	return &Remote{conn: conn, resizeFn: ctrl.Resize, aliveFn: ctrl.Alive, quitFn: ctrl.Quit}
}

// Read implements io.Reader on the data connection.
func (r *Remote) Read(p []byte) (int, error) { return r.conn.Read(p) }

// Write implements io.Writer on the data connection.
func (r *Remote) Write(p []byte) (int, error) { return r.conn.Write(p) }

// Close closes the data connection (this detaches the client).
func (r *Remote) Close() error { return r.conn.Close() }

// SetReadDeadline sets the read deadline when the underlying connection
// supports it (Unix sockets do).
func (r *Remote) SetReadDeadline(t time.Time) error {
	type deadliner interface{ SetReadDeadline(time.Time) error }
	if d, ok := r.conn.(deadliner); ok {
		return d.SetReadDeadline(t)
	}
	return nil
}

// Resize asks the daemon to apply a window size to the session PTY.
func (r *Remote) Resize(rows, cols uint16) error {
	if r.resizeFn == nil {
		return nil
	}
	r.ctrlMu.Lock()
	defer r.ctrlMu.Unlock()
	return r.resizeFn(rows, cols)
}

// Alive reports whether the session daemon still serves the session.
func (r *Remote) Alive() bool {
	if r.aliveFn == nil {
		return true
	}
	r.ctrlMu.Lock()
	defer r.ctrlMu.Unlock()
	return r.aliveFn()
}

// Quit asks the daemon to terminate the session.
func (r *Remote) Quit() error {
	if r.quitFn == nil {
		return nil
	}
	r.ctrlMu.Lock()
	defer r.ctrlMu.Unlock()
	return r.quitFn()
}
