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
	win := &session.Window{Pid: os.Getpid()}
	sess.Windows = []*session.Window{win}

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

func TestResolvePowerDetachTarget(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		args      []string
		want      string
	}{
		{
			name:      "explicit flag wins",
			flagValue: "named",
			args:      []string{"other", "arg"},
			want:      "named",
		},
		{
			name: "single positional is target",
			args: []string{"target"},
			want: "target",
		},
		{
			name: "multiple positionals are not treated as named target",
			args: []string{"target", "/bin/sh", "-c", "echo hi"},
			want: "",
		},
		{
			name: "no args",
			args: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		if got := resolvePowerDetachTarget(tt.flagValue, tt.args); got != tt.want {
			t.Fatalf("%s: resolvePowerDetachTarget(%q, %q) = %q, want %q", tt.name, tt.flagValue, tt.args, got, tt.want)
		}
	}
}

func TestIsOwnerSessionTarget(t *testing.T) {
	t.Setenv("USER", "alice")
	t.Setenv("USERNAME", "")

	tests := []struct {
		name string
		arg  string
		want bool
	}{
		{name: "empty", arg: "", want: false},
		{name: "plain name", arg: "demo", want: false},
		{name: "self owner session", arg: "alice/123.pts.host", want: false},
		{name: "other owner session", arg: "bob/123.pts.host", want: true},
		{name: "slash without owner", arg: "/123.pts.host", want: false},
		{name: "slash without session", arg: "bob/", want: false},
	}

	for _, tt := range tests {
		if got := isOwnerSessionTarget(tt.arg); got != tt.want {
			t.Fatalf("%s: isOwnerSessionTarget(%q) = %v, want %v", tt.name, tt.arg, got, tt.want)
		}
	}
}

func TestParseCommandChar(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  byte
	}{
		{"empty string", "", 0x01},
		{"caret notation uppercase", "^A", 0x01},
		{"caret notation lowercase", "^a", 0x01},
		{"caret notation middle", "^M", 0x0D},
		{"hex notation", "\\x01", 0x01},
		{"hex notation custom", "\\x1b", 0x1b},
		{"single character", "a", 'a'},
		{"invalid caret", "^@", 0x01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommandChar(tt.input)
			if got != tt.want {
				t.Errorf("parseCommandChar(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveSessionName(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		args     []string
		expected string
	}{
		{"flag takes precedence", "mysession", []string{"other"}, "mysession"},
		{"args when no flag", "", []string{"arg_session"}, "arg_session"},
		{"no flag no args", "", []string{}, ""},
		{"args ignored with flag", "flag_session", []string{"arg1", "arg2"}, "flag_session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSessionName(tt.flag, tt.args)
			if got != tt.expected {
				t.Errorf("resolveSessionName(%q, %v) = %q, want %q", tt.flag, tt.args, got, tt.expected)
			}
		})
	}
}

func TestNormalizeArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"no special args", []string{"-r", "-S", "test"}, []string{"-r", "-S", "test"}},
		{"dmS flag", []string{"-dmS", "test"}, []string{"-d", "-m", "-S", "test"}},
		{"dmS combined", []string{"-dmSmysession"}, []string{"-d", "-m", "-S", "mysession"}},
		{"dm flag", []string{"-dm"}, []string{"-d", "-m"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("normalizeArgs(%v) len = %d, want %d", tt.input, len(got), len(tt.expected))
			}
			for i := range tt.expected {
				if i >= len(got) || got[i] != tt.expected[i] {
					break
				}
			}
		})
	}
}

func TestEnsureInteractiveShellArgs(t *testing.T) {
	tests := []struct {
		name     string
		cmdPath  string
		args     []string
		expected []string
	}{
		{"zsh with no args", "/bin/zsh", nil, []string{"-i"}},
		{"bash with no args", "/bin/bash", nil, []string{"-i"}},
		{"sh with no args", "/bin/sh", nil, []string{"-i"}},
		{"ksh with no args", "/bin/ksh", nil, []string{"-i"}},
		{"fish with no args", "/usr/bin/fish", nil, []string{"-i"}},
		{"custom shell with no args", "/bin/custom", nil, nil},
		{"shell with args", "/bin/bash", []string{"-c", "echo hi"}, []string{"-c", "echo hi"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureInteractiveShellArgs(tt.cmdPath, tt.args)
			if len(got) != len(tt.expected) {
				t.Errorf("ensureInteractiveShellArgs(%q, %v) len = %d, want %d", tt.cmdPath, tt.args, len(got), len(tt.expected))
			}
			for i := range tt.expected {
				if i >= len(got) || got[i] != tt.expected[i] {
					break
				}
			}
		})
	}
}

func TestSocketWord(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{1, "Socket"},
		{2, "Sockets"},
		{0, "Sockets"},
		{10, "Sockets"},
	}

	for _, tt := range tests {
		got := socketWord(tt.count)
		if got != tt.expected {
			t.Errorf("socketWord(%d) = %q, want %q", tt.count, got, tt.expected)
		}
	}
}

func TestRequiresTerminalForOperation(t *testing.T) {
	tests := []struct {
		name               string
		reattach           bool
		reattachOrCreate   bool
		reattachOrCreateRR bool
		multiuser          bool
		detach             bool
		expected           bool
	}{
		{"all false", false, false, false, false, false, false},
		{"reattach only", true, false, false, false, false, true},
		{"reattachOrCreate only", false, true, false, false, false, true},
		{"reattachOrCreateRR only", false, false, true, false, false, true},
		{"multiuser only", false, false, false, true, false, true},
		{"detach only", false, false, false, false, true, false},
		{"detach and reattach", true, false, false, false, true, true},
		{"all true", true, true, true, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requiresTerminalForOperation(tt.reattach, tt.reattachOrCreate, tt.reattachOrCreateRR, tt.multiuser, tt.detach)
			if got != tt.expected {
				t.Errorf("requiresTerminalForOperation(%v, %v, %v, %v, %v) = %v, want %v",
					tt.reattach, tt.reattachOrCreate, tt.reattachOrCreateRR, tt.multiuser, tt.detach, got, tt.expected)
			}
		})
	}
}

func TestIsNoResumableError(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		expected bool
	}{
		{"no resumable error", "There is no screen to be resumed", true},
		{"no resumable with name", "There is no screen to be resumed matching demo", true},
		{"no resumable with period", "There is no screen to be resumed.", true},
		{"different error", "Permission denied", false},
		{"empty string", "", false},
		{"partial match", "There is no screen", false},
		{"case sensitive", "there is no screen to be resumed", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNoResumableError(tt.msg)
			if got != tt.expected {
				t.Errorf("isNoResumableError(%q) = %v, want %v", tt.msg, got, tt.expected)
			}
		})
	}
}

func TestNoResumableScreenMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty name", "", "There is no screen to be resumed."},
		{"with name", "demo", "There is no screen to be resumed matching demo."},
		{"with spaces", "my session", "There is no screen to be resumed matching my session."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noResumableScreenMessage(tt.input)
			if got != tt.expected {
				t.Errorf("noResumableScreenMessage(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNoAttachableScreenMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty name", "", "There is no screen to be attached."},
		{"with name", "demo", "There is no screen to be attached matching demo."},
		{"with spaces", "my session", "There is no screen to be attached matching my session."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noAttachableScreenMessage(tt.input)
			if got != tt.expected {
				t.Errorf("noAttachableScreenMessage(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNoDetachableScreenMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty name", "", "There is no screen to be detached."},
		{"with name", "demo", "There is no screen to be detached matching demo."},
		{"with spaces", "my session", "There is no screen to be detached matching my session."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noDetachableScreenMessage(tt.input)
			if got != tt.expected {
				t.Errorf("noDetachableScreenMessage(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
