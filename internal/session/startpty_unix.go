//go:build !windows
// +build !windows

package session

import "github.com/inoki/sgreen/internal/pty"

// startSessionPTY launches the session program on a PTY in this process.
func startSessionPTY(cmdPath string, args []string, envOverrides map[string]string) (*pty.PTYProcess, error) {
	return pty.StartWithEnv(cmdPath, args, envOverrides)
}
