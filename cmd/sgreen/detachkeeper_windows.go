//go:build windows
// +build windows

package main

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// setDetachSysProcAttr detaches the session daemon from our console: it
// must survive the creating terminal closing, and it never touches the
// console itself.
func setDetachSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
	}
}
