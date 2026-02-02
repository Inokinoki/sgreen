//go:build !windows
// +build !windows

package main

import (
	"os/exec"
	"syscall"
)

func setDetachSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

