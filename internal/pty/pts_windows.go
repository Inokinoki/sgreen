//go:build windows
// +build windows

package pty

import (
	"os"
)

// getPtsPathViaIoctl is not supported on Windows
func getPtsPathViaIoctl(ptyFile *os.File) (string, error) {
	// Windows doesn't support Unix-style PTY paths
	// Return empty string to indicate not available
	return "", os.ErrNotExist
}
