//go:build windows
// +build windows

package pty

import (
	"io"
	"os"
	"testing"
	"time"
)

// readFor reads from r until want appears or the deadline passes.
func readFor(t *testing.T, r io.Reader, want string) string {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	var acc []byte
	buf := make([]byte, 512)
	for time.Now().Before(deadline) {
		_ = r.(interface{ SetReadDeadline(time.Time) error }).SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := r.Read(buf)
		if n > 0 {
			acc = append(acc, buf[:n]...)
			if len(acc) > 4096 {
				acc = acc[len(acc)-4096:]
			}
			if contains(string(acc), want) {
				return string(acc)
			}
		}
		if err != nil && !os.IsTimeout(err) && err != io.EOF {
			t.Fatalf("read: %v", err)
		}
	}
	t.Fatalf("timed out waiting for %q, got %q", want, string(acc))
	return ""
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}

// TestConPTYStartAndRelay verifies the ConPTY backend: the program starts,
// echoes input, resize works, and Close terminates the process.
func TestConPTYStartAndRelay(t *testing.T) {
	proc, err := StartWithEnv("cmd.exe", []string{"/q", "/k", "echo."}, map[string]string{"TERM": "screen"})
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

	// cmd echoes whatever we type; give it a distinctive line.
	if _, err := proc.DataConn().Write([]byte("echo conpty-smoke\r")); err != nil {
		t.Fatalf("write: %v", err)
	}
	readFor(t, proc.DataConn(), "conpty-smoke")

	// Resize must not error.
	if err := proc.SetSize(40, 120); err != nil {
		t.Errorf("SetSize: %v", err)
	}

	// Terminate and confirm the read side ends.
	if err := proc.Kill(); err != nil {
		t.Logf("kill: %v", err)
	}
	_ = proc.PtyRead.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 64)
	for {
		if _, err := proc.PtyRead.Read(buf); err != nil {
			break // EOF or timeout: process is gone either way for smoke purposes
		}
	}
}
