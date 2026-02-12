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
	if errMsg != "No screen session found: demo" {
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
	if errMsg != "There is no screen to be resumed (the only session is attached)." {
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
