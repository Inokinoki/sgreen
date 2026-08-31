//go:build !windows
// +build !windows

package daemon

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inoki/sgreen/internal/pty"
	"github.com/inoki/sgreen/internal/session"
)

// daemonFixture runs a real in-process daemon over a real PTY (cat) and
// cleans up after itself.
type daemonFixture struct {
	t         *testing.T
	sockPath  string
	sessFile  string
	sessID    string
	master    *os.File
	ctrl      *Controller
	exitCalls []int
	mu        sync.Mutex
}

func newDaemonFixture(t *testing.T) *daemonFixture {
	t.Helper()

	proc, err := pty.StartWithEnv("/bin/cat", nil, map[string]string{"TERM": "screen"})
	if err != nil {
		t.Skipf("cannot start PTY on this platform: %v", err)
	}

	home := t.TempDir()
	sessDir := filepath.Join(home, ".sgreen", "sessions")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	sessID := "proto-test"
	sessFile := filepath.Join(sessDir, sessID+".json")
	sockPath := filepath.Join(sessDir, sessID+".sock")
	body := map[string]any{
		"id": sessID, "cmd_path": "cat", "pid": proc.Cmd.Process.Pid,
		"created_at": "2026-01-01T00:00:00Z",
		"windows": []map[string]any{{
			"id": 0, "number": "0", "cmd_path": "cat",
			"pid": proc.Cmd.Process.Pid, "created_at": "2026-01-01T00:00:00Z",
			"scrollback_size": 100,
		}},
		"current_window": 0,
	}
	raw, _ := json.Marshal(body)
	if err := os.WriteFile(sessFile, raw, 0644); err != nil {
		t.Fatal(err)
	}

	f := &daemonFixture{
		t: t, sockPath: sockPath, sessFile: sessFile, sessID: sessID,
		master: proc.Pty,
	}
	exitFunc = func(code int) {
		f.mu.Lock()
		f.exitCalls = append(f.exitCalls, code)
		f.mu.Unlock()
	}
	t.Cleanup(func() {
		_ = proc.Kill()
		_ = proc.Pty.Close()
		// Wait for the daemon's async teardown (its endSession path calls
		// exitFunc) before restoring the real os.Exit, so a late teardown
		// cannot kill the test binary.
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && !f.exited() {
			time.Sleep(20 * time.Millisecond)
		}
		exitFunc = os.Exit
		_ = os.Remove(sockPath)
		_ = os.Remove(sessFile)
	})

	go Run(proc.Pty, sessFile, sessID, sockPath, proc.Cmd.Process.Pid, nil)

	// Wait for the daemon to listen.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(sockPath); err == nil {
			f.ctrl = NewController(sockPath)
			return f
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("daemon socket did not appear")
	return nil
}

func (f *daemonFixture) readSessionFile() *session.Session {
	f.t.Helper()
	raw, err := os.ReadFile(f.sessFile)
	if err != nil {
		f.t.Fatalf("read session file: %v", err)
	}
	var s session.Session
	if err := json.Unmarshal(raw, &s); err != nil {
		f.t.Fatalf("parse session file: %v", err)
	}
	return &s
}

func (f *daemonFixture) exited() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.exitCalls) > 0
}

// readUntil reads from r until want appears or the deadline passes.
func readUntil(t *testing.T, r io.Reader, want string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var acc strings.Builder
	buf := make([]byte, 512)
	for time.Now().Before(deadline) {
		_ = r.(interface{ SetReadDeadline(time.Time) error }).SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := r.Read(buf)
		if n > 0 {
			acc.Write(buf[:n])
			if strings.Contains(acc.String(), want) {
				return acc.String()
			}
		}
		if err != nil && !os.IsTimeout(err) {
			t.Fatalf("read: %v", err)
		}
	}
	t.Fatalf("timed out waiting for %q, got %q", want, acc.String())
	return ""
}

func TestDaemonWindowLifecycle(t *testing.T) {
	f := newDaemonFixture(t)

	// Initial state: detached, one window, current 0.
	st, err := QueryStatus(f.sockPath)
	if err != nil {
		t.Fatalf("QueryStatus: %v", err)
	}
	if st.Attached || st.Current != 0 {
		t.Fatalf("initial status = %+v", st)
	}

	// Attach to window 0 and verify bidirectional relay through the daemon.
	sess := &session.Session{ID: f.sessID}
	if err := sess.RefreshFromDisk(); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	proc0, err := f.ctrl.OpenWindowProcess(sess.GetCurrentWindow())
	if err != nil {
		t.Fatalf("OpenWindowProcess: %v", err)
	}
	defer func() { _ = proc0.Close() }()

	if _, err := proc0.DataConn().Write([]byte("relay-check\n")); err != nil {
		t.Fatalf("write via data conn: %v", err)
	}
	readUntil(t, proc0.DataConn(), "relay-check")

	st, _ = QueryStatus(f.sockPath)
	if !st.Attached {
		t.Fatal("session should report attached after OpenWindowProcess")
	}

	// Create a second window inside the daemon.
	id1, err := f.ctrl.CreateWindow("/bin/cat", nil, "")
	if err != nil {
		t.Fatalf("CreateWindow: %v", err)
	}
	if id1 != 1 {
		t.Fatalf("new window id = %d, want 1", id1)
	}
	if s := f.readSessionFile(); len(s.Windows) != 2 || s.CurrentWindow != 1 {
		t.Fatalf("session file after create: windows=%d current=%d", len(s.Windows), s.CurrentWindow)
	}

	// The new window relays too.
	if err := sess.RefreshFromDisk(); err != nil {
		t.Fatal(err)
	}
	proc1, err := f.ctrl.OpenWindowProcess(sess.GetCurrentWindow())
	if err != nil {
		t.Fatalf("OpenWindowProcess(win1): %v", err)
	}
	defer func() { _ = proc1.Close() }()
	if _, err := proc1.DataConn().Write([]byte("win1-check\n")); err != nil {
		t.Fatal(err)
	}
	readUntil(t, proc1.DataConn(), "win1-check")

	// Switching moves the persisted cursor.
	for _, tc := range []struct {
		op, arg string
		want    int
	}{
		{"prev", "", 0}, {"next", "", 1}, {"select", "0", 0}, {"toggle", "", 1},
	} {
		got, err := f.ctrl.SwitchWindow(tc.op, tc.arg)
		if err != nil || got != tc.want {
			t.Fatalf("SwitchWindow(%q,%q) = %d, %v; want %d", tc.op, tc.arg, got, err, tc.want)
		}
	}
	if s := f.readSessionFile(); s.CurrentWindow != 1 {
		t.Fatalf("current after toggle = %d, want 1", s.CurrentWindow)
	}

	// Retitle persists.
	if err := f.ctrl.SetTitle(1, "renamed"); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}
	if s := f.readSessionFile(); s.Windows[1].Title != "renamed" {
		t.Fatalf("title = %q", s.Windows[1].Title)
	}

	// Killing a non-last window removes and renumbers it.
	if err := f.ctrl.KillWindow(1); err != nil {
		t.Fatalf("KillWindow(1): %v", err)
	}
	if s := f.readSessionFile(); len(s.Windows) != 1 {
		t.Fatalf("windows after kill = %d, want 1", len(s.Windows))
	}

	// Detach closes client connections.
	if err := SendDetach(f.sockPath, false); err != nil {
		t.Fatalf("SendDetach: %v", err)
	}
	st, _ = QueryStatus(f.sockPath)
	if st.Attached {
		t.Fatal("session should be detached after SendDetach")
	}

	if f.exited() {
		t.Fatal("daemon should still be running at the end of the fixture")
	}
}

func TestDaemonShrinksWhenWindowProgramExits(t *testing.T) {
	f := newDaemonFixture(t)

	// A short-lived window dies on its own; only the initial cat remains.
	if _, err := f.ctrl.CreateWindow("/bin/sh", []string{"-c", "exit 0"}, ""); err != nil {
		t.Fatalf("CreateWindow: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s := f.readSessionFile(); len(s.Windows) == 1 {
			if s.Windows[0].CmdPath == "cat" {
				return // shrank to the initial window
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("session did not shrink after window exit: %+v", f.readSessionFile().Windows)
}

func TestDaemonKillLastWindowEndsSession(t *testing.T) {
	f := newDaemonFixture(t)

	// GNU semantics: killing the last remaining window terminates the
	// session (daemon removes its files and exits).
	if err := f.ctrl.KillWindow(0); err != nil {
		t.Fatalf("KillWindow(0): %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if f.exited() {
			if _, err := os.Stat(f.sessFile); !os.IsNotExist(err) {
				t.Errorf("session file should be gone after last window kill, stat err = %v", err)
			}
			if _, err := os.Stat(f.sockPath); !os.IsNotExist(err) {
				t.Errorf("socket should be gone after last window kill, stat err = %v", err)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("daemon did not end the session after the last window was killed")
}
