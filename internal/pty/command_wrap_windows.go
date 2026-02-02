//go:build windows
// +build windows

package pty

func wrapCommandForDetach(cmdPath string, args []string) (string, []string) {
	return cmdPath, args
}

