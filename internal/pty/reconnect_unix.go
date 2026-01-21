//go:build !windows
// +build !windows

package pty

import (
	"os"
)

// Reconnect opens an existing PTY by its path (Unix version)
func Reconnect(ptsPath string) (*PTYProcess, error) {
	ptyFile, err := os.OpenFile(ptsPath, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	return &PTYProcess{
		Cmd:     nil, // No command reference when reconnecting
		Pty:     ptyFile,
		PtsPath: ptsPath,
	}, nil
}
