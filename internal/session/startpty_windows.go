//go:build windows
// +build windows

package session

import "github.com/inoki/sgreen/internal/pty"

// startSessionPTY returns no process on Windows: file-descriptor
// inheritance does not exist there, so the session daemon forked right
// after this starts the program itself (bootstrap mode).
func startSessionPTY(cmdPath string, args []string, envOverrides map[string]string) (*pty.PTYProcess, error) {
	return nil, nil
}
