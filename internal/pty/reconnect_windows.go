//go:build windows
// +build windows

package pty

import (
	"fmt"
	"os"
)

// Reconnect opens an existing PTY by its path (Windows version)
// Note: Windows doesn't support Unix-style PTY paths, so this is a no-op
func Reconnect(ptsPath string) (*PTYProcess, error) {
	// Windows doesn't support reconnecting to PTYs by path
	// Return an error indicating this is not supported
	return nil, fmt.Errorf("PTY reconnection is not supported on Windows: %w", os.ErrNotExist)
}

