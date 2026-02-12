// Package behavior runs CLI behavior tests against the sgreen binary.
// These tests can be run on all platforms (Linux, macOS, Windows, FreeBSD)
// and verify exit codes and output without requiring a TTY.
//
// Session state is isolated by setting HOME to a temporary directory so
// ~/.sgreen/sessions/ does not affect the user's real sessions.
package behavior

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// sgreenCmd returns the exec.Cmd to run sgreen with the given args.
// Use SGREEN_BINARY to point to a built binary (e.g. ./build/sgreen).
// Otherwise uses "go run ./cmd/sgreen" so no prior build is required.
func sgreenCmd(tb testing.TB, args []string) *exec.Cmd {
	tb.Helper()
	_, filename, _, _ := runtime.Caller(0)
	pkgDir := filepath.Dir(filename)
	modRoot := filepath.Join(pkgDir, "..", "..")

	if bin := os.Getenv("SGREEN_BINARY"); bin != "" {
		cmd := exec.Command(bin, args...)
		cmd.Dir = modRoot
		return cmd
	}
	if st, err := os.Stat(filepath.Join(modRoot, "build", "sgreen")); err == nil && !st.IsDir() {
		cmd := exec.Command(filepath.Join(modRoot, "build", "sgreen"), args...)
		cmd.Dir = modRoot
		return cmd
	}
	cmd := exec.Command("go", append([]string{"run", "./cmd/sgreen"}, args...)...)
	cmd.Dir = modRoot
	return cmd
}

// runSgreen runs sgreen with the given args and env. HOME is set to a temp dir
// so session state is isolated. Returns combined stdout+stderr and exit code.
func runSgreen(tb testing.TB, args []string, extraEnv map[string]string) (output string, exitCode int) {
	tb.Helper()
	cmd := sgreenCmd(tb, args)

	homeDir := tb.TempDir()
	env := os.Environ()
	env = setEnv(env, "HOME", homeDir)
	for k, v := range extraEnv {
		env = setEnv(env, k, v)
	}
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	output = string(out)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return output, exitCode
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func writeSessionFile(tb testing.TB, homeDir, id string, pid int) {
	tb.Helper()
	sessionsDir := filepath.Join(homeDir, ".sgreen", "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		tb.Fatalf("mkdir sessions dir: %v", err)
	}
	data := []byte(fmt.Sprintf(`{"id":%q,"pid":%d}`, id, pid))
	path := filepath.Join(sessionsDir, id+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		tb.Fatalf("write session file: %v", err)
	}
}

// --- B1.* Non-interactive CLI tests (no session, no TTY) ---

func TestVersion(t *testing.T) {
	out, code := runSgreen(t, []string{"-v"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -v: exit code %d, want 1 (GNU screen style)\n%s", code, out)
	}
	if !strings.Contains(out, "sgreen") || !strings.Contains(out, "version") {
		t.Fatalf("sgreen -v: output should contain 'sgreen' and 'version'\n%s", out)
	}
}

func TestHelpShort(t *testing.T) {
	out, code := runSgreen(t, []string{"-help"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -help: exit code %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "sgreen") {
		t.Fatalf("sgreen -help: output should contain 'Usage:' and 'sgreen'\n%s", out)
	}
}

func TestHelpLong(t *testing.T) {
	out, code := runSgreen(t, []string{"-help"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -help: exit code %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("sgreen -help: output should contain 'Usage:'\n%s", out)
	}
}

func TestListNoSessions(t *testing.T) {
	out, code := runSgreen(t, []string{"-ls"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -ls: exit code %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "No Sockets") {
		t.Fatalf("sgreen -ls with no sessions: expected 'No Sockets' message\n%s", out)
	}
}

func TestListAlternativeFlag(t *testing.T) {
	out, code := runSgreen(t, []string{"-list"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -list: exit code %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "No Sockets") {
		t.Fatalf("sgreen -list with no sessions: expected 'No Sockets' message\n%s", out)
	}
}

func TestReattachNoSessions(t *testing.T) {
	out, code := runSgreen(t, []string{"-r"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -r: exit code 0, want non-zero when no sessions\n%s", out)
	}
	if !strings.Contains(out, "No screen session found") && !strings.Contains(out, "No screen session") {
		t.Fatalf("sgreen -r: stderr should mention no session found\n%s", out)
	}
}

func TestReattachMissingName(t *testing.T) {
	out, code := runSgreen(t, []string{"-r", "nosuchsession123"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -r nosuchsession123: exit code 0, want non-zero\n%s", out)
	}
	if !strings.Contains(out, "No screen session found") && !strings.Contains(out, "nosuchsession123") {
		t.Fatalf("sgreen -r nosuchsession123: stderr should mention no session or name\n%s", out)
	}
}

func TestWipeNoSessions(t *testing.T) {
	out, code := runSgreen(t, []string{"-wipe"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -wipe: exit code %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "No Sockets") {
		t.Fatalf("sgreen -wipe with no sessions: expected no-sockets message\n%s", out)
	}
}

func TestDetachNoSessions(t *testing.T) {
	out, code := runSgreen(t, []string{"-d"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -d: exit code 0, want non-zero when no sessions\n%s", out)
	}
	if !strings.Contains(out, "No screen session found") && !strings.Contains(out, "No attached") {
		t.Fatalf("sgreen -d: stderr should mention no session\n%s", out)
	}
}

func TestPowerDetachNoSessions(t *testing.T) {
	out, code := runSgreen(t, []string{"-D"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -D: exit code 0, want non-zero when no sessions\n%s", out)
	}
	if !strings.Contains(out, "No screen session found") {
		t.Fatalf("sgreen -D: stderr should mention no session found\n%s", out)
	}
}

func TestSendCommandNoSessions(t *testing.T) {
	out, code := runSgreen(t, []string{"-X", "stuff", "x"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -X stuff x: exit code 0, want non-zero when no sessions\n%s", out)
	}
	if !strings.Contains(out, "No screen session found") {
		t.Fatalf("sgreen -X: stderr should mention no session found\n%s", out)
	}
}

func TestUnknownFlag(t *testing.T) {
	out, code := runSgreen(t, []string{"-unknown"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -unknown: exit code 0, want non-zero\n%s", out)
	}
	// Should mention flag or usage
	if out == "" {
		t.Fatalf("sgreen -unknown: expected some stderr output")
	}
}

// --- B3.* Flags and config ---

func TestQuiet(t *testing.T) {
	out, code := runSgreen(t, []string{"-q", "-ls"}, nil)
	if code != 8 {
		t.Fatalf("sgreen -q -ls: exit code %d, want 8 (GNU screen quiet no-sessions)\n%s", code, out)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("sgreen -q -ls: expected no output, got %q", out)
	}
}

func TestIgnoreSTY(t *testing.T) {
	// With STY set, -m should still allow -ls to run (ignore STY for attach)
	out, code := runSgreen(t, []string{"-m", "-ls"}, map[string]string{"STY": "12345.pts-0.host"})
	if code != 1 {
		t.Fatalf("sgreen -m -ls: exit code %d, want 1\n%s", code, out)
	}
}

func TestVersionThreeLines(t *testing.T) {
	out, code := runSgreen(t, []string{"-v"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -v: exit code %d, want 1\n%s", code, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("sgreen -v: expected at least 2 lines of version output, got %d\n%s", len(lines), out)
	}
}

// --- Additional non-interactive behavior tests ---

func TestVersionContainsVersionNumber(t *testing.T) {
	out, code := runSgreen(t, []string{"-v"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -v: exit code %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "0.1.0") {
		t.Fatalf("sgreen -v: output should contain version 0.1.0\n%s", out)
	}
}

func TestVersionExactlyThreeLines(t *testing.T) {
	out, code := runSgreen(t, []string{"-v"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -v: exit code %d, want 1\n%s", code, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("sgreen -v: expected exactly 3 lines, got %d\n%s", len(lines), out)
	}
}

func TestHelpContainsKeyOptions(t *testing.T) {
	out, code := runSgreen(t, []string{"-help"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -help: exit code %d, want 1\n%s", code, out)
	}
	for _, sub := range []string{"-r", "-R", "-ls", "-d", "-D", "-S"} {
		if !strings.Contains(out, sub) {
			t.Fatalf("sgreen -h: output should contain %q\n%s", sub, out)
		}
	}
}

func TestHelpMentionsDetach(t *testing.T) {
	out, code := runSgreen(t, []string{"-help"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -help: exit code %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "detach") && !strings.Contains(out, "Detach") &&
		!strings.Contains(out, "Ctrl+A") && !strings.Contains(out, "C-a") {
		t.Fatalf("sgreen -h: output should mention detach or Ctrl+A\n%s", out)
	}
}

func TestErrorExitCodeOne(t *testing.T) {
	tests := []struct {
		args []string
		name string
	}{
		{[]string{"-r"}, "reattach no sessions"},
		{[]string{"-r", "nosuch"}, "reattach wrong name"},
		{[]string{"-d"}, "detach no sessions"},
		{[]string{"-D"}, "power detach no sessions"},
		{[]string{"-X", "stuff", "x"}, "send command no sessions"},
	}
	for _, tt := range tests {
		_, code := runSgreen(t, tt.args, nil)
		if code != 1 {
			t.Errorf("%s: sgreen %v: exit code %d, want 1", tt.name, tt.args, code)
		}
	}
}

func TestSessionNameWithList(t *testing.T) {
	// -S name with -ls does not create a session; just lists (empty).
	out, code := runSgreen(t, []string{"-S", "myname", "-ls"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -S myname -ls: exit code %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "No ") && !strings.Contains(out, "no ") {
		t.Fatalf("sgreen -S myname -ls: expected no-sessions message\n%s", out)
	}
}

func TestConfigNonexistent(t *testing.T) {
	// -c with nonexistent config should still run -ls (config optional).
	out, code := runSgreen(t, []string{"-c", "/nonexistent/screenrc", "-ls"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -c /nonexistent -ls: exit code %d, want 1\n%s", code, out)
	}
	// May contain "No " (no sessions) or a config warning; either is acceptable.
}

func TestWipeExactMessage(t *testing.T) {
	out, code := runSgreen(t, []string{"-wipe"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -wipe: exit code %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "No Sockets found in ") {
		t.Fatalf("sgreen -wipe with no sessions: output should contain screen-style no-sockets message\n%s", out)
	}
}

func TestReattachErrorContainsSessionName(t *testing.T) {
	out, code := runSgreen(t, []string{"-r", "wrongname"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -r wrongname: exit code 0, want non-zero\n%s", out)
	}
	if !strings.Contains(out, "No screen session found") {
		t.Fatalf("sgreen -r wrongname: stderr should contain 'No screen session found'\n%s", out)
	}
	// When there are no sessions, message may be "No screen session found.";
	// when there are sessions but not this one, message may include "wrongname".
}

func TestMultiuserNoSessions(t *testing.T) {
	// -x (multiuser attach) with no sessions should fail.
	out, code := runSgreen(t, []string{"-x"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -x: exit code 0, want non-zero when no sessions\n%s", out)
	}
	if !strings.Contains(out, "No screen session found") && !strings.Contains(out, "Multiple sessions") {
		t.Fatalf("sgreen -x: expected 'No screen session found' or 'Multiple sessions'\n%s", out)
	}
}

func TestSendCommandNoSessionWithName(t *testing.T) {
	out, code := runSgreen(t, []string{"-X", "stuff", "x", "-S", "foo"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -X stuff x -S foo: exit code 0, want non-zero\n%s", out)
	}
	if !strings.Contains(out, "No screen session found") {
		t.Fatalf("sgreen -X -S foo: stderr should mention no session found\n%s", out)
	}
	// Session name "foo" may appear in error when implementation includes it.
}

func TestListExactMessage(t *testing.T) {
	out, code := runSgreen(t, []string{"-ls"}, nil)
	if code != 1 {
		t.Fatalf("sgreen -ls: exit code %d, want 1\n%s", code, out)
	}
	// Screen-compatible message when no sessions.
	if !strings.Contains(out, "No Sockets") && !strings.Contains(out, "No screen") {
		t.Fatalf("sgreen -ls with no sessions: expected 'No Sockets' or 'No screen'\n%s", out)
	}
}

func TestListAndListAlternativeSameBehavior(t *testing.T) {
	outLs, codeLs := runSgreen(t, []string{"-ls"}, nil)
	outList, codeList := runSgreen(t, []string{"-list"}, nil)
	if codeLs != 1 || codeList != 1 {
		t.Fatalf("both -ls and -list should exit 1 with no sessions: -ls=%d -list=%d", codeLs, codeList)
	}
	// Both should report no sessions (same kind of message).
	hasNoLs := strings.Contains(outLs, "No ") || strings.Contains(outLs, "no ")
	hasNoList := strings.Contains(outList, "No ") || strings.Contains(outList, "no ")
	if !hasNoLs || !hasNoList {
		t.Fatalf("-ls and -list should both show no-sessions message\n-ls: %q\n-list: %q", outLs, outList)
	}
}

func TestReattachNoSessionExactMessage(t *testing.T) {
	out, code := runSgreen(t, []string{"-r"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -r: exit code 0, want non-zero\n%s", out)
	}
	if !strings.Contains(out, "No screen session found.") && !strings.Contains(out, "No screen session found") {
		t.Fatalf("sgreen -r: stderr should contain 'No screen session found'\n%s", out)
	}
}

func TestUnknownFlagProducesOutput(t *testing.T) {
	out, code := runSgreen(t, []string{"-unknown"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -unknown: exit code 0, want non-zero\n%s", out)
	}
	if len(strings.TrimSpace(out)) == 0 {
		t.Fatalf("sgreen -unknown: expected usage or error message on stderr\n%q", out)
	}
}

func TestReattachMissingNameAlwaysMentionsRequestedSession(t *testing.T) {
	out, code := runSgreen(t, []string{"-r", "nosuchsession123"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -r nosuchsession123: exit code 0, want non-zero\n%s", out)
	}
	if !strings.Contains(out, "nosuchsession123") {
		t.Fatalf("sgreen -r nosuchsession123: output should include requested session name\n%s", out)
	}
}

func TestListSingleSessionShowsScreenStyleSummary(t *testing.T) {
	homeDir := t.TempDir()
	writeSessionFile(t, homeDir, "demo", os.Getpid())

	out, code := runSgreen(t, []string{"-ls"}, map[string]string{"HOME": homeDir})
	if code != 0 {
		t.Fatalf("sgreen -ls with one synthetic alive pid session: exit code %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "There is a screen on:") {
		t.Fatalf("sgreen -ls with one session: expected 'There is a screen on:'\n%s", out)
	}
	if !strings.Contains(out, "(demo)") {
		t.Fatalf("sgreen -ls with one session: expected session name '(demo)'\n%s", out)
	}
	if !strings.Contains(out, "1 Socket in ") {
		t.Fatalf("sgreen -ls with one session: expected socket summary line\n%s", out)
	}
}

func TestDetachedCreateDmSParses(t *testing.T) {
	out, code := runSgreen(t, []string{"-dmS", "demo", "/bin/sh", "-c", "sleep 1"}, nil)
	if code != 0 {
		t.Fatalf("sgreen -dmS demo ...: exit code %d, want 0\n%s", code, out)
	}
	if strings.Contains(out, "flag provided but not defined") {
		t.Fatalf("sgreen -dmS should be parsed as detached create, got parse error\n%s", out)
	}
}

func TestShortHRequiresArgument(t *testing.T) {
	out, code := runSgreen(t, []string{"-h"}, nil)
	if code == 0 {
		t.Fatalf("sgreen -h: exit code 0, want non-zero because -h expects scrollback value\n%s", out)
	}
	if !strings.Contains(out, "flag needs an argument") {
		t.Fatalf("sgreen -h: expected missing-argument error\n%s", out)
	}
}
