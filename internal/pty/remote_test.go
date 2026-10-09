package pty_test

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/inoki/sgreen/internal/pty"
)

type remoteCalls struct {
	resize      [][2]uint16
	resizeErr   error
	alive       int
	aliveResult bool
	quit        int
	quitErr     error
}

// newRemotePair wires a Remote to an in-memory pipe plus recording control
// functions, so the routing logic can be verified without a real daemon.
func newRemotePair(t *testing.T) (server net.Conn, proc *pty.PTYProcess, calls *remoteCalls) {
	t.Helper()

	server, client := net.Pipe()
	t.Cleanup(func() {
		_ = server.Close()
		_ = client.Close()
	})

	calls = &remoteCalls{}
	remote := pty.NewRemote(client, pty.RemoteControl{
		Resize: func(rows, cols uint16) error {
			calls.resize = append(calls.resize, [2]uint16{rows, cols})
			return calls.resizeErr
		},
		Alive: func() bool {
			calls.alive++
			return calls.aliveResult
		},
		Quit: func() error {
			calls.quit++
			return calls.quitErr
		},
	})
	return server, pty.NewRemoteProcess(remote, "/dev/pts/9"), calls
}

func TestRemoteProcessRouting(t *testing.T) {
	_, proc, calls := newRemotePair(t)

	if !proc.IsRemote() {
		t.Fatal("IsRemote() = false for a daemon-backed process")
	}
	if proc.DataConn() == nil {
		t.Fatal("DataConn() = nil for a remote process")
	}

	calls.resizeErr = errors.New("boom")
	if err := proc.SetSize(24, 80); err == nil || err.Error() != "boom" {
		t.Fatalf("SetSize should route through the daemon resize hook, got %v", err)
	}
	if len(calls.resize) != 1 || calls.resize[0] != [2]uint16{24, 80} {
		t.Fatalf("resize hook calls = %v", calls.resize)
	}

	calls.aliveResult = true
	if !proc.IsAlive() {
		t.Fatal("IsAlive should route through the daemon status hook")
	}
	if calls.alive != 1 {
		t.Fatalf("alive hook calls = %d, want 1", calls.alive)
	}

	calls.quitErr = errors.New("nope")
	if err := proc.Kill(); err == nil || err.Error() != "nope" {
		t.Fatalf("Kill should route through the daemon quit hook, got %v", err)
	}
}

func TestRemoteProcessWithoutHooks(t *testing.T) {
	// Zero RemoteControl: hooks are optional and must degrade gracefully.
	server, client := net.Pipe()
	t.Cleanup(func() {
		_ = server.Close()
		_ = client.Close()
	})

	remote := pty.NewRemote(client, pty.RemoteControl{})
	proc := pty.NewRemoteProcess(remote, "")

	if err := proc.SetSize(1, 1); err != nil {
		t.Fatalf("SetSize without hook should be a no-op, got %v", err)
	}
	if !proc.IsAlive() {
		t.Fatal("IsAlive without hook should assume alive")
	}
	if err := proc.Kill(); err != nil {
		t.Fatalf("Kill without hook should be a no-op, got %v", err)
	}
}

func TestRemoteDataConnRelay(t *testing.T) {
	server, proc, _ := newRemotePair(t)

	readDone := make(chan struct{})
	var got []byte
	go func() {
		defer close(readDone)
		var err error
		got, err = io.ReadAll(proc.DataConn())
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			t.Errorf("read via data conn: %v", err)
		}
	}()

	if _, err := server.Write([]byte("hello")); err != nil {
		t.Fatalf("write to server side: %v", err)
	}
	// net.Pipe is unbuffered; close to end the reader.
	_ = server.Close()

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for relayed data")
	}
	if string(got) != "hello" {
		t.Fatalf("relayed data = %q, want %q", got, "hello")
	}
}
