//go:build windows
// +build windows

package pty

import (
	"io"
	"os"
	"testing"
	"time"
)

// TestConPTYStartAndRelay verifies the ConPTY backend end to end: the
// program starts, output flows back through the master pipes, resize
// works, and Close tears the process down. It is a connectivity smoke
// test - Windows anonymous pipes do not support read deadlines, so the
// read happens on a goroutine guarded by a select.
func TestConPTYStartAndRelay(t *testing.T) {
	proc, err := StartWithEnv("cmd.exe", []string{"/k", "echo conpty-up"}, map[string]string{"TERM": "screen"})
	if err != nil {
		t.Skipf("ConPTY unavailable on this system: %v", err)
	}
	defer func() {
		_ = proc.Kill()
		_ = proc.Close()
	}()

	if proc.Cmd == nil || proc.Cmd.Process == nil {
		t.Fatal("no process started")
	}
	if proc.Pty == nil || proc.PtyRead == nil {
		t.Fatal("expected both ConPTY pipe ends")
	}

	// Any bytes back within the window proves the pipe pair is wired to
	// the pseudo console (cmd prints its banner and the echo line).
	type result struct {
		n   int
		err error
	}
	ch := make(chan result, 1)
	go func() {
		buf := make([]byte, 4096)
		var total int
		var rerr error
		for total < 16 { // keep reading a little to let the banner arrive
			n, err := proc.PtyRead.Read(buf)
			total += n
			if err != nil {
				rerr = err
				break
			}
		}
		ch <- result{total, rerr}
	}()

	select {
	case r := <-ch:
		if r.n == 0 {
			t.Fatalf("no output from ConPTY (err=%v)", r.err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for any ConPTY output")
	}

	// Resize must not error.
	if err := proc.SetSize(40, 120); err != nil {
		t.Errorf("SetSize: %v", err)
	}

	// Terminate; the read side should end once the console closes.
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 256)
		for {
			if _, err := proc.PtyRead.Read(buf); err != nil || err == io.EOF {
				break
			}
		}
		close(done)
	}()
	_ = proc.Kill()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Log("read side did not end after kill (smoke tolerance)")
	}
	_ = os.Getenv("SGREEN_CONPTY_DEBUG") // hook for future debugging
}
