package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inoki/sgreen/internal/session"
)

// TestMain isolates HOME before any session API runs, so the lazy sessions
// directory (and every file a test writes) lands in scratch space instead
// of the developer's real ~/.sgreen.
func TestMain(m *testing.M) {
	tmpHome, err := os.MkdirTemp("", "sgreen-session-test-")
	if err != nil {
		os.Stderr.WriteString("fatal: " + err.Error())
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(tmpHome) }()
	_ = os.Setenv("HOME", tmpHome)
	// os.UserHomeDir reads USERPROFILE (not HOME) on Windows.
	_ = os.Setenv("USERPROFILE", tmpHome)
	os.Exit(m.Run())
}

// newSession starts a real session with a deterministic command. The
// command must exist on the platform; tests that need it skip on Windows.
func newSession(t *testing.T, id string) *session.Session {
	t.Helper()
	if os.PathSeparator == '\\' {
		t.Skip("process-spawning session test not supported on Windows")
	}

	s, err := session.New(id, "/bin/sleep", []string{"30"})
	if err != nil {
		t.Fatalf("New(%q): %v", id, err)
	}
	t.Cleanup(func() {
		if err := session.Delete(s.ID); err != nil {
			t.Errorf("cleanup Delete(%q): %v", s.ID, err)
		}
	})
	return s
}

func TestSessionCreateAndDelete(t *testing.T) {
	s := newSession(t, "create-delete")

	if s.ID != "create-delete" {
		t.Errorf("ID = %q, want %q", s.ID, "create-delete")
	}
	if s.Pid <= 0 {
		t.Errorf("Pid = %d, want > 0", s.Pid)
	}
	if len(s.Windows) != 1 {
		t.Fatalf("initial windows = %d, want 1", len(s.Windows))
	}
	if s.Windows[0].Pid != s.Pid {
		t.Errorf("window pid = %d, want session pid %d", s.Windows[0].Pid, s.Pid)
	}

	// The session must be visible on disk while alive.
	if _, err := os.Stat(session.FilePath(s.ID)); err != nil {
		t.Fatalf("session file missing after create: %v", err)
	}

	if err := session.Delete(s.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := session.Load(s.ID); err == nil {
		t.Error("Load should fail after Delete")
	}
	if _, err := os.Stat(session.FilePath(s.ID)); !os.IsNotExist(err) {
		t.Errorf("session file should be gone after Delete, stat err = %v", err)
	}
}

func TestSessionNameValidation(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"", false},
		{"test", true},
		{"test123", true},
		{"test.session", true},
		{"test-session", true},
		{"test_session", true},
		{"test session", false},
		{"test@session", false},
		{"test/session", false},
		{"test\\session", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			s, err := session.New(tt.id, "/bin/sleep", []string{"30"})
			if tt.valid {
				if err != nil {
					t.Fatalf("valid name %q rejected: %v", tt.id, err)
				}
				if err := session.Delete(s.ID); err != nil {
					t.Errorf("Delete(%q): %v", tt.id, err)
				}
				return
			}
			if err == nil {
				_ = session.Delete(s.ID)
				t.Fatalf("invalid name %q accepted", tt.id)
			}
		})
	}
}

func TestSessionMetadata(t *testing.T) {
	s := newSession(t, "metadata")

	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}
	if time.Since(s.CreatedAt) > time.Minute {
		t.Errorf("CreatedAt suspiciously old: %v", s.CreatedAt)
	}
	if s.Owner == "" {
		t.Error("Owner not set")
	}
	if s.CmdPath != "/bin/sleep" {
		t.Errorf("CmdPath = %q, want /bin/sleep", s.CmdPath)
	}
	if s.CurrentWindow != 0 {
		t.Errorf("CurrentWindow = %d, want 0", s.CurrentWindow)
	}
}

func TestSessionSaveLoadRoundtrip(t *testing.T) {
	s := newSession(t, "save-load")

	// Daemon-era runtime fields must survive persistence.
	s.Attached = true
	s.DaemonPid = os.Getpid()
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := session.LoadFromFile(session.FilePath(s.ID), s.ID)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if loaded.ID != s.ID || loaded.Pid != s.Pid || loaded.PtsPath != s.PtsPath {
		t.Errorf("roundtrip identity mismatch: %+v", loaded)
	}
	if !loaded.Attached || loaded.DaemonPid != os.Getpid() {
		t.Errorf("runtime fields lost: attached=%v daemon_pid=%d", loaded.Attached, loaded.DaemonPid)
	}
	if len(loaded.Windows) != 1 || loaded.Windows[0].CmdPath != "/bin/sleep" {
		t.Errorf("windows lost in roundtrip: %+v", loaded.Windows)
	}
}

func TestDiskOnlySessionLifecycle(t *testing.T) {
	// Simulate a session left behind by another process: only the JSON
	// file exists, nothing in this process's registry.
	id := "disk-only"
	body := `{"id":"` + id + `","cmd_path":"sleep","cmd_args":["1"],"pid":1,` +
		`"created_at":"2026-01-01T00:00:00Z","attached":true,"daemon_pid":1,` +
		`"windows":null,"current_window":0}`
	if err := os.MkdirAll(filepath.Dir(session.FilePath(id)), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(session.FilePath(id), []byte(body), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	sessions := session.List()
	found := false
	for _, s := range sessions {
		if s.ID == id {
			found = true
			if !s.Attached && s.DaemonPid == 1 {
				t.Errorf("runtime fields not parsed: attached=%v daemon_pid=%d", s.Attached, s.DaemonPid)
			}
		}
	}
	if !found {
		t.Fatal("List did not include the disk-only session")
	}

	// Delete must work without an in-memory entry (used by -wipe).
	if err := session.Delete(id); err != nil {
		t.Fatalf("Delete on disk-only session: %v", err)
	}
	if _, err := os.Stat(session.FilePath(id)); !os.IsNotExist(err) {
		t.Errorf("file should be gone, stat err = %v", err)
	}
}

func TestSocketAndFilePaths(t *testing.T) {
	id := "paths"
	wantJSON := filepath.Join(os.Getenv("HOME"), ".sgreen", "sessions", id+".json")
	wantSock := filepath.Join(os.Getenv("HOME"), ".sgreen", "sessions", id+".sock")
	if got := session.FilePath(id); got != wantJSON {
		t.Errorf("FilePath = %q, want %q", got, wantJSON)
	}
	if got := session.SocketPath(id); got != wantSock {
		t.Errorf("SocketPath = %q, want %q", got, wantSock)
	}
}

func TestSessionRename(t *testing.T) {
	s := newSession(t, "rename-old")

	if err := s.Rename("rename-new"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if s.ID != "rename-new" {
		t.Errorf("ID = %q after rename", s.ID)
	}
	if _, err := os.Stat(session.FilePath("rename-new")); err != nil {
		t.Errorf("renamed file missing: %v", err)
	}
	if _, err := os.Stat(session.FilePath("rename-old")); !os.IsNotExist(err) {
		t.Error("old file still present after rename")
	}
	if _, err := session.Load("rename-old"); err == nil {
		t.Error("Load under old name should fail after rename")
	}
}

func TestWindowManagement(t *testing.T) {
	s := newSession(t, "windows")

	initial := len(s.Windows)
	win, err := s.CreateWindow("/bin/sleep", []string{"30"}, nil)
	if err != nil {
		t.Fatalf("CreateWindow: %v", err)
	}
	if win == nil || win.ID != initial {
		t.Fatalf("new window = %+v, want ID %d", win, initial)
	}
	if len(s.Windows) != initial+1 {
		t.Fatalf("window count = %d, want %d", len(s.Windows), initial+1)
	}
	if s.CurrentWindow != initial {
		t.Errorf("CurrentWindow = %d, want %d (auto-switch to new window)", s.CurrentWindow, initial)
	}

	// Switching back and forth updates the cursor.
	if err := s.SwitchToWindow("0"); err != nil {
		t.Fatalf("SwitchToWindow(0): %v", err)
	}
	if s.CurrentWindow != 0 || s.LastWindow != initial {
		t.Errorf("switch state wrong: current=%d last=%d", s.CurrentWindow, s.LastWindow)
	}
	if err := s.SwitchToWindow("99"); err == nil {
		t.Error("SwitchToWindow(99) should fail")
	}
}

func TestWindowNavigation(t *testing.T) {
	s := newSession(t, "navigate")

	for i := 0; i < 2; i++ {
		if _, err := s.CreateWindow("/bin/sleep", []string{"30"}, nil); err != nil {
			t.Fatalf("CreateWindow %d: %v", i, err)
		}
	}
	if got := len(s.Windows); got != 3 {
		t.Fatalf("windows = %d, want 3", got)
	}

	s.NextWindow()
	if s.CurrentWindow != 0 {
		t.Errorf("NextWindow from 2 should wrap to 0, got %d", s.CurrentWindow)
	}
	s.PrevWindow()
	if s.CurrentWindow != 2 {
		t.Errorf("PrevWindow from 0 should wrap to 2, got %d", s.CurrentWindow)
	}
	s.ToggleLastWindow()
	if s.CurrentWindow != 0 {
		t.Errorf("ToggleLastWindow should return to previous window, got %d", s.CurrentWindow)
	}
}

func TestWindowKillKeepsSessionAlive(t *testing.T) {
	s := newSession(t, "window-kill")

	if _, err := s.CreateWindow("/bin/sleep", []string{"30"}, nil); err != nil {
		t.Fatalf("CreateWindow: %v", err)
	}

	win := s.GetCurrentWindow()
	if err := win.Kill(); err != nil {
		t.Logf("Kill: %v (non-fatal, process may have exited already)", err)
	}
	if len(s.Windows) != 2 {
		t.Errorf("killed window must stay listed until removed, count = %d", len(s.Windows))
	}
	if !strings.Contains(session.FilePath(s.ID), ".sgreen") {
		t.Error("session file path sanity check failed")
	}
}
