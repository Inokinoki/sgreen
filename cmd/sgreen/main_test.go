package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inoki/sgreen/internal/session"
)

func TestSelectReattachSession_NoSessionsWithName(t *testing.T) {
	_, errMsg, printList := selectReattachSession(nil, "demo", false, nil, nil)
	if errMsg != "There is no screen to be resumed matching demo." {
		t.Fatalf("unexpected error message: %q", errMsg)
	}
	if printList {
		t.Fatalf("printList = true, want false")
	}
}

func TestSelectReattachSession_NamedAttachedRequiresForce(t *testing.T) {
	sess := &session.Session{ID: "demo", Pid: os.Getpid()}
	loadByName := func(name string) (*session.Session, error) {
		if name != "demo" {
			return nil, errors.New("missing")
		}
		return sess, nil
	}
	isAttached := func(s *session.Session) bool {
		return s == sess
	}

	_, errMsg, printList := selectReattachSession([]*session.Session{sess}, "demo", false, loadByName, isAttached)
	if !strings.Contains(errMsg, "is attached") {
		t.Fatalf("unexpected error message: %q", errMsg)
	}
	if printList {
		t.Fatalf("printList = true, want false")
	}

	selected, errMsg, printList := selectReattachSession([]*session.Session{sess}, "demo", true, loadByName, isAttached)
	if errMsg != "" || printList {
		t.Fatalf("unexpected error selecting with -x: err=%q printList=%v", errMsg, printList)
	}
	if selected != sess {
		t.Fatalf("selected session mismatch")
	}
}

func TestSelectReattachSession_UnnamedAttachedOnly(t *testing.T) {
	sess := &session.Session{ID: "only", Pid: os.Getpid()}
	isAttached := func(s *session.Session) bool { return s == sess }

	selected, errMsg, printList := selectReattachSession(
		[]*session.Session{sess},
		"",
		false,
		func(string) (*session.Session, error) { return nil, errors.New("unused") },
		isAttached,
	)
	if selected != nil {
		t.Fatalf("expected no selected session")
	}
	if errMsg != "There is no screen to be resumed." {
		t.Fatalf("unexpected error message: %q", errMsg)
	}
	if printList {
		t.Fatalf("printList = true, want false")
	}
}

func TestSelectReattachSession_MultiuserUnnamedMultiple(t *testing.T) {
	s1 := &session.Session{ID: "one", Pid: os.Getpid()}
	s2 := &session.Session{ID: "two", Pid: os.Getpid()}

	selected, errMsg, printList := selectReattachSession(
		[]*session.Session{s1, s2},
		"",
		true,
		func(string) (*session.Session, error) { return nil, errors.New("unused") },
		func(*session.Session) bool { return false },
	)
	if selected != nil {
		t.Fatalf("expected no selected session")
	}
	if errMsg != "Multiple sessions found. Specify session name with -x:" {
		t.Fatalf("unexpected error message: %q", errMsg)
	}
	if !printList {
		t.Fatalf("printList = false, want true")
	}
}

func TestSelectReattachSession_OneDetached(t *testing.T) {
	sess := &session.Session{ID: "detached", Pid: os.Getpid()}

	selected, errMsg, printList := selectReattachSession(
		[]*session.Session{sess},
		"",
		false,
		func(string) (*session.Session, error) { return nil, errors.New("unused") },
		func(*session.Session) bool { return false },
	)
	if errMsg != "" || printList {
		t.Fatalf("unexpected result: err=%q printList=%v", errMsg, printList)
	}
	if selected != sess {
		t.Fatalf("selected session mismatch")
	}
}

func TestScreenSocketDirForDisplay(t *testing.T) {
	t.Setenv("USER", "alice")
	t.Setenv("USERNAME", "")
	got := screenSocketDirForDisplay()
	want := filepath.Join(os.TempDir(), "screens", "S-alice")
	if got != want {
		t.Fatalf("screenSocketDirForDisplay() = %q, want %q", got, want)
	}
}

func TestDefaultSessionNameUsesValidCharacters(t *testing.T) {
	name := defaultSessionName()
	if name == "" {
		t.Fatalf("defaultSessionName() returned empty name")
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' {
			continue
		}
		t.Fatalf("defaultSessionName() contains invalid char %q in %q", r, name)
	}

	parts := strings.Split(name, ".")
	if len(parts) < 3 {
		t.Fatalf("defaultSessionName() = %q, want at least <pid>.<tty>.<host>", name)
	}
	if parts[0] == "" {
		t.Fatalf("defaultSessionName() missing pid component: %q", name)
	}
	host := parts[len(parts)-1]
	if host == "" {
		t.Fatalf("defaultSessionName() missing host component: %q", name)
	}
}

func TestResolveSessionAndCommandArgs(t *testing.T) {
	tests := []struct {
		name        string
		flagValue   string
		args        []string
		wantSession string
		wantArgs    []string
	}{
		{
			name:        "flag session keeps full args as command",
			flagValue:   "named",
			args:        []string{"/bin/sh", "-c", "echo ok"},
			wantSession: "named",
			wantArgs:    []string{"/bin/sh", "-c", "echo ok"},
		},
		{
			name:        "positional first arg is session",
			args:        []string{"target", "/bin/echo", "hello"},
			wantSession: "target",
			wantArgs:    []string{"/bin/echo", "hello"},
		},
		{
			name:        "no args returns empty session",
			args:        nil,
			wantSession: "",
			wantArgs:    nil,
		},
	}

	for _, tt := range tests {
		gotSession, gotArgs := resolveSessionAndCommandArgs(tt.flagValue, tt.args)
		if gotSession != tt.wantSession {
			t.Fatalf("%s: session=%q, want %q", tt.name, gotSession, tt.wantSession)
		}
		if strings.Join(gotArgs, "\x00") != strings.Join(tt.wantArgs, "\x00") {
			t.Fatalf("%s: args=%q, want %q", tt.name, gotArgs, tt.wantArgs)
		}
	}
}
