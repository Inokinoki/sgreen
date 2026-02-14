//go:build !windows
// +build !windows

package pty

import (
	"path/filepath"
	"strings"
)

// wrapCommandForDetach ensures interactive shells ignore SIGHUP on Unix.
// This helps foreground jobs survive detach even if the PTY master closes.
func wrapCommandForDetach(cmdPath string, args []string) (string, []string) {
	base := strings.ToLower(filepath.Base(cmdPath))
	switch base {
	case "zsh", "bash", "sh", "ksh", "fish":
		if len(args) == 0 || containsInteractiveFlag(args) {
			wrapped := append([]string{cmdPath}, args...)
			return "nohup", wrapped
		}
	}
	return cmdPath, args
}

func containsInteractiveFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-i" || arg == "--interactive" {
			return true
		}
	}
	return false
}
